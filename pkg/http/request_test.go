package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestRequest_WriteRequest(t *testing.T) {
	type fields struct {
		Target *Target
		Route  *Route
	}
	type args struct {
		basepath []byte
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		expected string
	}{
		{"host", fields{&Target{Hostname: "google.com", Port: 80}, &Route{}}, args{}, "http://google.com/"},
		{"host path", fields{&Target{Hostname: "google.com", Port: 80}, &Route{Path: []byte("/foo")}}, args{}, "http://google.com/foo"},
		{"host base path", fields{&Target{Hostname: "google.com", BasePath: "/targetbase", Port: 80}, &Route{Path: []byte("/foo")}}, args{}, "http://google.com/targetbase/foo"},
		{"host base path extra base", fields{&Target{Hostname: "google.com", BasePath: "/targetbase", Port: 80}, &Route{Path: []byte("/foo")}}, args{[]byte("/argbase")}, "http://google.com/targetbase/argbase/foo"},
		{"host port", fields{&Target{Hostname: "google.com", Port: 9090}, &Route{Path: []byte("/foo")}}, args{}, "http://google.com:9090/foo"},
		{"host port tls", fields{&Target{Hostname: "google.com", Port: 9090, IsTLS: true}, &Route{Path: []byte("/foo")}}, args{}, "https://google.com:9090/foo"},
		{"host query params", fields{&Target{Hostname: "google.com", Port: 80}, &Route{Path: []byte("/foo"), Query: []byte("foo=bar&baz=boo")}}, args{}, "http://google.com/foo?foo=bar&baz=boo"},
		{"host base path extra base tls query params port", fields{&Target{Hostname: "google.com", BasePath: "/targetbase", Port: 9090, IsTLS: true}, &Route{Path: []byte("/foo"), Query: []byte("foo=bar&baz=boo")}}, args{[]byte("/argbase")}, "https://google.com:9090/targetbase/argbase/foo?foo=bar&baz=boo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := &fasthttp.Request{}
			r := &Request{
				Target: tt.fields.Target,
				Route:  tt.fields.Route,
			}
			r.Target.ParseHostHeader()
			r.WriteRequest(dst, tt.args.basepath)
			assert.Equal(t, tt.expected, dst.URI().String())
		})
	}
}
