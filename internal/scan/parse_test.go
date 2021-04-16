package scan

import (
	"context"
	"reflect"
	"testing"

	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/stretchr/testify/assert"
)

func TestParseInput(t *testing.T) {
	type args struct {
		in     string
		infile []string
	}
	tests := []struct {
		name    string
		args    args
		want    []*http.Target
		wantErr bool
	}{
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInput(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseInput() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseStdin(t *testing.T) {
	tests := []struct {
		name    string
		want    chan []*http.Target
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStdin(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStdin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseStdin() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDomain(t *testing.T) {
	type args struct {
		domain string
	}
	tests := []struct {
		name    string
		args    args
		want    []*http.Target
		wantErr bool
	}{
		{"simple", args{"foo.com"}, []*http.Target{{IsTLS: true, Hostname: "foo.com", Port: 443}, {Hostname: "foo.com", Port: 80}}, false},
		{"full uri", args{"https://foo.com"}, []*http.Target{{IsTLS: true, Hostname: "foo.com", Port: 443}}, false},
		{"full uri trailing slash", args{"https://foo.com/"}, []*http.Target{{IsTLS: true, Hostname: "foo.com", Port: 443, BasePath: "/"}}, false},
		{"full uri trailing slash subdir", args{"https://foo.com/bar/"}, []*http.Target{{IsTLS: true, Hostname: "foo.com", Port: 443, BasePath: "/bar/"}}, false},
		{"full uri with port", args{"https://foo.com:8443"}, []*http.Target{{IsTLS: true, Hostname: "foo.com", Port: 8443}}, false},
		{"full http with port", args{"http://foo.com:8080"}, []*http.Target{{IsTLS: false, Hostname: "foo.com", Port: 8080}}, false},
		{"full http with port trailing slash", args{"http://foo.com:8080/"}, []*http.Target{{IsTLS: false, Hostname: "foo.com", Port: 8080, BasePath: "/"}}, false},
		{"full http with port and path", args{"http://foo.com:8080/path"}, []*http.Target{{IsTLS: false, Hostname: "foo.com", Port: 8080, BasePath: "/path"}}, false},
		{"host with port tls", args{"foo.com:8443"}, []*http.Target{{IsTLS: true, Hostname: "foo.com", Port: 8443}}, false},
		{"host with port notls", args{"foo.com:8080"}, []*http.Target{{IsTLS: false, Hostname: "foo.com", Port: 8080}}, false},
		{"host with port notls and path", args{"foo.com:8080/bar"}, []*http.Target{{IsTLS: false, Hostname: "foo.com", Port: 8080, BasePath: "/bar"}}, false},
		{"host with path", args{"foo.com/bar"}, []*http.Target{{IsTLS: true, Hostname: "foo.com", Port: 443, BasePath: "/bar"}, {Hostname: "foo.com", Port: 80, BasePath: "/bar"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDomain(tt.args.domain)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.ElementsMatch(t, tt.want, got, "want: %v got: %v", tt.want, got)
		})
	}
}

func TestParseFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    []*http.Target
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFile(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseInput1(t *testing.T) {
	type args struct {
		in string
	}
	tests := []struct {
		name    string
		args    args
		want    []*http.Target
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInput(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseInput() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseStdin1(t *testing.T) {
	tests := []struct {
		name    string
		want    chan []*http.Target
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStdin(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStdin() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseStdin() got = %v, want %v", got, tt.want)
			}
		})
	}
}
