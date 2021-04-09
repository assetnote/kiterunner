package http

import "testing"

func Test_getDepth(t *testing.T) {
	type args struct {
		path  string
		depth int64
	}
	tests := []struct {
		name    string
		args    args
		wantRet string
	}{
		{"simple", args{"/foo/bar", 2}, "/foo/bar"},
		{"simple shorter", args{"/foo/bar/baz", 2}, "/foo/bar"},
		{"simple shorter again", args{"/foo", 2}, "/foo"},
		{"simple shorter root", args{"/", 2}, "/"},
		{"simple no prefix", args{"foo/bar", 2}, "/foo/bar"},
		{"shorter", args{"foo", 2}, "/foo"},
		{"longer", args{"foo/bar/baz", 2}, "/foo/bar"},
		{"longer", args{"foo/bar/baz", 0}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRet := getDepth(tt.args.path, tt.args.depth); gotRet != tt.wantRet {
				t.Errorf("getDepth() = %v, want %v", gotRet, tt.wantRet)
			}
		})
	}
}
