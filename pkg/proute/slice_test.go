package proute

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromStringSlice(t *testing.T) {
	type args struct {
		in     []string
		source string
		opts   []PRouteOption
	}
	tests := []struct {
		name    string
		args    args
		want    API
		wantErr bool
	}{
		{"simple", args{[]string{"/foo"}, "sometext.file", nil}, API{URL: "sometext.file", Routes: []Route{{Method: "GET", TemplatePath: "/foo"}}}, false},
		{"simple add slash", args{[]string{"foo"}, "sometext.file", nil}, API{URL: "sometext.file", Routes: []Route{{Method: "GET", TemplatePath: "/foo"}}}, false},
		{"simple two paths", args{[]string{"/foo", "/bar"}, "sometext.file", nil}, API{URL: "sometext.file", Routes: []Route{{Method: "GET", TemplatePath: "/foo"},{Method: "GET", TemplatePath: "/bar"}}}, false},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromStringSlice(tt.args.in, tt.args.source, tt.args.opts...)
			if !tt.wantErr {
				assert.Nil(t, err)
			}

			tt.want.ID = got.ID
			assert.Equal(t, tt.want, got)
		})
	}
}
