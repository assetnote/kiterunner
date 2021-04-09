package kiterunner

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type WildcardResponse struct {
	DefaultStatusCode     int
	DefaultContentLength  int
	AdjustedContentLength int // adjustedContentLength is the content length adjusted for the length of the requested path
	AdjustmentScale       int // number of times the requested path appears in the request
	DefaultWordCount      int // number of spaces + 1
	DefaultLineCount      int // number of newlines + 1
	// TODO: Include request method in matching
	// However not sure if this is a good idea, as OPTIONS/PUT/DELETE are likely to all match on GET/POST
	// so maybe including request method might increase noise than reducing it
	//   RequestMethod []byte
}

type WildcardResponses []WildcardResponse

func (w WildcardResponses) UniqueAdd(wr WildcardResponse) (WildcardResponses, bool) {
	for _, v := range w {
		if v == wr {
			return w, false
		}
	}
	w = append(w, wr)
	return w, true
}

type ReqMsgType int

const (
	PreflightMsg ReqMsgType = iota
	RequestMsg
)

type ReqMsg struct {
	typ       ReqMsgType
	Preflight *subpathBaseline
	Job       *job
}

func (s *ReqMsg) reset() {
	s.Preflight = nil
	s.Job = nil
}

var (
	ReqMsgPool sync.Pool
)

// AcquireReqMsg retrieves a host from the shared header pool
func acquireReqMsg() *ReqMsg {
	v := ReqMsgPool.Get()
	if v == nil {
		return &ReqMsg{}
	}
	return v.(*ReqMsg)
}

// ReleaseReqMsg releases a host into the shared header pool
func releaseReqMsg(h *ReqMsg) {
	h.reset()
	ReqMsgPool.Put(h)
}

type subpathRoutes struct {
	routes    []*http.Route
	responses chan *subpathBaseline
}

type subpathBaseline struct {
	base     []byte
	target   *http.Target
	route    *http.Route
	baseline WildcardResponse
	err      error
	resp     chan *subpathBaseline
}

func (s *subpathBaseline) reset() {
	s.base = s.base[:0]
	s.route = nil
	s.target = nil
	s.baseline = WildcardResponse{}
	s.resp = nil
	s.err = nil
}

var (
	subpathBaselinePool sync.Pool
)

// AcquiresubpathBaseline retrieves a host from the shared header pool
func acquireSubpathBaseline() *subpathBaseline {
	v := subpathBaselinePool.Get()
	if v == nil {
		return &subpathBaseline{}
	}
	return v.(*subpathBaseline)
}

// ReleasesubpathBaseline releases a host into the shared header pool
func releaseSubpathBaseline(h *subpathBaseline) {
	h.reset()
	subpathBaselinePool.Put(h)
}

var (
	PreflightCheckRoutes = []*http.Route{
		{ // try a nested directory
			Method: http.GET,
			Path:   []byte("/" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16] + "/" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16] ),
		},
		{ // docroot
			Method: http.GET,
			Path:   []byte("/"),
		},
		{ // docroot with long path to break nginx. After this length, it just cries
			Method: http.GET,
			Path:   []byte("/" + strings.Repeat("A", 1500)),
		},
		{ // docroot
			Method: http.POST,
			Path:   []byte("/"),
		},
		{ // docroot with auth header with post method
			Method:  http.PUT,
			Path:    []byte("/auth" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16]),
			// previously 1:1\n (MToxCg==) would cause applications to 500 instead of 401
			// so now using 1:1 instead to trigger a baseline
			Headers: []http.Header{{"Authorization", "Basic MTox"}},
		},
		{ // docroot with auth header
			Method:  http.GET,
			Path:    []byte("/auth" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16]),
			// previously 1:1\n (MToxCg==) would cause applications to 500 instead of 401
			// so now using 1:1 instead to trigger a baseline
			Headers: []http.Header{{"Authorization", "Basic MTox"}},
		},
		// { // method body mismatch. Catches google 400s
		// 	Method:  http.GET,
		// 	Headers: []http.Header{{"Content-Type", "applicaton/json"}},
		// 	Path:    []byte("/" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16]),
		// 	Body:    []byte("{}"),
		// },
		// Attempt other methods for NodeJS stuff
		{ // a random file
			Method: http.GET,
			Path:   []byte("/" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16]),
		},
		{
			Method: http.PUT,
			Path:   []byte("/" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16]),
		},
		{
			Method: http.POST,
			Path:   []byte("/" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16]),
		},
		{
			Method: http.DELETE,
			Path:   []byte("/" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16]),
		},
		{
			Method: http.PATCH,
			Path:   []byte("/" + strings.Replace(uuid.New().String(), "-", "", -1)[0:16]),
		},
	}
)

type ErrFailedPreflight struct {
	err error
}

func (e ErrFailedPreflight) Error() string {
	return fmt.Sprintf("failed to pass preflight checks: %s", e.err)
}

func (e ErrFailedPreflight) Unwrap() error {
	return e.err
}

// targetWildcardDetection attempts to determine what elements of the response
// correspond to a wildcard. This is then used when validating future requests against the target
// to discard any erronous results
func targetWildcardDetection(resp *fasthttp.Response, basepath string) WildcardResponse {
	// remove the preceeding slash
	if basepath[0] == '/' {
		basepath = basepath[1:]
	}

	wr := WildcardResponse{}

	// calculate the adjustment and auto-detection of invalid responses
	wr.DefaultStatusCode = resp.StatusCode()

	body := resp.Body()
	wr.DefaultContentLength = len(body)
	// TODO: Benchmark if this is even computationally efficient
	wr.DefaultWordCount = bytes.Count(body, []byte(" "))
	wr.DefaultLineCount = bytes.Count(body, []byte("\n"))

	// if its a 1 word body with no spaces, then naturally word and line count will be 0
	// so we +1 to mitigate against off by one
	if len(body) > 0 {
		wr.DefaultWordCount += 1
		wr.DefaultLineCount += 1
	}

	adjustedBody := bytes.ReplaceAll(body, []byte(basepath), []byte(""))
	wr.AdjustedContentLength = len(adjustedBody)
	diff := wr.DefaultContentLength - wr.AdjustedContentLength
	if diff > 0 && diff%len(basepath) != 0 {
		log.Fatal().Int("adjustedContentLength", wr.AdjustedContentLength).
			Int("defaultContentLength", wr.DefaultContentLength).
			Int("pathLength", len(basepath)).
			Str("path", basepath).
			Msg("basepath does not match scale factor expectation")
	}
	if len(basepath) != 0 {
		wr.AdjustmentScale = diff / len(basepath)
	}

	log.Debug().Int("adjustedContentLength", wr.AdjustedContentLength).
		Int("defaultContentLength", wr.DefaultContentLength).
		Int("pathLength", len(basepath)).
		Str("path", basepath).
		Int("bodyLength", len(body)).
		// Str("body", string(body)).
		// Bytes("headers", resp.Header.Header()).
		Int("statusCode", wr.DefaultStatusCode).
		Msg("wildcard detection complete")

	return wr
}

// preflightCheck will perform a basic http request against the target and return an error if it failed.
// this will also perform wildcard detection and populate the target with the corresponding wildcardThreshold data
// if nil is returned the request succeeded
// PreflightCheck will perform a set of preflight checks against the provided basepath
// This should be a '/' prefixed string, if this is empty, it will default to '/'
// this will return the wildcard responses calculated from the preflight check routes
// semaphore can be the jobsemaphore or nil. If its nil, then we instantiate one temporarily
// this ensures the client doesn't exhaust all its available connections
func preflightCheck(route *http.Route, t *http.Target, config *Config, basepath []byte) (ret WildcardResponse, err error) {
	hr := http.Request{
		Target: t,
		Route:  route,
	}
	var (
		req  = fasthttp.AcquireRequest()
		resp = fasthttp.AcquireResponse()
	)
	hr.WriteRequest(req, basepath)

	for _, h := range config.HTTP.ExtraHeaders {
		req.Header.Set(h.Key, h.Value)
	}

	c := t.HTTPClient(config.MaxConnPerHost, config.HTTP.Timeout)
	if err := c.Do(req, resp); err != nil {
		if strings.Contains(err.Error(), "too many open files") {
			log.Fatal().Err(err).Msg("low fd limit detected. please increase your open file limit (ulimit -n 20000)")
		}
		// make sure we mark this as "complete". Suboptimal yes. but what else can we do
		return ret, err
	}

	if config.WildcardDetection {
		ret = targetWildcardDetection(resp, string(req.URI().Path()))
	}

	// log.Error().Str("req", req.URI().String()).Msgf("preflight req. baseline: %+v", ret)

	fasthttp.ReleaseRequest(req)
	fasthttp.ReleaseResponse(resp)
	return ret, nil
}
