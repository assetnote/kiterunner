package http

import (
	"fmt"
	"sync"

	"github.com/valyala/fasthttp"
)

type Request struct {
	Target *Target
	Route *Route
}

func (r *Request) String() string{
	return fmt.Sprintf("{ request %v }", r.Target)
}

func (r *Request) reset() {
	r.Target = nil
	r.Route = nil
}

// WriteRequest will populate the request object with all the required information from the
// Target details, and from the route details
// The path is construction with target.Basepath + basepath + route.Path
// no slashes are added inbetween. we assume the user supplies everything
func (r *Request) WriteRequest(dst *fasthttp.Request, basepath []byte) {
	tmp := make([]byte, 0, 64)
	dst.Header.SetHostBytes(r.Target.AppendHostHeader(tmp[:0]))
	dst.Header.SetMethodBytes(r.Route.Method)

	for _, h := range r.Target.Headers {
		dst.Header.Set(h.Key, h.Value)
	}

	for _, h := range r.Route.Headers {
		dst.Header.Set(h.Key, h.Value)
	}

	dst.SetBody(r.Route.Body)

	// parse the URI and then set everything.
	u := dst.URI()

	u.SetSchemeBytes(r.Target.AppendScheme(tmp[:0]))
	tmp = tmp[:0]
	tmp = append(tmp, r.Target.BasePath...)
	tmp = append(tmp, basepath...)
	tmp = r.Route.AppendPath(tmp)
	// todo: implement your own setpathbytes that performs some normalization, however
	// does not normalize repeated slashes and /./
	u.SetPathBytes(tmp)
	u.SetQueryStringBytes(r.Route.AppendQuery(tmp[:0]))
}

var (
	requestPool sync.Pool
)

// AcquireRequest retrieves a host from the shared header pool
func AcquireRequest() *Request {
	v := requestPool.Get()
	if v == nil {
		return &Request{}
	}
	return v.(*Request)
}

// ReleaseRequest releases a host into the shared header pool
func ReleaseRequest(h *Request) {
	h.reset()
	requestPool.Put(h)
}
