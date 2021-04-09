package proute

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeProtoAPI(t *testing.T) {
	type args struct {
		input APIS
	}
	tests := []struct {
		name    string
		args    args
	}{
		{"simple", args{ input: APIS{
			API{
				ID:  "0cc39f78f36dbe7fe7ea94c0f2687d269d728f96",
				URL: "projectplay.xyz",
				HeaderCrumbs: []Crumb{
					RandomStringCrumb{
						Name:    "X-Developer-Key",
						Charset: ASCIIHex,
						Length:  32,
					},
				}, Routes: []Route{
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
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tmp bytes.Buffer
			err := tt.args.input.EncodeProto(&tmp)
			assert.Nil(t, err)


			got, err := DecodeProtoAPI(bytes.NewReader(tmp.Bytes()))
			assert.Nil(t, err)
			assert.Equal(t, tt.args.input, got)
		})
	}
}
