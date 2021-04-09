package http

import (
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog"
)

type Method []byte

var (
	GET    Method = []byte("GET")
	POST   Method = []byte("POST")
	PATCH  Method = []byte("PATCH")
	DELETE Method = []byte("DELETE")
	PUT    Method = []byte("PUT")
	TRACE  Method = []byte("TRACE")

	ErrUnsupportedMethod = fmt.Errorf("unsupported method")
)

func MethodFromString(m string) (Method, error) {
	switch strings.ToUpper(m) {
	case "GET":
		return GET, nil
	case "POST":
		return POST, nil
	case "PATCH":
		return PATCH, nil
	case "DELETE":
		return DELETE, nil
	case "PUT":
		return PUT, nil
	case "TRACE":
		return TRACE, nil
	}
	return GET, ErrUnsupportedMethod
}

var (
	bSlash = []byte("/")
)

type ChunkedRoutes [][]*Route

var (
	ChunkedRoutesPool sync.Pool
)

// acquireChunkedRoutes retrieves a host from the shared header pool
func AcquireChunkedRoutes() *ChunkedRoutes {
	v := ChunkedRoutesPool.Get()
	if v == nil {
		v := make(ChunkedRoutes, 0)
		return &v
	}
	return v.(*ChunkedRoutes)
}

// releaseChunkedRoutes releases a host into the shared header pool
func ReleaseChunkedRoutes(h *ChunkedRoutes) {
	*h = (*h)[:0]
	ChunkedRoutesPool.Put(h)
}

func ChunkRoutes(items []*Route, src *ChunkedRoutes, chunks int) *ChunkedRoutes {
	chunkSize := (len(items) / chunks) + 1
	for chunkSize < len(items) {
		items, *src = items[chunkSize:], append(*src, items[0:chunkSize:chunkSize])
	}

	*src = append(*src, items)
	return src
}

type Route struct {
	Headers []Header
	Path    []byte
	Query   []byte
	Source  string
	Method  Method
	Body    []byte
}

func (r Route) String() string {
	return fmt.Sprintf("%s %s%s", r.Method, r.Path, r.Query)
}

func (r Route) MarshalZerologObject(e *zerolog.Event) {
	e.Bytes("method", r.Method).
		Bytes("path", r.Path).
		Bytes("query", r.Query).
		Array("headers", Headers(r.Headers))
}

func (r Route) AppendShortBytes(b []byte) []byte {
	b = append(b, r.Method...)
	b = append(b, " "...)
	b = append(b, r.Path...)
	return b
}

func (r Route) AppendBytes(b []byte) []byte {
	b = append(b, r.Method...)
	b = append(b, " "...)
	b = r.AppendPath(b)
	if len(r.Query) > 0 {
		b = append(b, "?"...)
		b = r.AppendQuery(b)
	}
	b = append(b, " HTTP/1.1\r\n"...)

	for _, v := range r.Headers {
		b = v.AppendBytes(b)
		b = append(b, "\r\n"...)
	}

	b = append(b, "\r\n"...)
	b = append(b, r.Body...)
	return b
}

func (r *Route) AppendPath(dst []byte) []byte {
	dst = append(dst, r.Path...)
	return dst

}

func (r *Route) AppendQuery(dst []byte) []byte {
	dst = append(dst, r.Query...)
	return dst
}
