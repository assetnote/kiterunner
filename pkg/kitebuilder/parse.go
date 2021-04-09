package kitebuilder

import (

	// "github.com/goccy/go-json"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"

	errors2 "github.com/assetnote/kiterunner/pkg/errors"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/hashicorp/go-multierror"
)

// LoadJSONFile will load the json schema from the specified file
func LoadJSONFile(filename string) (schema []API, err error) {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return
	}
	defer jsonFile.Close()

	data, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}

	return LoadJSONBytes(data)
}

func LoadJSONReader(r io.Reader) (schema []API, err error) {
	res, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return LoadJSONBytes(res)
}

// SlowLoadJSONFile will load the json schema from the specified file
func SlowLoadJSONFile(filename string) (schema []API, err error) {
	jsonFile, err := os.Open(filename)
	if err != nil {
		return
	}
	defer jsonFile.Close()

	data, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}

	return SlowLoadJSONBytes(data)
}

// LoadJSONBytes will load the schema from the provided bytes
func LoadJSONBytes(buf []byte) ([]API, error) {
	var ret []API
	if err := json.Unmarshal(buf, &ret); err != nil {
		return ret, fmt.Errorf("failed to unmarshal json: %w", err)
	}
	return ret, nil
}

// SlowLoadJSONBytes will unmarshal the buf into an interface{} and attempt
// to unmarshal each individual API while printing errors about the object
// This is a painful manual process when the kitebuilder spec was non-finalised
func SlowLoadJSONBytes(buf []byte) ([]API, error) {
	var (
		ret    []API
		all    []interface{}
		merr = &multierror.Error{}
	)
	log.Debug().Msg("beginning slow load of json bytes")
	if err := json.Unmarshal(buf, &all); err != nil {
		return ret, fmt.Errorf("failed to unmarshal json: %w", err)
	}

	for _, v := range all {
		// encode json for debugging
		bv, err := json.Marshal(v)
		if err != nil {
			return ret, fmt.Errorf("failed to encode object: %w", err)
		}

		switch v.(type) {
		case []interface{}:
			// this is a nested slice
			for _, v := range v.([]interface{}) {
				// encode json for debugging
				bv, err := json.Marshal(v)
				if err != nil {
					log.Trace().Err(err).Msg("failed to encode object")
				}

				switch v.(type) {
				case map[string]interface{}:
					// we have reached the API object
					a, err := unmarshalToAPI(v.(map[string]interface{}))
					if err != nil {
						merr = multierror.Append(merr, &errors2.ParserError{
							ID:      "root",
							RawJSON: bv,
							Err:     fmt.Errorf("failed to unmarhsal api: %w", err),
							Context: "root.[].[].v",
						})
					}
					ret = append(ret, a)
				default:
					got := reflect.TypeOf(v).Kind()
					merr = multierror.Append(merr, &errors2.ParserError{
						ID:      "root",
						RawJSON: bv,
						Err:     fmt.Errorf("unexpected type: %s", got),
						Context: "root.[].[].v",
					})
				}
			}
		case map[string]interface{}:
			// we have reached the API object
			a, err := unmarshalToAPI(v.(map[string]interface{}))
			if err != nil {
				merr = multierror.Append(merr, &errors2.ParserError{
					ID:      "root",
					RawJSON: bv,
					Err:     fmt.Errorf("failed to unmarshal api: %w", err),
					Context: "root.[].v",
				})
			}
			ret = append(ret, a)
		default:
			got := reflect.TypeOf(v).Kind()
			merr = multierror.Append(merr, &errors2.ParserError{
				ID:      "root",
				RawJSON: bv,
				Err:     fmt.Errorf("unexpected type: %s", got),
				Context: "root.[].v",
			})
		}
	}
	return ret, merr.ErrorOrNil()
}

func unmarshalToAPI(v map[string]interface{}) (API, error) {
	var (
		ret  API
		err  error
		merr = &multierror.Error{}
	)

	ret.SecurityDefinitions = make(map[string]SecurityDefinition)
	ret.Paths = make(map[Path]Operations)

	bv, err := json.Marshal(v)
	if err != nil {
		return ret, fmt.Errorf("failed to marshal map for input: %w", err)
	}

	// decode url
	ret.URL, err = GetMapString(v, "url")
	if err != nil {
		merr = multierror.Append(merr, &errors2.ParserError{
			Context: "api",
			Err:     fmt.Errorf("failed to get url: %w", err),
			RawJSON: bv,
		})
	}

	// decode id
	ret.ID, err = GetMapString(v, "ksuid")
	if err != nil {
		merr = multierror.Append(merr, &errors2.ParserError{
			Context: "api",
			Err:     fmt.Errorf("failed to get api id: %w", err),
			RawJSON: bv,
		})
	}

	log.Trace().Str("url", ret.URL).Str("id", ret.ID).Msg("parsing api")
	if err := UnmarshalJSONString(v, "securityDefinitions", &ret.SecurityDefinitions); err != nil {
		merr = multierror.Append(merr, &errors2.ParserError{
			ID:      ret.ID,
			Context: "api",
			Err:     fmt.Errorf("failed to get security definitions: %w", err),
			RawJSON: bv,
		})
	}

	// decode paths
	if v, ok := v["paths"]; ok {
		if mv, ok := v.(map[string]interface{}); ok {
			for path, v := range mv {
				ret.Paths[Path(path)] = make(map[OperationTypes]Operation)
				if opv, ok := v.(map[string]interface{}); ok {
					for operation, v := range opv {
						bv, err := json.Marshal(v)
						if err != nil {
							return ret, fmt.Errorf("failed to marshal map for input: %w", err)
						}

						if op, ok := v.(map[string]interface{}); ok {
							var o Operation
							if o.Description, err = GetMapString(op, "description"); err != nil {
								merr = multierror.Append(merr, &errors2.ParserError{
									ID:      ret.ID,
									Route:   path,
									Method:  operation,
									Context: "api.paths.operations.v",
									Err:     fmt.Errorf("failed to get operation description: %w", err),
									RawJSON: bv,
								})
							}
							if o.OperationID, err = GetMapString(op, "operationId"); err != nil {
								merr = multierror.Append(merr, &errors2.ParserError{
									ID:      ret.ID,
									Route:   path,
									Method:  operation,
									Context: "api.paths.operations.v",
									Err:     fmt.Errorf("failed to get operation id: %w", err),
									RawJSON: bv,
								})
							}

							if err := UnmarshalJSONString(op, "consumes", &o.Consumes); err != nil {
								merr = multierror.Append(merr, &errors2.ParserError{
									ID:      ret.ID,
									Route:   path,
									Method:  operation,
									Context: "api.paths.operations.v",
									Err:     fmt.Errorf("failed to get consumes: %w", err),
									RawJSON: bv,
								})
							}

							if err := UnmarshalJSONString(op, "produces", &o.Produces); err != nil {
								merr = multierror.Append(merr, &errors2.ParserError{
									ID:      ret.ID,
									Route:   path,
									Method:  operation,
									Context: "api.paths.operations.v",
									Err:     fmt.Errorf("failed to get produces: %w", err),
									RawJSON: bv,
								})
							}

							if v, ok := op["parameters"]; ok {
								if err := unmarshalToParameters(v, &o.Parameters); err != nil {
									merr = multierror.Append(merr, &errors2.ParserError{
										ID:      ret.ID,
										Route:   path,
										Method:  operation,
										Context: "api.paths.operations.parameters",
										Err:     fmt.Errorf("failed to get parameters: %w", err),
										RawJSON: bv,
									})
								}
							}

							ret.Paths[Path(path)][OperationTypes(operation)] = o
						} else {
							got := reflect.TypeOf(v).Kind()
							merr = multierror.Append(merr, &errors2.ParserError{
								ID:      ret.ID,
								Route:   path,
								Method:  operation,
								Context: "api.paths.operations.v",
								Err:     fmt.Errorf("unexpected type for operation: %s", got),
								RawJSON: bv,
							})
						}
					}
				} else {
					got := reflect.TypeOf(v).Kind()
					merr = multierror.Append(merr, &errors2.ParserError{
						ID:      ret.ID,
						Route:   path,
						Context: "api.paths.operations",
						Err:     fmt.Errorf("unexpected type for operations map: %s", got),
						RawJSON: bv,
					})
				}
			}
		} else {
			got := reflect.TypeOf(v).Kind()
			merr = multierror.Append(merr, &errors2.ParserError{
				ID:      ret.ID,
				Context: "api.paths",
				Err:     fmt.Errorf("unexpected type for path: %s", got),
				RawJSON: bv,
			})
		}
	} else {
		merr = multierror.Append(merr, &errors2.ParserError{
			ID:      ret.ID,
			Context: "api",
			Err:     fmt.Errorf("missing paths"),
			RawJSON: bv,
		})
	}

	return ret, merr.ErrorOrNil()
}

func unmarshalToParameters(params interface{}, dst *[]Parameter) error {
	merr := &multierror.Error{}

	rootbv, err := json.Marshal(params)
	if err != nil {
		log.Trace().Err(err).Msg("failed to encode object")
	}

	switch params.(type) {
	case []interface{}:
		for _, params := range params.([]interface{}) {
			switch params.(type) {
			case []interface{}:
				// nested slices
				for _, v := range params.([]interface{}) {
					// if we hit a nil we should just skippo
					if v == nil {
						merr = multierror.Append(merr, &errors2.ParserError{
							RawJSON: rootbv,
							Context: "param.[].[].v",
							Err:     fmt.Errorf("found unexpected nil"),
						})
					}

					bv, err := json.Marshal(v)
					if err != nil {
						merr = multierror.Append(merr, &errors2.ParserError{
							RawJSON: rootbv,
							Context: "param.[].[].v",
							Err:     fmt.Errorf("failed to marshal object: %w", err),
						})
					}

					var tmp Parameter
					if vv, ok := v.(map[string]interface{}); ok {
						err = json.Unmarshal(bv, &tmp)
						if err != nil {
							merr = multierror.Append(merr, &errors2.ParserError{
								RawJSON: bv,
								Context: "param.[].[].map[string]interface{}",
								Err:     fmt.Errorf("failed to unmarshal param : %w", err),
							})
						}
						*dst = append(*dst, tmp)
					} else {
						got := reflect.TypeOf(vv).Kind()
						merr = multierror.Append(merr, &errors2.ParserError{
							RawJSON: bv,
							Context: "param.[].[].v",
							Err:     fmt.Errorf("unexpected type: %s", got),
						})
					}
				}
			case map[string]interface{}:
				var tmp Parameter
				v := params.(map[string]interface{})
				bv, err := json.Marshal(v)
				if err != nil {
					merr = multierror.Append(merr, &errors2.ParserError{
						RawJSON: rootbv,
						Context: "param.[].v",
						Err:     fmt.Errorf("failed to marshal param: %w", err),
					})
				}

				err = json.Unmarshal(bv, &tmp)
				if err != nil {
					merr = multierror.Append(merr, &errors2.ParserError{
						RawJSON: bv,
						Context: "param.[].v",
						Err:     fmt.Errorf("failed to unmarshal param: %w", err),
					})
				}
				*dst = append(*dst, tmp)
			default:
				got := reflect.TypeOf(params)
				if got == nil {
					merr = multierror.Append(merr, &errors2.ParserError{
						RawJSON: rootbv,
						Context: "param.[].v",
						Err:     fmt.Errorf("unexpected nil type"),
					})
					continue
				}
				gots := got.Kind()
				merr = multierror.Append(merr, &errors2.ParserError{
					RawJSON: rootbv,
					Context: "param.[].v",
					Err:     fmt.Errorf("unexpected type: %s", gots),
				})
			}
		}
	default:
		got := reflect.TypeOf(params).Kind()
		merr = multierror.Append(merr, &errors2.ParserError{
			RawJSON: rootbv,
			Context: "param.v",
			Err:     fmt.Errorf("unexpected type: %s", got),
		})
	}

	return merr.ErrorOrNil()
}

func UnmarshalJSONString(v map[string]interface{}, key string, dest interface{}) error {
	if v, ok := v[key]; ok {
		bv, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal %s", key)
		}

		err = json.Unmarshal(bv, dest)
		if err != nil {
			return &errors2.ParserError{
				RawJSON: bv,
				Context: key,
				Err:     fmt.Errorf("failed to unmarshal: %w", err),
			}
		}
	}
	return nil
}

func GetMapString(v map[string]interface{}, key string) (string, error) {
	bv, err := json.Marshal(v)
	if err != nil {
		log.Trace().Err(err).RawJSON("param", bv).Msg("failed to encode object")
	}

	if vv, ok := v[key]; ok {
		if vv == nil {
			return "", nil
		}
		if sv, ok := vv.(string); ok {
			return sv, nil
		} else {
			got := reflect.TypeOf(vv).Kind()
			return "", fmt.Errorf("unexpected type for url: %s", got)
		}
	}
	return "", &errors2.ParserError{
		RawJSON: bv,
		Context: key,
		Err:     fmt.Errorf("missing field"),
	}
}

// LoadString will load the schema from the provided string
func LoadJSONString(buf string) ([]API, error) {
	return LoadJSONBytes([]byte(buf))
}
