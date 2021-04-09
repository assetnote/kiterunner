package http

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"github.com/valyala/fasthttp/reuseport"
)

type fataler interface {
	Fatal(args ...interface{})
}

func httpServer(t fataler) net.Listener {
	ln, err := reuseport.Listen("tcp4", "127.0.0.1:9447")
	if err != nil {
		t.Fatal("failed to listen", err)
	}
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.Response.Header.AddBytesKV([]byte("x-custom-header"), []byte("key"))
			ctx.Response.AppendBodyString("foo")
		},
	}
	go s.Serve(ln)
	return ln
}

func memoryServer(t fataler) *fasthttputil.InmemoryListener {
	ln := fasthttputil.NewInmemoryListener()
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.Response.Header.AddBytesKV([]byte("x-custom-header"), []byte("key"))
			ctx.Response.AppendBodyString("foo")
		},
	}
	go s.Serve(ln)
	return ln
}

func memoryRedirectServer(t fataler) *fasthttputil.InmemoryListener {
	ln := fasthttputil.NewInmemoryListener()
	s := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.Response.SetStatusCode(302)
			ctx.Response.Header.AddBytesKV([]byte("location"), []byte("/foo"))
		},
	}
	go s.Serve(ln)
	return ln
}

func BenchmarkBenchServRequest(b *testing.B) {
	b.ReportAllocs()
	var (
		path     = "/foo"
		client   = NewHTTPClient("127.0.0.1:14000", false)
		hostname = "127.0.0.1"
		port     = 14000
		req      = Request{Route: &Route{Path: []byte( path )}, Target: &Target{Hostname: hostname, Port: port}}
		res      = Response{}
		config   = Config{
			Timeout:     1 * time.Second,
			ReadHeaders: false,
			ReadBody:    false,
		}
	)
	client.ReadTimeout = 100 * time.Millisecond
	client.WriteTimeout = 100 * time.Millisecond

	var err error
	req.Target.ParseHostHeader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err = DoClient(client, req, &config)
		if err != nil {
			b.Fatal("bad err", err)
		}
		if res.BodyLength != 9 {
			b.Fatal("bad length", res.BodyLength)
		}
		if res.Words != 2 {
			b.Fatal("bad wordcount", res.Words)
		}
	}
	_ = res
	_ = err
}


func BenchmarkTCPRequest(b *testing.B) {
	b.ReportAllocs()
	var (
		memlist  = httpServer(b)
		path     = "/foo"
		client   = NewHTTPClient(memlist.Addr().String(), false)
		hostname = strings.SplitN(memlist.Addr().String(), ":", 2)[0]
		port     = 80
		req      = Request{Route: &Route{Path: []byte( path )}, Target: &Target{Hostname: hostname, Port: port}}
		res      = Response{}
		config   = Config{
			Timeout:     1 * time.Second,
			ReadHeaders: false,
			ReadBody:    false,
		}
	)
	defer memlist.Close()
	client.ReadTimeout = 10 * time.Millisecond
	client.WriteTimeout = 10 * time.Millisecond

	var err error
	req.Target.ParseHostHeader()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		res, err = DoClient(client, req, &config)
		if err != nil {
			b.Fatal("bad err", err)
		}
		if res.BodyLength != 3 {
			b.Fatal("bad length", res.BodyLength)
		}
		if res.Words != 1 {
			b.Fatal("bad wordcount", res.Words)
		}
	}
	_ = res
	_ = err
}

func BenchmarkMemoryRequest(b *testing.B) {
	b.ReportAllocs()
	var (
		memlist  = memoryServer(b)
		path     = "/foo"
		client   = NewHTTPClient(memlist.Addr().String(), false)
		hostname = strings.SplitN(memlist.Addr().String(), ":", 2)[0]
		port     = 80
		req      = Request{Route: &Route{Path: []byte( path )}, Target: &Target{Hostname: hostname, Port: port}}
		config   = Config{
			Timeout:     1 * time.Second,
			ReadHeaders: false,
			ReadBody:    false,
		}
	)
	defer memlist.Close()
	client.Dial = func(addr string) (net.Conn, error) {
		return memlist.Dial()
	}
	client.ReadTimeout = 10 * time.Millisecond
	client.WriteTimeout = 10 * time.Millisecond

	var err error
	for i := 0; i < b.N; i++ {
		_, err = DoClient(client, req, &config)
		if err != nil {
			b.Fatalf("bad %v", err)
		}
	}
	_ = err
}

func BenchmarkMemoryRedirectRequest(b *testing.B) {
	log.SetLevelString("error")
	b.ReportAllocs()
	var (
		memlist  = memoryRedirectServer(b)
		path     = "/foo"
		client   = NewHTTPClient(memlist.Addr().String(), false)
		hostname = strings.SplitN(memlist.Addr().String(), ":", 2)[0]
		port     = 80
		req      = Request{Route: &Route{Path: []byte( path )}, Target: &Target{Hostname: hostname, Port: port}}
		resp     = Response{}
		config   = Config{
			Timeout:      1 * time.Second,
			ReadHeaders:  false,
			ReadBody:     false,
			MaxRedirects: 1,
		}
	)
	defer memlist.Close()
	client.Dial = func(addr string) (net.Conn, error) {
		return memlist.Dial()
	}
	client.ReadTimeout = 10 * time.Millisecond
	client.WriteTimeout = 10 * time.Millisecond

	req.Target.ParseHostHeader()

	var err error
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, err = DoClient(client, req, &config)
		if err != nil {
			b.Fatalf("bad %v", err)
		}

		if resp.Next == nil {
			b.Fatalf("bad resp %v", resp)
		}
	}
	_ = err
}
