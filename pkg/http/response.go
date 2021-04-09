package http

import (
	"fmt"
	"sync"

	"github.com/rs/zerolog"
)

type Response struct {
	StatusCode  int
	Words       int
	Lines       int
	BodyLength  int
	HTTPVersion string

	// these headers should not be used elsewhere, they are released into the pool on object release into the pool
	Headers []*Header
	Body    []byte

	// URI is the URI for the request. We store this here to avoid referencing the request when walking the tree
	// The first request will have the URI be nil. This is because you can reconstruct the first response
	// based off the origin request.
	// This will only be set in secondary/subsequent requests in a redirect chain
	URI []byte

	// OriginRequest corresponds to the original request used. this may not correlate to a redirected request
	OriginRequest Request
	Next          *Response
	Error         error // Whether an error occurred during the request
}

type Responses []*Response

func (rr Responses) MarshalZerologArray(a *zerolog.Array) {
	for _, u := range rr {
		a.Object(u)
	}
}

func (r *Response) Flatten() (ret Responses) {
	for v := r; v != nil; v = v.Next {
		ret = append(ret, v)
	}
	return ret
}

func (r Response) MarshalZerologObject(e *zerolog.Event) {
	e.Bytes("uri", r.URI).
		Int("sc", r.StatusCode).
		Int("len", r.BodyLength)
}

func (r *Response) AppendRedirectChain(b []byte) []byte {
	if r == nil {
		return b
	}

	b = append(b, "-> "...)
	b = append(b, r.URI...)
	b = append(b, " "...)
	return r.Next.AppendRedirectChain(b)
}

func (r *Response) String() string {
	if r == nil {
		return ""
	}

	uri := r.URI
	maxlen := 96
	if len(uri) > maxlen {
		uri = uri[0:maxlen]
		uri = append(uri, "..."...)
	}
	if r.Next != nil {
		if len(uri) == 0 {
			return fmt.Sprintf("(%d) %d -> %s", len(r.Body), r.StatusCode, r.Next)
		} else {
			return fmt.Sprintf("%s (%d) %d -> %s", uri, len(r.Body), r.StatusCode, r.Next)
		}
	}
	if len(uri) == 0 {
		return fmt.Sprintf("(%d) %d", len(r.Body), r.StatusCode)
	} else {
		return fmt.Sprintf("%s (%d) %d", uri, len(r.Body), r.StatusCode)
	}
}

func (r *Response) AddHeader(k, v []byte) {
	h := AcquireHeader()
	h.Key = string(k)
	h.Value = string(v)
	r.Headers = append(r.Headers, h)
}

func (r *Response) Reset() {
	r.StatusCode = 0
	r.HTTPVersion = ""
	for _, v := range r.Headers {
		ReleaseHeader(v)
	}
	r.Headers = r.Headers[:0]
	r.OriginRequest.Target = nil
	r.OriginRequest.Route = nil
	r.Next = nil
	r.Error = nil
	r.URI = r.URI[:0]
	r.Body = r.Body[:0]
}

var (
	responsePool sync.Pool
)

// AcquireResponse retrieves a host from the shared header pool
func AcquireResponse() *Response {
	v := responsePool.Get()
	if v == nil {
		return &Response{}
	}
	return v.(*Response)
}

// ReleaseResponse releases a host into the shared header pool
func ReleaseResponse(h *Response) {
	h.Reset()
	responsePool.Put(h)
}
