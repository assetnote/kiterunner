package kitebuilder

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadJSONBytes(t *testing.T) {
	type args struct {
		buf []byte
	}
	tests := []struct {
		name    string
		args    args
		want    API
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadJSONBytes(tt.args.buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadJSONBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadJSONBytes() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadJSONFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name       string
		args       args
		wantSchema []API
		wantErr    bool
	}{
		{name: "simple.json", args: args{"testdata/simple.json"}, wantSchema: []API{{
			ID:  "0cc39f78f36dbe7fe7ea94c0f2687d269d728f96",
			URL: "projectplay.xyz",
			SecurityDefinitions: map[string]SecurityDefinition{
				"developerKey": {
					In:   "header",
					Name: "X-Developer-Key",
					Type: "apiKey",
				},
			},
			Paths: map[Path]Operations{
				"/onigokko/player": map[OperationTypes]Operation{
					POST: {
						Consumes: []ContentType{
							"application/json",
						},
						Produces: []ContentType{
							"text/plain",
						},
						Description: "Adds a new Player to the system",
						OperationID: "CreatePlayer",
						Parameters: []Parameter{
							{
								In:          "header",
								Name:        "Token",
								Description: "session token for validation purposes",
								Required:    true,
								Type:        "string",
								Default:     "example-token",
							},
							{
								In:          "query",
								Name:        "QueryParam",
								Description: "hahah123",
								Required:    true,
								Type:        "string",
								Default:     "example-query",
							},
							{
								In:          "header",
								Name:        "Token",
								Description: "session token for validation purposes",
								Required:    true,
								Type:        "string",
								Default:     "example-token",
							},
							{
								In:          "body",
								Name:        "player",
								Description: "Player to create",
								Required:    true,
								Schema: &Schema{
									Type:     "object",
									Required: []string{"id", "name"},
									Properties: map[string]Schema{
										"id": {
											Type:    "integer",
											Example: int(123455),
											Default: int(5),
										},
										"name": {
											Type:    "string",
											Example: "Nathan Reline",
											Default: "Nathan Reline",
										},
									},
								},
							},
						},
					},
				},
			},
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSchema, err := LoadJSONFile(tt.args.filename)
			assert.Nil(t, err)

			expect, err := json.MarshalIndent(tt.wantSchema, "", "  ")
			assert.Nil(t, err)

			got, err := json.MarshalIndent(gotSchema, "", "  ")
			assert.Nil(t, err)

			assert.Equal(t, string(expect), string(got))

			//	if !reflect.DeepEqual(gotSchema, tt.wantSchema) {
			//		t.Errorf("LoadJSONFile() gotSchema = %v, want %v", gotSchema, tt.wantSchema)
			//	}
		})
	}
}

func TestLoadJSONString(t *testing.T) {
	type args struct {
		buf string
	}
	tests := []struct {
		name    string
		args    args
		want    API
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadJSONString(tt.args.buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadJSONString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadJSONString() got = %v, want %v", got, tt.want)
			}
		})
	}
}
