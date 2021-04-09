package proute

import (
	"testing"

	"github.com/assetnote/kiterunner/pkg/kitebuilder"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestFromKitebuilderAPI(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want API
	}{
		{name: "simple.json", args: args{"./testdata/simple.json"},
			want: API{
				ID:  "0cc39f78f36dbe7fe7ea94c0f2687d269d728f96",
				URL: "projectplay.xyz",
				HeaderCrumbs: []Crumb{
					RandomStringCrumb{
						Name:    "X-Developer-Key",
						Charset: ASCIIHex,
						Length:  32,
					},
				},
				Routes: []Route{
					{
						TemplatePath: "/onigokko/player",
						Method:       "post",
						HeaderCrumbs: []Crumb{
							StaticCrumb{
								K: "Token",
								V: "example-token",
							},
							StaticCrumb{
								K: "Token",
								V: "example-token",
							},
						},
						QueryCrumbs: []Crumb{
							StaticCrumb{
								K: "QueryParam",
								V: "example-query",
							},
						},
						BodyCrumbs: []Crumb{
							ObjectCrumb{
								Name: "player",
								Elements: []Crumb{
									FloatCrumb{
										Name:  "id",
										Fixed: true,
										Val:   5,
									},
									StaticCrumb{
										K: "name",
										V: "Nathan Reline",
									},
								},
							},
						},
						ContentType: []ContentType{"application/json"},
					},
				},
			},
		},
		{name: "content-type-json", args: args{"./testdata/content-type-param.json"},
			want: API{
				ID:  "0cc39f78f36dbe7fe7ea94c0f2687d269d728f96",
				URL: "projectplay.xyz",
				HeaderCrumbs: []Crumb{
					RandomStringCrumb{
						Name:    "X-Developer-Key",
						Charset: ASCIIHex,
						Length:  32,
					},
				},
				Routes: []Route{
					{
						TemplatePath: "/onigokko/player",
						Method:       "post",
						HeaderCrumbs: []Crumb{
							StaticCrumb{
								K: "Token",
								V: "example-token",
							},
							StaticCrumb{
								K: "Token",
								V: "example-token",
							},
						},
						BodyCrumbs: []Crumb{
							ObjectCrumb{
								Name: "player",
								Elements: []Crumb{
									FloatCrumb{
										Name:  "id",
										Fixed: true,
										Val:   5,
									},
									StaticCrumb{
										K: "name",
										V: "Nathan Reline",
									},
								},
							},
						},
						ContentType: []ContentType{"application/json"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSchema, err := kitebuilder.LoadJSONFile(tt.args.filename)
			assert.Nil(t, err)
			for _, v := range gotSchema {
				got, err := FromKitebuilderAPI(v)
				assert.Nil(t, err)
				spew.Dump(err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAPI_ToKitebuilderAPI_Reflection(t *testing.T) {
	tests := []struct {
		name string
		in   API
	}{
		{"simple",
			API{
				ID:  "0cc39f78f36dbe7fe7ea94c0f2687d269d728f96",
				URL: "projectplay.xyz",
				HeaderCrumbs: []Crumb{
					RandomStringCrumb{
						Name:    "X-Developer-Key",
						Charset: ASCIIHex,
						Length:  32,
					},
				},
				Routes: []Route{
					{
						TemplatePath: "/onigokko/player",
						Method:       "post",
						HeaderCrumbs: []Crumb{
							StaticCrumb{
								K: "Token",
								V: "example-token",
							},
							StaticCrumb{
								K: "Token",
								V: "example-token",
							},
						},
						BodyCrumbs: []Crumb{
							ObjectCrumb{
								Name: "player",
								Elements: []Crumb{
									FloatCrumb{
										Name:  "id",
										Fixed: true,
										Val:   5,
									},
									StaticCrumb{
										K: "name",
										V: "Nathan Reline",
									},
								},
							},
						},
						ContentType: []ContentType{"application/json"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.ToKitebuilderAPI()
			assert.Nil(t, err)

			back, err := FromKitebuilderAPI(got)
			assert.Nil(t, err)

			assert.Equal(t, tt.in, back)
		})
	}
}
