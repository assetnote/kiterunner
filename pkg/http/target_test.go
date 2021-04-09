package http

import (
	"bytes"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/assetnote/kiterunner/pkg/log"
)

type fields struct {
	Hostname              string
	HostHeader            []byte
	muHostHeader          sync.Mutex
	IP                    string
	Port                  int
	IsTLS                 bool
	BasePath              string
	Headers               []Header
	DefaultStatusCode     int
	DefaultContentLength  int
	AdjustedContentLength int
	AdjustmentScale       int
	DefaultWordCount      int
	DefaultLineCount      int
	hits                  int64
	quarantineHits        int64
	quarantined           int32
	httpClient            *HTTPClient
}

func TestTarget_AppendHost(t1 *testing.T) {
	type args struct {
		buf []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{"simple host", fields{Hostname: "foo.com", Port: 80}, []byte("foo.com")},
		{"simple host diff port", fields{Hostname: "foo.com", Port: 90}, []byte("foo.com:90")},
		{"simple host tls", fields{Hostname: "foo.com", Port: 443, IsTLS: true}, []byte("foo.com")},
		{"simple host tls diff port", fields{Hostname: "foo.com", Port: 4443, IsTLS: true}, []byte("foo.com:4443")},
		{"ip specified host", fields{Hostname: "foo.com", IP: "1.1.1.1", Port: 80, IsTLS: false}, []byte("1.1.1.1")},
		{"ip specified host diff port", fields{Hostname: "foo.com", IP: "1.1.1.1", Port: 90, IsTLS: false}, []byte("1.1.1.1:90")},
		{"ip specified host tls", fields{Hostname: "foo.com", IP: "1.1.1.1", Port: 443, IsTLS: true}, []byte("1.1.1.1")},
		{"ip specified host tls diff port ", fields{Hostname: "foo.com", IP: "1.1.1.1", Port: 4443, IsTLS: true}, []byte("1.1.1.1:4443")},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.AppendHost(nil); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("AppendHost() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTarget_AppendHostHeader(t1 *testing.T) {
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{"simple host", fields{Hostname: "foo.com", Port: 80}, []byte("foo.com")},
		{"simple host diff port", fields{Hostname: "foo.com", Port: 90}, []byte("foo.com:90")},
		{"simple host tls", fields{Hostname: "foo.com", Port: 443, IsTLS: true}, []byte("foo.com")},
		{"simple host tls diff port", fields{Hostname: "foo.com", Port: 4443, IsTLS: true}, []byte("foo.com:4443")},
		{"ip specified host", fields{Hostname: "foo.com", IP: "1.1.1.1", Port: 80, IsTLS: false}, []byte("1.1.1.1")},
		{"ip specified host diff port", fields{Hostname: "foo.com", IP: "1.1.1.1", Port: 90, IsTLS: false}, []byte("1.1.1.1:90")},
		{"ip specified host tls", fields{Hostname: "foo.com", IP: "1.1.1.1", Port: 443, IsTLS: true}, []byte("1.1.1.1")},
		{"ip specified host tls diff port ", fields{Hostname: "foo.com", IP: "1.1.1.1", Port: 4443, IsTLS: true}, []byte("1.1.1.1:4443")},
		{"host header specified ip specified host tls diff port ", fields{HostHeader: []byte("override-host.com:9999"), Hostname: "foo.com", IP: "1.1.1.1", Port: 4443, IsTLS: true}, []byte("override-host.com:9999")},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			t.ParseHostHeader()
			if got := t.AppendHostHeader(nil); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("AppendHostHeader() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestTarget_AppendIPOrHostname(t1 *testing.T) {
	type args struct {
		buf []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.AppendIPOrHostname(tt.args.buf); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("AppendIPOrHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_AppendScheme(t1 *testing.T) {
	type args struct {
		buf []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.AppendScheme(tt.args.buf); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("AppendScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_HTTPClient(t1 *testing.T) {
	type args struct {
		maxConnections int
		timeout        time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *HTTPClient
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.HTTPClient(tt.args.maxConnections, tt.args.timeout); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("HTTPClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_HitIncr(t1 *testing.T) {
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.HitIncr(); got != tt.want {
				t1.Errorf("HitIncr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_HitReset(t1 *testing.T) {
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.HitReset(); got != tt.want {
				t1.Errorf("HitReset() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_Hits(t1 *testing.T) {
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.Hits(); got != tt.want {
				t1.Errorf("Hits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_Host(t1 *testing.T) {
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.Host(); got != tt.want {
				t1.Errorf("Host() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_String(t1 *testing.T) {
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.String(); got != tt.want {
				t1.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_Write(t1 *testing.T) {
	tests := []struct {
		name    string
		fields  fields
		wantB   string
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			b := &bytes.Buffer{}
			got, err := t.Write(b)
			if (err != nil) != tt.wantErr {
				t1.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotB := b.String(); gotB != tt.wantB {
				t1.Errorf("Write() gotB = %v, want %v", gotB, tt.wantB)
			}
			if got != tt.want {
				t1.Errorf("Write() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTarget_appendColonPort(t1 *testing.T) {
	type args struct {
		buf []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []byte
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Target{
				Hostname:       tt.fields.Hostname,
				HostHeader:     tt.fields.HostHeader,
				muHostHeader:   tt.fields.muHostHeader,
				IP:             tt.fields.IP,
				Port:           tt.fields.Port,
				IsTLS:          tt.fields.IsTLS,
				BasePath:       tt.fields.BasePath,
				Headers:        tt.fields.Headers,
				hits:           tt.fields.hits,
				quarantineHits: tt.fields.quarantineHits,
				quarantined:    tt.fields.quarantined,
				httpClient:     tt.fields.httpClient,
			}
			if got := t.appendColonPort(tt.args.buf); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("appendColonPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Example TargetUsage shows how to acquire a target, perform a request then release the target
// using the provided helper functions.
func Example_targetUsage() {
	t := AcquireTarget()
	t.Hostname = "example.com"
	t.Port = 443
	t.IsTLS = true
	t.BasePath = "basepath"

	// you have to call ParseHostHeader after you're done instantiating the client, otherwise
	// the request will have no host header and will fail
	t.ParseHostHeader()

	req := Request{
		Route:  &Route{Path: []byte( "/")},
		Target: t,
	}
	config := &Config{
		Timeout:     1 * time.Second,
		ReadHeaders: false,
		ReadBody:    false,
	}

	c := t.HTTPClient(5, 1*time.Second)
	resp, err := DoClient(c, req, config)
	if err != nil {
		log.Error().Err(err).Msg("failed to make request")
	}

	log.Info().Msgf("success: %+v", resp)
	ReleaseTarget(t)
}
