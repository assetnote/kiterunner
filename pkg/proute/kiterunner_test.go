package proute

import (
	"testing"

	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/assetnote/kiterunner/pkg/kitebuilder"
	"github.com/stretchr/testify/assert"
)

func TestToKiterunnerRoutes(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		want    []*http.Route
		wantErr bool
	}{
		{name: "simple.json", args: args{"./testdata/simple.json"},
			want: []*http.Route{
				{
					Method: http.Method("POST"),
					Path:   []byte("/onigokko/player"),
					Source: "0cc39f78f36dbe7fe7ea94c0f2687d269d728f96",
					Headers: []http.Header{
						{Key: "X-Developer-Key", Value: "example-api-key"},
						{Key: "Token", Value: "example-token"},
						{Key: "Token", Value: "example-token"},
						{Key: "Content-Type", Value: "application/json"},
					},
					Query: []byte("QueryParam=example-query"),
					Body:  []byte(  `{"player":{"id":5,"name":"Nathan Reline"}}`  ),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSchema, err := kitebuilder.LoadJSONFile(tt.args.filename)
			assert.Nil(t, err)
			proutes, err := FromKitebuilderAPIs(gotSchema)
			assert.Nil(t, err)
			final, err := APIsToKiterunnerRoutes(proutes)
			assert.Nil(t, err)

			// need to normalize the random fields
			devkey := ""
			for _, v := range final[0].Headers {
				if v.Key == "X-Developer-Key" {
					devkey = v.Value
				}
			}
			for i, v := range tt.want[0].Headers {
				if v.Key == "X-Developer-Key" {
					tt.want[0].Headers[i].Value = devkey
				}
			}

			assert.Equal(t, tt.want[0].Method, final[0].Method)
			assert.Equal(t, tt.want[0].Path, final[0].Path)
			// TODO: Get determinsitic key sorting
			assert.Equal(t, tt.want[0].Body, final[0].Body)
			assert.Equal(t, tt.want[0].Source, final[0].Source)
			assert.Equal(t, tt.want[0].Query, final[0].Query)
			assert.ElementsMatch(t, tt.want[0].Headers, final[0].Headers)
		})
	}
}

func TestToKiterunnerRoutesFromAPI(t *testing.T) {
	type args struct {
		input API
	}
	tests := []struct {
		name    string
		args    args
		want    []*http.Route
		wantErr bool
	}{
		{name: "json", args: args{
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
						QueryCrumbs: []Crumb{
							StaticCrumb{
								K: "ChildQueryParam",
								V: "example-query",
							},
						},
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
			want: []*http.Route{
				{
					Method: http.Method("POST"),
					Path:   []byte("/onigokko/player"),
					Source: "0cc39f78f36dbe7fe7ea94c0f2687d269d728f96",
					Headers: []http.Header{
						{Key: "X-Developer-Key", Value: "example-api-key"},
						{Key: "Token", Value: "example-token"},
						{Key: "Token", Value: "example-token"},
						{Key: "Content-Type", Value: "application/json"},
					},
					Query: []byte("ChildQueryParam=example-query"),
					Body:  []byte(  `{"player":{"id":5,"name":"Nathan Reline"}}`  ),
				},
			},
		},
		{name: "post body", args: args{
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
						QueryCrumbs: []Crumb{
							StaticCrumb{
								K: "ChildQueryParam",
								V: "example-query",
							},
						},
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
							IntCrumb{
								Name:  "key",
								Fixed: true,
								Val:   123,
							},
						},
						ContentType: []ContentType{"multipart/form-data"},
					},
				},
			},
		},
			want: []*http.Route{
				{
					Method: http.Method("POST"),
					Path:   []byte("/onigokko/player"),
					Source: "0cc39f78f36dbe7fe7ea94c0f2687d269d728f96",
					Headers: []http.Header{
						{Key: "X-Developer-Key", Value: "example-api-key"},
						{Key: "Token", Value: "example-token"},
						{Key: "Token", Value: "example-token"},
						{Key: "Content-Type", Value: "multipart/form-data; boundary=hahahahahformboundaryhahahaha"},
					},
					Query: []byte("ChildQueryParam=example-query"),
					Body:  []byte(  "--hahahahahformboundaryhahahaha\r\nContent-Disposition: form-data; name=\"key\"\r\n\r\n123\r\n--hahahahahformboundaryhahahaha--\r\n"  ),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			final, err := ToKiterunnerRoutes(tt.args.input)
			assert.Nil(t, err)

			// need to normalize the random fields
			devkey := ""
			for _, v := range final[0].Headers {
				if v.Key == "X-Developer-Key" {
					devkey = v.Value
				}
			}
			for i, v := range tt.want[0].Headers {
				if v.Key == "X-Developer-Key" {
					tt.want[0].Headers[i].Value = devkey
				}
			}

			assert.Equal(t, tt.want[0].Method, final[0].Method, "mismatch method")
			assert.Equal(t, tt.want[0].Path, final[0].Path, "mismatch path")
			// TODO: Get determinsitic key sorting
			assert.Equal(t, tt.want[0].Body, final[0].Body, "mismatch body")
			assert.Equal(t, tt.want[0].Source, final[0].Source, "mismatch suorce")
			assert.Equal(t, tt.want[0].Query, final[0].Query, "mismatch query")
			assert.ElementsMatch(t, tt.want[0].Headers, final[0].Headers, "mismatch headers")
		})
	}
}
