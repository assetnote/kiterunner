package http

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDoClientMemoryRequest(t *testing.T) {
	var (
		memlist  = memoryServer(t)
		path     = "foo"
		client   = NewHTTPClient(memlist.Addr().String(), false)
		hostname = strings.SplitN(memlist.Addr().String(), ":", 2)[0]
		port     = 80
		req      = Request{Route: &Route{Path: []byte(path)}, Target: &Target{Hostname: hostname, Port: port}}
		resp     = Response{}
		config   = Config{
			Timeout:     1 * time.Second,
			ReadHeaders: true,
			ReadBody:    true,
		}
	)
	defer memlist.Close()
	client.Dial = func(addr string) (net.Conn, error) {
		return memlist.Dial()
	}
	client.ReadTimeout = 10 * time.Millisecond
	client.WriteTimeout = 10 * time.Millisecond

	req.Target.ParseHostHeader()
	resp, err := DoClient(client, req, &config)
	assert.Nil(t, err)

	// we expect the first uri to be empty
	assert.Equal(t, "", string(resp.URI))

	// we also expect our headers and body to come back as expected
	expected := []*Header{
		{"X-Custom-Header", "key"},
	}
	for _, v := range expected {
		assert.Contains(t, resp.Headers, v)
	}

	assert.Equal(t, "foo", string(resp.Body))
}

// TestDoClientHeaderBodyRequest will test whether the request will populate the header and body requirements
// from the target route
// this should also validate that a custom set host header in the target or route will overwrite
// the default host header
func TestDoClientHeaderBodyRequest(t *testing.T) {
	simpleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for k, v := range r.Header {
			w.Header().Set(k, v[0])
		}
		w.Header().Set("x-request-url", r.URL.String())
		w.Header().Set("x-host", r.Host)

		w.WriteHeader(201)
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			// yes this won't fail the test but w.e.
			t.Fatalf("failed to read body: %v", err)
		}
		w.Write(body)
	}))

	var (
		path     = "/foo"
		client   = NewHTTPClient(simpleServer.Listener.Addr().String(), false)
		hostname = strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[0]
		port, _  = strconv.Atoi(strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[1])
		req      = Request{
			Route: &Route{
				Path: []byte( path ),
				Headers: []Header{
					{Key: "route-header", Value: "1"},
					{Key: "overlap-header", Value: "2"}},
				Body: []byte( "route body" ),
			},
			Target: &Target{
				Hostname: hostname,
				Port:     port,
				Headers: []Header{
					{Key: "overlap-header", Value: "1"},
					{Key: "target-header", Value: "1"},
					{Key: "Host", Value: "fakehost123"},
				},
			},
		}
		resp   = Response{}
		config = Config{
			Timeout:     1 * time.Second,
			ReadHeaders: true,
			ReadBody:    true,
		}
	)
	req.Target.ParseHostHeader()
	resp, err := DoClient(client, req, &config)
	assert.Nil(t, err)

	// we expect the request uri for the first request to be empty
	_ = fmt.Sprintf("%s%s", simpleServer.URL, path)
	assert.Equal(t, "", string(resp.URI))

	// we also expect our headers and body to come back as expected
	expected := []*Header{
		{"X-Request-Url", "/foo"},
		{"Overlap-Header", "2"},
		{"Target-Header", "1"},
		{"Route-Header", "1"},
		{"X-Host", "fakehost123"},
	}
	for _, v := range expected {
		assert.Contains(t, resp.Headers, v)
	}

	assert.Equal(t, "route body", string(resp.Body))
}

// TestDoClientHostHeaderRequest should validate that a custom hostname for a target will be obeyed
// this will not have the port appended. You need to manually append the port if youw ant a custom host header
func TestDoClientHostHeaderRequest(t *testing.T) {
	simpleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-custom-header", "key")
		w.Header().Set("x-request-url", r.URL.String())
		w.Header().Set("x-host", r.Host)

		w.WriteHeader(201)
		body, err := httputil.DumpRequest(r, true)
		assert.Nil(t, err)
		w.Write(body)
	}))

	var (
		path     = "/foo"
		client   = NewHTTPClient(simpleServer.Listener.Addr().String(), false)
		hostname = strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[0]
		port, _  = strconv.Atoi(strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[1])
		req      = Request{
			Route: &Route{Path: []byte( path )},
			Target: &Target{
				HostHeader: []byte("diff-hostname-123"),
				IP:         hostname,
				Port:       port,
			},
		}
		resp   = Response{}
		config = Config{
			Timeout:     1 * time.Second,
			ReadHeaders: true,
			ReadBody:    true,
		}
	)
	req.Target.ParseHostHeader()
	resp, err := DoClient(client, req, &config)
	assert.Nil(t, err)

	// we expect the request uri for the first request to be empty
	_ = fmt.Sprintf("%s%s", simpleServer.URL, path)
	assert.Equal(t, "", string(resp.URI))

	// we also expect our headers and body to come back as expected
	expected := []*Header{
		{"X-Custom-Header", "key"},
		{"X-Request-Url", "/foo"},
		{"X-Host", "diff-hostname-123"},
	}
	for _, v := range expected {
		assert.Contains(t, resp.Headers, v)
	}

	assert.NotEqual(t, "", string(resp.Body))
}

// TestDoClientHostHeaderRequestCustom should validate that a custom hostname for a target will be obeyed
// and will not include the port
func TestDoClientHostHeaderRequestCustom(t *testing.T) {
	simpleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-custom-header", "key")
		w.Header().Set("x-request-url", r.URL.String())
		w.Header().Set("x-host", r.Host)

		w.WriteHeader(201)
		body, err := httputil.DumpRequest(r, true)
		assert.Nil(t, err)
		w.Write(body)
	}))

	var (
		path     = "/foo"
		client   = NewHTTPClient(simpleServer.Listener.Addr().String(), false)
		hostname = strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[0]
		port, _  = strconv.Atoi(strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[1])
		req      = Request{
			Route: &Route{Path: []byte( path )},
			Target: &Target{
				HostHeader: []byte("no-port-host-header"),
				Hostname:   "diff-hostname-123",
				IP:         hostname,
				Port:       port,
			},
		}
		resp   = Response{}
		config = Config{
			Timeout:     1 * time.Second,
			ReadHeaders: true,
			ReadBody:    true,
		}
	)
	req.Target.ParseHostHeader()
	resp, err := DoClient(client, req, &config)
	assert.Nil(t, err)

	// we expect the request uri for the first request to be empty
	_ = fmt.Sprintf("%s%s", simpleServer.URL, path)
	assert.Equal(t, "", string(resp.URI))

	// we also expect our headers and body to come back as expected
	expected := []*Header{
		{"X-Custom-Header", "key"},
		{"X-Request-Url", "/foo"},
		{"X-Host", "no-port-host-header"},
	}
	for _, v := range expected {
		assert.Contains(t, resp.Headers, v)
	}

	assert.NotEqual(t, "", string(resp.Body))
}

func TestDoClientSimpleRequest(t *testing.T) {
	simpleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-custom-header", "key")
		w.Header().Set("x-request-url", r.URL.String())
		w.Header().Set("x-host", r.Host)

		w.WriteHeader(201)
		body, err := httputil.DumpRequest(r, true)
		assert.Nil(t, err)
		w.Write(body)
	}))

	var (
		path     = "/foo"
		client   = NewHTTPClient(simpleServer.Listener.Addr().String(), false)
		hostname = strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[0]
		port, _  = strconv.Atoi(strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[1])
		req      = Request{Route: &Route{Path: []byte(path)}, Target: &Target{Hostname: hostname, Port: port}}
		resp     = Response{}
		config   = Config{
			Timeout:     1 * time.Second,
			ReadHeaders: true,
			ReadBody:    true,
		}
	)
	req.Target.ParseHostHeader()
	resp, err := DoClient(client, req, &config)
	assert.Nil(t, err)

	// we expect the request uri for the first request to be empty
	_ = fmt.Sprintf("%s%s", simpleServer.URL, path)
	assert.Equal(t, "", string(resp.URI))

	// we also expect our headers and body to come back as expected
	expected := []*Header{
		{"X-Custom-Header", "key"},
		{"X-Request-Url", "/foo"},
	}
	for _, v := range expected {
		assert.Contains(t, resp.Headers, v)
	}

	assert.NotEqual(t, "", string(resp.Body))
}

func TestDoClientRedirectRequest(t *testing.T) {
	simpleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-custom-header", "key")
		w.Header().Set("x-request-url", r.URL.String())
		w.Header().Set("x-host", r.Host)

		if r.URL.Path == "/before" {
			w.Header().Set("Location", "/after")
			w.WriteHeader(302)
		} else if r.URL.Path == "/after" {
			w.WriteHeader(201)
		}

		body, err := httputil.DumpRequest(r, true)
		assert.Nil(t, err)
		w.Write(body)
	}))

	var (
		path     = "/before"
		client   = NewHTTPClient(simpleServer.Listener.Addr().String(), false)
		hostname = strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[0]
		port, _  = strconv.Atoi(strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[1])
		req      = Request{Route: &Route{Path: []byte(path)}, Target: &Target{Hostname: hostname, Port: port}}
		resp     = Response{}
		config   = Config{
			Timeout:      1 * time.Second,
			ReadHeaders:  true,
			ReadBody:     true,
			MaxRedirects: 2,
		}
	)
	req.Target.ParseHostHeader()
	resp, err := DoClient(client, req, &config)
	assert.Nil(t, err)

	// First Request
	resp = resp
	// we expect the request uri to be empty
	assert.Equal(t, "", string(resp.URI))
	// it should be a redirect
	assert.Equal(t, 302, resp.StatusCode)
	// we also expect our headers and body to come back as expected
	expected := []*Header{
		{"X-Custom-Header", "key"},
		{"X-Request-Url", "/before"},
	}
	for _, v := range expected {
		assert.Contains(t, resp.Headers, v)
	}
	assert.NotEqual(t, "", string(resp.Body))

	// redirected request
	nresp := resp.Next
	// url := fmt.Sprintf("%s/after", simpleServer.URL)
	// we expect the URI to contain the location header
	assert.Equal(t, "/after", string(nresp.URI))
	// it should be a redirect
	assert.Equal(t, 201, nresp.StatusCode)
	// we also expect our headers and body to come back as expected
	expected = []*Header{
		{"X-Custom-Header", "key"},
		// only expect the path because its the correct host
		{"X-Request-Url", "/after"},
	}
	for _, v := range expected {
		assert.Contains(t, nresp.Headers, v)
	}
	assert.NotEqual(t, "", string(nresp.Body))

	// There should be no more after this
	assert.Nil(t, nresp.Next)
}

func TestDoClientRedirectRequestSameFullHost(t *testing.T) {
	simpleServer := httptest.NewServer(nil)
	simpleServer.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-custom-header", "key")
		w.Header().Set("x-request-url", r.URL.String())
		w.Header().Set("x-host", r.Host)

		if r.URL.Path == "/before" {
			w.Header().Set("Location", simpleServer.URL+"/after")
			w.WriteHeader(302)
		} else if r.URL.Path == "/after" {
			w.WriteHeader(201)
		}

		body, err := httputil.DumpRequest(r, true)
		assert.Nil(t, err)
		w.Write(body)
	})

	var (
		path     = "/before"
		client   = NewHTTPClient(simpleServer.Listener.Addr().String(), false)
		hostname = strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[0]
		port, _  = strconv.Atoi(strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[1])
		req      = Request{Route: &Route{Path: []byte(path)}, Target: &Target{Hostname: hostname, Port: port}}
		resp     = Response{}
		config   = Config{
			Timeout:      1 * time.Second,
			ReadHeaders:  true,
			ReadBody:     true,
			MaxRedirects: 2,
		}
	)
	req.Target.ParseHostHeader()
	resp, err := DoClient(client, req, &config)
	assert.Nil(t, err)

	// First Request
	resp = resp
	// we expect the request uri to be empty
	assert.Equal(t, "", string(resp.URI))
	// it should be a redirect
	assert.Equal(t, 302, resp.StatusCode)
	// we also expect our headers and body to come back as expected
	expected := []*Header{
		{"X-Custom-Header", "key"},
		{"X-Request-Url", "/before"},
	}
	for _, v := range expected {
		assert.Contains(t, resp.Headers, v)
	}
	assert.NotEqual(t, "", string(resp.Body))

	// redirected request
	nresp := resp.Next
	url := fmt.Sprintf("%s/after", simpleServer.URL)
	// we expect the URI to contain the location header
	assert.Equal(t, url, string(nresp.URI))
	// it should be a redirect
	assert.Equal(t, 201, nresp.StatusCode)
	// we also expect our headers and body to come back as expected
	expected = []*Header{
		{"X-Custom-Header", "key"},
		// we expect this to be the path because the hostname is the same
		{"X-Request-Url", "/after"},
	}
	for _, v := range expected {
		assert.Contains(t, nresp.Headers, v)
	}
	assert.NotEqual(t, "", string(nresp.Body))

	// There should be no more after this
	assert.Nil(t, nresp.Next)
}

func TestDoClientRedirectRequestMultiHost(t *testing.T) {
	afterServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-custom-header", "key2")
		w.Header().Set("x-request-url", r.URL.String())
		w.Header().Set("x-host", r.Host)
		w.WriteHeader(201)

		body, err := httputil.DumpRequest(r, true)
		assert.Nil(t, err)
		w.Write(body)
	}))

	simpleServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-custom-header", "key")
		w.Header().Set("x-request-url", r.URL.String())
		w.Header().Set("x-host", r.Host)
		w.Header().Set("Location", afterServer.URL)
		w.WriteHeader(302)

		body, err := httputil.DumpRequest(r, true)
		assert.Nil(t, err)
		w.Write(body)
	}))

	var (
		path     = "/ignored"
		client   = NewHTTPClient(simpleServer.Listener.Addr().String(), false)
		hostname = strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[0]
		port, _  = strconv.Atoi(strings.SplitN(simpleServer.Listener.Addr().String(), ":", 2)[1])
		req      = Request{Route: &Route{Path: []byte(path)}, Target: &Target{Hostname: hostname, Port: port}}
		resp     = Response{}
		config   = Config{
			Timeout:      1 * time.Second,
			ReadHeaders:  true,
			ReadBody:     true,
			MaxRedirects: 2,
		}
	)
	req.Target.ParseHostHeader()
	resp, err := DoClient(client, req, &config)
	assert.Nil(t, err)

	// First Request
	resp = resp
	// we expect the request uri to be empty
	assert.Equal(t, "", string(resp.URI))
	// it should be a redirect
	assert.Equal(t, 302, resp.StatusCode)
	// we also expect our headers and body to come back as expected
	expected := []*Header{
		{"X-Custom-Header", "key"},
		{"X-Request-Url", "/ignored"},
	}
	for _, v := range expected {
		assert.Contains(t, resp.Headers, v)
	}
	assert.NotEqual(t, "", string(resp.Body))

	// redirected request
	nresp := resp.Next
	// we expect the request uri to be empty
	url := fmt.Sprintf("%s", afterServer.URL)
	assert.Equal(t, url, string(nresp.URI))
	// it should be a redirect
	assert.Equal(t, 201, nresp.StatusCode)
	// we also expect our headers and body to come back as expected
	expected = []*Header{
		{"X-Custom-Header", "key2"},
		{"X-Request-Url", "/"},
	}
	for _, v := range expected {
		assert.Contains(t, nresp.Headers, v)
	}
	assert.NotEqual(t, "", string(nresp.Body))

	// There should be no more after this
	assert.Nil(t, nresp.Next)
}
