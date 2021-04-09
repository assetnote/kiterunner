package http

import (
	"bytes"
	"time"

	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/valyala/fasthttp"
)

var (
	strLocation = []byte(fasthttp.HeaderLocation)
)

// HTTPClient is a type alias for the actual host client we use.
// We do this instead of using an interface to avoid reflecting
type HTTPClient = fasthttp.HostClient

// BackupClient is a normal fasthttpClient that can adapt to different hosts
// This is used to handle redirects that change the host/port of the request
type BackupClient = fasthttp.Client

// NewHTTPClient will create a http client configured specifically for requesting against the targetted host.
// This is backed by the fasthttp.HostClient
func NewHTTPClient(host string, tls bool) *HTTPClient {
	return &HTTPClient{
		Addr:      host,
		IsTLS:     tls,
		TLSConfig: defaultTLSConfig,
	}
}

// Config provides all the options available to a request, this is used by DoClient
type Config struct {
	// Timeout is the duration to wait when performing a DoTimeout request
	Timeout      time.Duration `toml:"timeout" json:"timeout" mapstructure:"timeout"`
	// MaxRedirects corresponds to how many redirects to follow. 0 means the first request will return and no
	// redirects are followed
	MaxRedirects int           `toml:"max_redirects" json:"max_redirects" mapstructure:"max_redirects"`

	// ReadBody defines whether to copy the body into the Response object. We will always read the body of the request
	// off the wire to perform the length and word count calculations
	ReadBody    bool
	// ReadHeaders defines whether to copy the headers into the Response object. We will always peek the location header
	// to determine whether to follow redirects
	ReadHeaders bool
	// BlacklistRedirects is a slice of strings that are substring matched against location headers.
	// if a string is blacklisted, e.g. okta.com, the redirect is not followed
	BlacklistRedirects []string `toml:"blacklist_redirects" json:"blacklist_redirects" mapstructure:"blacklist_redirects"`

	// ExtraHeaders are added to the request last and will overwrite the route headers
	ExtraHeaders []Header

	backupClient *BackupClient
}

// IsBlacklistedRedirect will compare the stored hosts against the provided host.
// We use a prefix match with linear probing to simplify the check.
// e.g. blacklist[okta.com, onelogin.com] will match against okta.com:80
func (c *Config) IsBlacklistedRedirect(host []byte) bool {
	log.Trace().Str("host", string(host)).Strs("blhosts", c.BlacklistRedirects).Msg("checking for blacklist hosts")
	for _, v := range c.BlacklistRedirects {
		if bytes.HasPrefix(host, []byte( v )) {
			return true
		}
	}
	return false
}

// BackupClient provides a generic http client that is not bound to a single host.
// it is generated on demand based off the config options. This is done only once per config.
// if you change the options after calling BackupClient(), you must call ResetBackupClient()
func (c *Config) BackupClient() *BackupClient {
	if c.backupClient == nil {
		c.backupClient = &BackupClient{
			ReadTimeout:              c.Timeout,
			WriteTimeout:             c.Timeout,
			TLSConfig:                defaultTLSConfig,
			NoDefaultUserAgentHeader: true,
		}
	}
	return c.backupClient
}

// ResetBackupClient will update the settings on the BackupClient. This is to be called if the timeout
// is changed on the config after calling BackupClient
func (c *Config) ResetBackupClient() {
	if c.backupClient != nil {
		c.backupClient.ReadTimeout = c.Timeout
		c.backupClient.WriteTimeout = c.Timeout
	}
}

// DoClient performs the provided request. We recommend avoiding letting the Response escape to the heap
// to prevent allocating where not necessary. This will handle the redirects for the request.
// Redirect responses are added to the linked list in the Response
// The returned response chain will have the Response.OriginRequest populated
// This will always read the body from the wire, but will only copy the body into the Response if config.ReadBody is true
// We will always perform the calculations for the BodyLength, Words and Lines, as these require 0 allocations
// given the response body is already read into memory
//
// Responses part of the chain are allocated dynamically using AcquireResponse and should be appropriately released
// when the response is no longer needed
func DoClient(c *HTTPClient, req Request, config *Config) (Response, error) {
	// the timeout should be set on the client already
	var (
		freq  = fasthttp.AcquireRequest()
		fresp = fasthttp.AcquireResponse()
	)
	req.WriteRequest(freq, nil)

	for _, h := range config.ExtraHeaders {
		freq.Header.Set(h.Key, h.Value)
	}

	fresp.Header.DisableNormalizing()

	// we still need to read the body to get counts for assessing the doc length
	if false && !config.ReadBody {
		fresp.SkipBody = true
	}

	ret, err := doRequestFollowRedirects(c, freq, fresp, config)
	fasthttp.ReleaseRequest(freq)
	fasthttp.ReleaseResponse(fresp)
	if err != nil &&
		// only abort if there's a timeout error
		// we do direct comparison here because i don't want to deal with errors.Is doing reflection and unwrapping
		err != fasthttp.ErrTooManyRedirects &&
		err != fasthttp.ErrMissingLocation {
		return ret, err
	}

	// populate our linked list
	ret.OriginRequest = req
	for r := ret.Next; r != nil; r = r.Next {
		r.OriginRequest = req
	}

	return ret, nil
}

// doRequestFollowRedirects will use the client provided and attempt to follow config.MaxRedirects number of redirects
// This will just exit early if it hits a redirect
// If a redirect is found, it builds out the response linked list
// We use a concrete HTTPClient instead of an interface to avoid reflection and saving us a few cycles
func doRequestFollowRedirects(c *HTTPClient, fastreq *fasthttp.Request, fastresp *fasthttp.Response, config *Config) (ret Response, err error) {
	redirectsCount := 0
	var url []byte
	backupClient := false

	resp := &ret
	for {
		// we don't need to modify the fastreq on the first iteration of the loop since it should be prepared already from the caller
		// subsequent iterations of the loop when handling redirects need to have fastreq updated

		// we don't need timeouts here because we rely on the client with the read and write timeout to timeout our request for us
		if backupClient {
			if err = config.BackupClient().Do(fastreq, fastresp); err != nil {
				return ret, err
			}
		} else {
			if err = c.Do(fastreq, fastresp); err != nil {
				return ret, err
			}
		}
		statusCode := fastresp.Header.StatusCode()

		// update our response with the results of the request
		resp.StatusCode = statusCode
		// the first URI will be empty. Yes i know. this is because the first request we know what the URI is (given the target and route)
		// we update the next response's URI based off the location header

		// we don't need to allocate to read this
		b := fastresp.Body()
		// this allocation might need to be disabled in release
		resp.BodyLength = len(b)
		resp.Words = bytes.Count(b, []byte(" "))
		resp.Lines = bytes.Count(b, []byte("\n"))
		// address off by 1 if its non-0
		if len(b) > 0 {
			resp.Words += 1
			resp.Lines += 1
		}

		if config.ReadBody {
			resp.Body = append(resp.Body[:0], b...)
		}
		// TODO: handle cookie updates here
		if config.ReadHeaders {
			fastresp.Header.VisitAll(func(k, v []byte) {
				resp.AddHeader(k, v)
			})
		}

		// log.Trace().Int("statuscode", statusCode).Msg("received status code")
		// Handle redirects if we need to keep going
		if !StatusCodeIsRedirect(statusCode) {
			break
		}

		redirectsCount++
		if redirectsCount > config.MaxRedirects {
			log.Trace().Msg("bailing out. reached max redirects")
			err = fasthttp.ErrTooManyRedirects
			break
		}

		// don't allocate the response if we can't walk that far. The user should realise they've limited the redirect count
		resp.Next = AcquireResponse()
		resp = resp.Next

		location := fastresp.Header.PeekBytes(strLocation)
		if len(location) == 0 {
			log.Trace().Msg("bailing out. reached missing location header")
			err = fasthttp.ErrMissingLocation
			// track this error so we can print it to the user later
			resp.Error = err
			break
		}

		// update the URI with the location header. This will show the user an accurate representation of
		// the redirect chain
		// in theory, instead of using a temp `url` variable, we can use this directly, but on same host
		// redirects, that would mean resp.URI is longer (since it'll contain the full https://host:port) instead
		// of just /redirectedpath
		resp.URI = append(resp.URI[:0], location...)

		// note this follows fasthttp.URI.UpdateBytes logic, where if its a relative path, we move to the relative path
		// /foo/bar 302 -> baz => /foo/baz with the new relative path. Not sure if this is what we want
		var samehost bool
		uri := fastreq.URI()
		samehost = updateRedirectURL(uri, url, location)

		// this is a single direction switch. once we move to the backup client, we can't go back
		// since we've moved off our original host, it doesnt make sense to use the hostclient anymore
		if !samehost {
			backupClient = true
		}

		if config.IsBlacklistedRedirect(uri.Host()) {
			break
		}

		log.Trace().
			Bytes("location", location).
			Msg("following redirect")
	}

	return ret, err
}

// getRedirectURL will construct the redirect URL based off the location header. This will also return if the
// redirect is on the same host or not.
func updateRedirectURL(base *fasthttp.URI, buf []byte, location []byte) bool {
	// preserve the old values to determine whether our scheme/host has changed
	var (
		host   = append([]byte{}, base.Host()...)
		scheme = append([]byte{}, base.Scheme()...)
	)
	base.UpdateBytes(location)
	// we need to compare the host (including port) and the scheme (protocol), otherwise we'll be trying a http
	// request against a https redirect
	sameHost := bytes.Equal(host, base.Host()) && bytes.Equal(scheme, base.Scheme())
	return sameHost
}

// StatusCodeIsRedirect returns true if the status code indicates a redirect.
func StatusCodeIsRedirect(statusCode int) bool {
	return statusCode == fasthttp.StatusMovedPermanently ||
		statusCode == fasthttp.StatusFound ||
		statusCode == fasthttp.StatusSeeOther ||
		statusCode == fasthttp.StatusTemporaryRedirect ||
		statusCode == fasthttp.StatusPermanentRedirect
}
