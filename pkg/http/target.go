package http

import (
	"context"
	"crypto/tls"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/valyala/bytebufferpool"
)

var (
	defaultTLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}

	bHTTPS = []byte("https")
	bHTTP  = []byte("http")
)

// Target encapsulates the basic data required to scan any given application. This can be specified to attach to a specific
// vhost or subpath configured webserver.
//
// When using target, you *MUST* call ParseHostHeader before calling AppendHostHeader to ensure the header is parsed.
// This developer unfriendly design decision was made to avoid having to create locks around generating the host header
// and to avoid allocating additional memory when requesting the host header. We recommend against changing the fields
// of a target after ParseHostHeader has been called as this will result in unexpected behaviour.
//
// The request is sent to the value returned by Host(). this is derived from the IP:port or Hostname:port.
//
// This target can be used to instantiate its own client using HTTPClient(maxconns, timeout).
//
// The Target should only be instantiated once for each use and is passed around via a pointer to ensure that the
// HTTPClient is reused.
//
// The Quarantine and Context features of a target can be used indicate to other readers across goroutines that the
// target is responding in an unexpected behaviour (i.e. responding to too many requests unexpected) and/or the
// target should be abandoned.
//
// A sync.Pool is provided for minimising allocations when creating/reusing targets. You can use AcquireTarget() and
// ReleaseTarget() correspondingly to reuse targets. After a target has been released, it is not safe for reuse and can
// result in race conditions
type Target struct {
	Hostname string // Hostname is the bare hostname without the port.

	// HostHeader is the host header to use in the request.
	// The host header for a request will be determined in the following order (If the value is nil, or empty, then
	// the next option will be chosen)
	// 1. HostHeader
	// 2. Hostname:Port
	HostHeader   []byte
	muHostHeader sync.Mutex

	IP       string   // the IP is the address used to reach the server. If empty, Hostname will be used
	Port     int      // Port will be the port used to reach the server.
	IsTLS    bool     // IsTLS defines whether to use a TLS dialer or normal dialer
	BasePath string   // BasePath allows for custom discovery on a target below the root directory

	// Header is an ordered list of http headers to add to the request.
	// Header behaviour is determined by fasthttp.Request  and fasthttp.RequestHeader
	// Authentication and Content-Type headers should be added to here
	// the Host header is separately governed by HostHeader
	Headers  []Header

	ctx       context.Context
	ctxcancel func()

	hits           int64 // number of hits. if this gets too high, we'll quarantine the host
	quarantineHits int64
	quarantined    int32

	b []byte

	httpClient *HTTPClient
}

// SetContext will overwrite this context and cancellation with the provided context
func (t *Target) SetContext(c context.Context) {
	t.ctx, t.ctxcancel = context.WithCancel(c)
}

// Context will return the context of this object. If nil, it will initialize a context for this target
// This context can be used across goroutines to assess the validity of the goroutine. This is independent
// of the quarantine state.
//
//		select {
//		case <-target.Context().Done():
//			return
//		case default:
//			resp, err := http.DoClient(target.HTTPClient(5, time.Second), req, &config.HTTP)
//		}
func (t *Target) Context() context.Context {
	if t.ctx == nil {
		log.Panic().Str("target", t.String()).Msg("creating fresh context for target")
		t.ctx, t.ctxcancel = context.WithCancel(context.Background())
	}
	return t.ctx
}

// Cancel will cancel the context associated with the host. This can be used to expire the target's validity across
// multiple running goroutines. This will simply cancel the context associated with the host. It relies upon user
// implementation to ensure that further interactions with the target do not occur
func (t *Target) Cancel() {
	if t.ctxcancel != nil {
		t.ctxcancel()
	}
}

// HitIncr will threadsafe increment the target hit counter.
// This should be called whenever a request is performed against the host
func (t *Target) HitIncr() int64 {
	return atomic.AddInt64(&t.hits, 1)
}

// Reset returns the old quarantine counter
func (t *Target) HitReset() int64 {
	return atomic.SwapInt64(&t.hits, 0)
}

// Hits will return the value of the number of hits this target has been used for
// use HitIncr to incremenet the hits
func (t *Target) Hits() int64 {
	return atomic.LoadInt64(&t.hits)
}

// QuarantineIncr will threadsafe incremenet the quarantine counter
// This will return the new value of the quarantine counter
func (t *Target) QuarantineIncr() int64 {
	return atomic.AddInt64(&t.quarantineHits, 1)
}

// QuarantineReset will threadsafe unquarantine the host
func (t *Target) QuarantineReset() int64 {
	return atomic.SwapInt64(&t.quarantineHits, 0)
}

// Quarantine will threadsafe quarantine the host.
// To reset the quarantine state use QuarantineReset
func (t *Target) Quarantine() {
	atomic.SwapInt32(&t.quarantined, 1)
}

// Quarantined will return whether the target has been quarantined
func (t *Target) Quarantined() bool {
	return atomic.LoadInt32(&t.quarantined) == 1
}

// ParseHostHeader will perform a thread safe update of t.HostHeader using the existing fields
// If t.HostHeader is already set, this operation will just return t.HostHeader
// otherwise, this will perform t.AppendHost(t.HostHeader[:0])
func (t *Target) ParseHostHeader() []byte {
	t.muHostHeader.Lock()
	defer t.muHostHeader.Unlock()
	if len(t.HostHeader) == 0 {
		t.HostHeader = t.AppendHost(t.HostHeader[:0])
	}
	return t.HostHeader
}

// AppendHostHeader will return the HostHeader to be used in the HTTP Request. This will first prioritize
// t.HostHeader, then t.Hostname:t.Port. You should always call ParseHostHeader before calling AppendHostHeader
// otherwise there might not be a host header. We avoid acquiring a lock since this is in the hotpath for requests
// we don't parse the host header here because that's expensive and requires acquiring a lock
func (t *Target) AppendHostHeader(buf []byte) []byte {
	return append(buf, t.HostHeader...)
}

// AppendScheme will append the scheme to the host not including the ://
func (t *Target) AppendScheme(buf []byte) []byte {
	if t.IsTLS {
		return append(buf, bHTTPS...)
	}
	return append(buf, bHTTP...)
}

// appendColonPort will append :1234 only if its not a standard port (i.e. http://:80 https://:443)
// this avoids unexpected behaviour with random clients
func (t *Target) appendColonPort(buf []byte) []byte {
	// http://:80
	if t.Port == 80 && !t.IsTLS {
		return buf
	}
	// https://:443
	if t.IsTLS && t.Port == 443 {
		return buf
	}

	buf = append(buf, ":"...)
	buf = append(buf, strconv.Itoa(t.Port)...)
	return buf
}

// AppendIPOrHostname will append the ip if set, otherwise the hostname
// use this to determine where to send the request, not the host header
// This does not include the port
func (t *Target) AppendIPOrHostname(buf []byte) []byte {
	if t.IP != "" {
		return append(buf, t.IP...)
	}
	return append(buf, t.Hostname...)
}

// AppendHost will append the host to make the request including the port.
// e.g. foo.com:80 or if t.IP is set, 1.1.1.1:80
// this can be used for the HostHeader if HostHeader is not set
func (t *Target) AppendHost(buf []byte) []byte {
	buf = t.AppendIPOrHostname(buf)
	buf = t.appendColonPort(buf)
	return buf
}

// Host will return the Host:Port or IP:Port of the target
func (t *Target) Host() string {
	w := bytebufferpool.Get()
	ret := string(t.AppendHost(w.B))
	bytebufferpool.Put(w)
	return ret
}

// HTTPClient will return a HTTPClient configured for the particular target with the configured
// maxConnections and timeout.
// This is cached after the first call, so subsequent changes to the Host and IsTLS after the
// first call of HTTPClient will not be respected
func (t *Target) HTTPClient(maxConnections int, timeout time.Duration) *HTTPClient {
	if t.httpClient == nil {
		t.httpClient = NewHTTPClient(t.Host(), t.IsTLS)
		t.httpClient.SetMaxConns(maxConnections)
		t.httpClient.ReadTimeout = timeout
		t.httpClient.WriteTimeout = timeout
	}
	return t.httpClient
}

// AppendBytes will append the full request details including the headers and scheme to the provided buffer
// e.g. http://google.com:80/foo {x-forwarded-for:127.0.0.1}
func (t *Target) AppendBytes(b []byte) []byte {
	b = t.AppendScheme(b)
	b = append(b, "://"...)
	b = t.AppendHost(b)
	b = append(b, t.BasePath...)
	if len(t.Headers) > 0 {
		b = append(b, " "...)
		for _, v := range t.Headers {
			b = append(b, "{"...)
			b = v.AppendBytes(b)
			b = append(b, "}"...)
		}
	}
	return b
}

// Write will write the target out to the buffer specified
func (t *Target) Write(b io.Writer) (int, error) {
	var (
		count int
		err   error
		n     int
	)
	if t.IsTLS {
		n, err = b.Write([]byte("https://"))
		count += n
	} else {
		n, err = b.Write([]byte("http://"))
		count += n
	}
	n, err = b.Write([]byte(t.Hostname))
	count += n
	n, err = b.Write([]byte(":"))
	count += n
	n, err = b.Write([]byte(strconv.Itoa(t.Port)))
	count += n
	n, err = b.Write([]byte(t.BasePath))
	count += n

	if len(t.Headers) > 0 {
		n, err = b.Write([]byte(" {"))
		count += n
		for _, v := range t.Headers {
			n, err = v.Write(b)
			count += n
			n, err = v.Write(b)
			count += n
		}
		n, err = b.Write([]byte("}"))
		count += n
	}
	return count, err
}

// String will return a string representation of the target
func (t *Target) String() string {
	return string(t.Bytes())
}

// Bytes will return the same output as String. This is cached in t.b
// If the target is changed after Bytes() is called, the changes will not be
// reflected
func (t *Target) Bytes() []byte {
	if len(t.b) == 0 {
		t.b = t.AppendBytes(t.b)
	}
	return t.b
}

// reset will nil out all the values. This should only be internally called by ReleaseTarget
func (t *Target) reset() {
	t.Hostname = ""
	t.IsTLS = false
	t.BasePath = ""
	t.IP = ""
	t.Hostname = ""
	t.Headers = t.Headers[:0]
	t.b = t.b[:0]
	t.QuarantineReset()
	t.HitReset()

	t.ctx = nil
	t.ctxcancel = nil
}

var (
	targetPool sync.Pool
)

// AcquireTarget retrieves a host from the shared target pool
func AcquireTarget() *Target {
	v := targetPool.Get()
	if v == nil {
		return &Target{}
	}
	return v.(*Target)
}

// ReleaseTarget releases a host into the shared target pool
func ReleaseTarget(h *Target) {
	h.reset()
	targetPool.Put(h)
}
