package http

import (
	"io"
	"sync"

	"github.com/rs/zerolog"
	"github.com/valyala/bytebufferpool"
)

// Header encapsulates a header key value entry
// TODO: replace strings with byte slices
type Header struct {
	Key   string
	Value string
}

type Headers []Header

func (rr Headers) MarshalZerologArray(a *zerolog.Array) {
	for _, u := range rr {
		a.Object(u)
	}
}

func (h Header) MarshalZerologObject(e *zerolog.Event) {
	e.Str("k", h.Key).
		Str("v", h.Value)
}

func (h *Header) AppendBytes(b []byte) []byte {
	b = append(b, h.Key...)
	b = append(b, ": "...)
	b = append(b, h.Value...)
	return b
}

func (h *Header) Write(buf io.Writer) (int, error) {
	var count int
	c, err := buf.Write([]byte(h.Key))
	count += c
	c, err = buf.Write([]byte(":"))
	count += c
	c, err = buf.Write([]byte(h.Value))
	count += c
	return count, err
}

func (h *Header) String() string {
	w := bytebufferpool.Get()
	ret := string(h.AppendBytes(w.B))
	bytebufferpool.Put(w)
	return ret
}

func (h *Header) reset() {
	h.Key = ""
	h.Value = ""
}

var (
	headerPool sync.Pool
)

// AcquireHeader retrieves a host from the shared header pool
func AcquireHeader() *Header {
	v := headerPool.Get()
	if v == nil {
		return &Header{}
	}
	return v.(*Header)
}

// ReleaseHeader releases a host into the shared header pool
func ReleaseHeader(h *Header) {
	h.reset()
	headerPool.Put(h)
}
