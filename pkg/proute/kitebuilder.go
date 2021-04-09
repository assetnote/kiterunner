package proute

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/assetnote/kiterunner/pkg/errors"
	"github.com/assetnote/kiterunner/pkg/kitebuilder"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/hashicorp/go-multierror"
)

var (
	braceRegex = regexp.MustCompile(`(\{.*\})`)
)

func floatCrumbFromInterface(name string, v interface{}) (FloatCrumb, error) {
	var (
		fixed  = false
		dfloat float64
		err    error
		ret    = FloatCrumb{Name: name, Fixed: fixed, Val: dfloat}
	)
	switch vv := v.(type) {
	case string:
		if vv != "" {
			dfloat, err = strconv.ParseFloat(vv, 64)
			if err != nil {
				return ret, fmt.Errorf("failed to parse float: %s: %w", vv, err)
			}
			ret.Fixed = true
		}
	case float32:
		ret.Fixed = true
		ret.Val = float64(vv)
	case float64:
		ret.Fixed = true
		ret.Val = float64(vv)
	case int:
		ret.Fixed = true
		ret.Val = float64(vv)
	case int32:
		ret.Fixed = true
		ret.Val = float64(vv)
	case int64:
		ret.Fixed = true
		ret.Val = float64(vv)
	}
	return ret, nil
}

// objectCrumbFromSchema will construct an object crumb based off the schema. this operates recursively since an
// OpenAPI Schema can contain multiple nested schemas.
// if the name is empty, we will attempt to deduce the name from the schema name.
// Neither of these are always possible, if all else fails it just becomes a string.
// This handles a Typer interface so it can handle the Parameter and the Schema types, (both are kinda similar)
// This way we don't need to duplicate logic to handle different types. but it might be a bad idea later
func objectCrumbFromSchema(name string, v kitebuilder.Typer) (Crumb, error) {
	if name == "" {
		name = v.GetName()
	}
	ret := ObjectCrumb{
		Name: name,
	}
	var merr *multierror.Error

	switch v.GetType() {
	case "null": // not sure if this is sufficient, to put {} instead of null
		return ObjectCrumb{Name: name}, nil
	case "", "object", "json": // this case also catches the "body" case, but we treat it the same
		ret := ObjectCrumb{
			Name: name,
		}
		if vv := v.GetSchema(); vv != nil && !vv.IsZero() {
			c, err := objectCrumbFromSchema(name, vv)
			if err != nil {
				merr = multierror.Append(merr, &errors.ParserError{
					Err:     fmt.Errorf("failed to create body object: %w", err),
					Context: fmt.Sprintf("%s.Schema", name),
				})
			}
			return c, merr.ErrorOrNil()
		}
		// its some other kind of object. so lets parse out all the fields

		if vv := v.GetAdditionalProperties(); vv != nil && !vv.IsZero() {
			c, err := objectCrumbFromSchema(name, vv)
			if err != nil {
				merr = multierror.Append(merr, &errors.ParserError{
					Err:     fmt.Errorf("failed to create body object: %w", err),
					Context: fmt.Sprintf("%s.AdditionalProperties", name),
				})
			}
			ret.Elements = append(ret.Elements, c)
		}

		for k, vv := range v.GetProperties() {
			c, err := objectCrumbFromSchema(k, &vv)
			if err != nil {
				return ret, &errors.ParserError{
					Err:     fmt.Errorf("failed to parse property: %w", err),
					Context: fmt.Sprintf("%s.properties.%s", name, k),
				}
			}
			ret.Elements = append(ret.Elements, c)
		}

		for _, vv := range v.GetAllOf() {
			c, err := objectCrumbFromSchema("", &vv)
			if err != nil {
				return ret, &errors.ParserError{
					Err:     fmt.Errorf("failed to parse allof: %w", err),
					Context: fmt.Sprintf("%s.allOf", name),
				}
			}
			ret.Elements = append(ret.Elements, c)
		}

		// temporarily disable because its very noisy and very likely
		if false && len(ret.Elements) == 0 {
			return ret, &errors.ParserError{
				Context: name,
				Err:     fmt.Errorf("object with no elements"),
			}
		}

		// we should have a complete or default object at this point
		return ret, nil
	case "array":
		ret := ArrayCrumb{
			Name:    name,
			Element: RandomStringCrumb{Name: name},
		}
		if vv := v.GetSchema(); vv != nil && !vv.IsZero() {
			c, err := objectCrumbFromSchema(name, vv)
			if err != nil {
				return ret, &errors.ParserError{
					Context: name,
					Err:     fmt.Errorf("failed to parse array schema: %w", err),
				}
			}
			ret.Element = c
			return ret, nil
		}

		if v.GetItems() != nil && !v.GetItems().IsZero() {
			v, err := objectCrumbFromSchema("item", v.GetItems())
			if err != nil {
				return ret, &errors.ParserError{
					Err:     fmt.Errorf("failed to parse array items: %w", err),
					Context: fmt.Sprintf("%s", name),
				}
			}
			ret.Element = v
			return ret, nil
		}

		return ret, &errors.ParserError{
			Context: name,
			Err:     fmt.Errorf("array with no elements"),
		}
	case "boolean":
		var (
			fixed = false
			dbool bool
		)
		// 💎🤲
		if dfv, ok := v.GetDefault().(bool); ok {
			dbool = dfv
			fixed = true
		}
		return BoolCrumb{Name: name, Fixed: fixed, Val: dbool}, nil
	case "datetime", "date-time":
		fallthrough // not sure if this is acceptable behaviour but we will do so until told otherwise
	case "date":
		// Most places just acecpt unix. so just pick some time recently.
		return IntCrumb{Name: name, Fixed: true, Val: time.Now().UTC().Add(-1 * time.Hour).Unix()}, nil
	case "file":
		// TODO: implement a better file
		return StaticCrumb{K: name, V: "/etc/passwd"}, nil
	case "float", "float32", "float64":
		c, err := floatCrumbFromInterface(name, v.GetDefault())
		if err != nil {
			merr = multierror.Append(merr, &errors.ParserError{
				Context: name,
				Err:     fmt.Errorf("failed to create float: %w", err),
			})
		}
		return c, merr.ErrorOrNil()
	case "int", "uint", "integer", "number", "numeric", "long", "int32", "int64", "uint32", "uint64":
		// numbers are scuffed, we will fallback on format if there is any.
		switch v.GetFormat() {
		case "float":
			c, err := floatCrumbFromInterface(name, v.GetDefault())
			if err != nil {
				merr = multierror.Append(merr, &errors.ParserError{
					Context: name,
					Err:     fmt.Errorf("failed to create float: %w", err),
				})
			}
			return c, merr.ErrorOrNil()
		default:
			if v.GetDefault() != nil {
				switch tmp := v.GetDefault().(type) {
				// handle the main cases, since we can't generically type it
				case int:
					return IntCrumb{Name: name, Fixed: true, Val: int64(tmp)}, nil
				case int32:
					return IntCrumb{Name: name, Fixed: true, Val: int64(tmp)}, nil
				case int64:
					return IntCrumb{Name: name, Fixed: true, Val: tmp}, nil
				case float32, float64:
					c, err := floatCrumbFromInterface(name, tmp)
					if err != nil {
						merr = multierror.Append(merr, &errors.ParserError{
							Context: name,
							Err:     fmt.Errorf("failed to create float: %w", err),
						})
					}
					return c, merr.ErrorOrNil()
				case string:
					ctmp, err := strconv.ParseInt(tmp, 10, 64)
					if err != nil {
						merr = multierror.Append(merr, &errors.ParserError{
							Context: name,
							Err:     fmt.Errorf("failed to create int: %w", err),
						})
						break
					}
					return IntCrumb{Name: name, Fixed: true, Val: int64(ctmp)}, merr.ErrorOrNil()
				}
			}
			return IntCrumb{Name: name, Min: int64(v.GetMinimum()), Max: int64(v.GetMaximum())}, merr.ErrorOrNil()
		}
	case "uuid":
		return UUIDCrumb{Name: name}, nil
	case "jwttoken":
		fallthrough
	case "string", "byte", "any":
		// there's a whole lot of complex types they shove into type: "string"
		if v.GetPattern() != "" { // if there's a pattern its regex
			ret := RegexStringCrumb{Name: name, Regex: v.GetPattern()}
			if err := ret.Validate(); err != nil {
				merr = multierror.Append(merr, &errors.ParserError{
					Context: fmt.Sprintf("%s.Pattern", name),
					Err:     fmt.Errorf("failed to compile pattern: %w", err),
				})
			}
			return ret, merr
		} else if v.GetDefault() != nil { // if there's a default, lets just take that and stop thinking
			if vv, ok := v.GetDefault().(string); ok {
				return StaticCrumb{K: name, V: vv}, nil
			}

			// we have a non string default variable? not sure how this is possible
			merr = multierror.Append(merr, &errors.ParserError{
				Context: fmt.Sprintf("%s.Default", name),
				Err:     fmt.Errorf("unexpected default type: %v", v.GetDefault()),
			})
			return RandomStringCrumb{Name: name}, merr
		} else {
			f := strings.TrimSpace(strings.ToLower(v.GetFormat()))
			switch f {
			case "ip", "ipv4":
				return StaticCrumb{K: name, V: "1.1.1.1"}, nil

			case "json":
				return StaticCrumb{K: name, V: `{"wtf": "no.pls.why"}`}, nil
			case "binary":
				return StaticCrumb{K: name, V: "00000001"}, nil
			case "duration":
				return StaticCrumb{K: name, V: "1s"}, nil
			case "uuid", "guid":
				return UUIDCrumb{Name: name}, nil
			case "email":
				// TODO: implement proper email crumb
				return StaticCrumb{K: name, V: "lpage@google.com"}, nil
			case "path":
				// TODO: implement proper URL crumb
				return StaticCrumb{K: name, V: "/etc/passwd"}, nil
			case "uri", "url", "link":
				// TODO: implement proper URL crumb
				return StaticCrumb{K: name, V: "http://example.com"}, nil
			case "date", "datetime", "date-time":
				// assumption that date can just be a UTC timestamp
				return StringCrumbCrumb{Name: name, Child: IntCrumb{Name: name, Fixed: true, Val: time.Now().UTC().Add(-1 * time.Hour).Unix()}}, nil
			case "int64", "int32", "int", "uint64", "integer", "long", "int-or-string":
				return StringCrumbCrumb{Name: name, Child: IntCrumb{Name: name}}, nil
			case "float64", "float32", "float":
				return StringCrumbCrumb{Name: name, Child: FloatCrumb{Name: name}}, nil
			case "byte", "bytes", "token":
				return RandomStringCrumb{Name: name, Charset: ASCIIHex}, nil
			case "any", "string", "password": // not sure if byte should be handled this way but suck it
				return RandomStringCrumb{Name: name}, nil
			case "object":
				// TODO: implement object parsing for a string
				fallthrough
			case "array":
				// TODO: implement array parsing for a string
				fallthrough
			default:
				if strings.Contains(f, "date") {
					// assuming that date can just be a utc timestamp. some people specify dd/mm/yyyy but i cbf
					// parsing this into a usable format. thats way too much ML
					return StringCrumbCrumb{Name: name, Child: IntCrumb{Name: name, Fixed: true, Val: time.Now().UTC().Add(-1 * time.Hour).Unix()}}, nil
				} else if v.GetFormat() != "" { // it has a format that we don't know how to handle
					merr = multierror.Append(merr, &errors.ParserError{
						Context: name,
						Err:     fmt.Errorf("unexpected string format: %s", v.GetFormat()),
					})
				}
				return RandomStringCrumb{Name: name}, merr.ErrorOrNil()
			}
		}
		return RandomStringCrumb{Name: name}, nil
		// %+v map[:1615 binary:5 bytes:10 date:21 date-format:1 date-time:47 datetime:6 duration:14
		// email:10 locale:2 markdown:3 password:4 uri:8 uuid:6]
	default:
		return ret, fmt.Errorf("unexpected type %s", v.GetType())
		// %+v map[:376 array:283 boolean:199 integer:250 jwttoken:1 number:72 object:68 string:1752]
	}
	// TODO: Handle enums

	return ret, fmt.Errorf("unexpected type %s", v.GetType())
}

func FromKitebuilderAPIs(src []kitebuilder.API) ([]API, error) {
	var merr *multierror.Error
	ret := make([]API, 0)
	for _, v := range src {
		tmp, err := FromKitebuilderAPI(v)
		if err != nil {
			multierror.Append(merr, err)
		}
		ret = append(ret, tmp)
	}
	return ret, merr.ErrorOrNil()
}

// FromKitebuilderAPI will convert an kitebuilder API to a proute API. Errors are swallowed and printed to stdout because
// we hate ourselves other developers who try to use this. I can't believe I have to parse json types into usable data
// future readers beware. https://www.youtube.com/watch?v=LZWnM_u7Vfg
func FromKitebuilderAPI(src kitebuilder.API) (API, error) {
	ret := NewAPI(src.ID, src.URL)
	var (
		merr *multierror.Error
		err  error
	)
	log.Trace().Str("url", ret.URL).Str("id", ret.ID).Msg("parsing kitebuilder api to proute api")

	// Parse the security definitions and their types
	for name, v := range src.SecurityDefinitions {
		// we prefer the element name
		keyName := v.Name
		// fallback onto the object name
		if keyName == "" {
			keyName = name
		}
		if keyName == "" {
			keyName = "Authorization"
			merr = multierror.Append(merr, &errors.ParserError{
				ID:      src.ID,
				Context: fmt.Sprintf("api.securityDefinitions.[%s]", name),
				Err:     fmt.Errorf("no valid name"),
			})
		} else if strings.Contains(keyName, " ") {
			// if we have some weird case where they mess around with the name. try and normalize it
			// spaces is not a good normalization
			// TODO: use proper charset normalization
			merr = multierror.Append(merr, &errors.ParserError{
				ID:      src.ID,
				Context: fmt.Sprintf("api.securityDefinitions.[%s]", name),
				Err:     fmt.Errorf("invalid valid security definition name: %s", keyName),
			})
			keyName = strings.Replace(keyName, " ", "-", -1)
		}

		var header Crumb
		// What are they
		switch strings.ToLower(v.Type) {
		case "apikey":
			header = RandomStringCrumb{Name: keyName, Length: 32, Charset: ASCIIHex}
		case "basic":
			header = BasicAuthCrumb{Name: keyName, Random: true}
		default:
			merr = multierror.Append(merr, &errors.ParserError{
				ID:      src.ID,
				Context: fmt.Sprintf("api.securityDefinitions.[%s].Type", name),
				Err:     fmt.Errorf("unexpected type: %s", v.Type),
			})
			continue
		}

		// Where do they go
		switch strings.ToLower(v.In) {
		case "", "header":
			ret.HeaderCrumbs = append(ret.HeaderCrumbs, header)
		case "body":
			ret.BodyCrumbs = append(ret.BodyCrumbs, header)
		case "query":
			ret.QueryCrumbs = append(ret.QueryCrumbs, header)
		case "cookie":
			ret.CookieCrumbs = append(ret.CookieCrumbs, header)
		default:
			merr = multierror.Append(merr, &errors.ParserError{
				ID:      src.ID,
				Context: fmt.Sprintf("api.securityDefinitions.[%s].In", name),
				Err:     fmt.Errorf("unexpected destination: %s", v.In),
			})
			continue
		}
	}

	for path, ops := range src.Paths {
		for method, op := range ops {
			var r Route
			r.Method = string(method)
			r.TemplatePath = string(path)

			for _, v := range op.Consumes {
				// TODO: normalize content type
				r.ContentType = append(r.ContentType, ContentType(v))
			}

			// build out the potential parameters so we can backref them when looking at the path
			paramInMap := make(map[string]string)

			for idx, v := range op.Parameters {
				var c Crumb
				// dedupe our parameters in the event the input is bad
				// lets not care about dupes because we want to stay true to the input
				if prevIn, ok := paramInMap[v.Name]; false && ok {
					// skip dupes
					// ok lets not skip dupes so we have an accurate representation of what comes in
					// on the caller to ensure there are no dupes
					if prevIn == v.In {
						continue
					}
					// different types, but we'll include them all anyway
					merr = multierror.Append(merr, &errors.ParserError{
						ID:      src.ID,
						Route:   string(path),
						Method:  string(method),
						Context: fmt.Sprintf("api.path.method.parameters.[%d].Name", idx),
						Err:     fmt.Errorf("duplicate keys found: %s", v.Name),
					})
				}
				paramInMap[v.Name] = v.In

				c, err = objectCrumbFromSchema(v.Name, v)
				if err != nil {
					merr = multierror.Append(merr, &errors.ParserError{
						ID:      src.ID,
						Route:   string(path),
						Method:  string(method),
						Context: fmt.Sprintf("api.path.method.parameters.[%d]", idx),
						Err:     fmt.Errorf("failed to parse parameter: %w", err),
					})
				}

				// this is a bad case, since we've gone through our huge as fuck switch statement and we havent
				// found an appropriate type. Probably should notify the user or something
				if c == nil {
					log.Fatal().Str("type", v.Type).Str("name", v.Name).Str("In", v.In).Msg("failed to assign type after switch")
				}

				// handle content-type explicitly because not everyone puts it in the right place
				if strings.ToLower(v.Name) == "content-type" {
					found := false
					if strv, ok := v.Default.(string); ok && strv != "" {
						found = true
						r.ContentType = append(r.ContentType, ContentType(strv))
					}
					if strv, ok := v.Example.(string); ok && strv != "" {
						found = true
						r.ContentType = append(r.ContentType, ContentType(strv))
					}
					for _, vv := range v.Enum {
						if strv, ok := vv.(string); ok && strv != "" {
							found = true
							r.ContentType = append(r.ContentType, ContentType(strv))
						}
					}
					if found {
						continue
					}
				}

				// Now we determine where this parameter goes..
				switch strings.ToLower(v.In) {
				case "body", "formdata":
					r.BodyCrumbs = append(r.BodyCrumbs, c)
				case "header", "headers":
					r.HeaderCrumbs = append(r.HeaderCrumbs, c)
				case "path", "modelbinding":
					r.PathCrumbs = append(r.PathCrumbs, c)
				case "parameter", "query":
					r.QueryCrumbs = append(r.QueryCrumbs, c)
				case "":
					fallthrough
				default:
					merr = multierror.Append(merr, &errors.ParserError{
						ID:      src.ID,
						Route:   string(path),
						Method:  string(method),
						Context: fmt.Sprintf("api.path.method.parameters.[%d]", idx),
						Err:     fmt.Errorf("unhandled location: %s", v.In),
					})
					continue
					// some data from an early revision
					// %+v map[:7 body:9860 formdata:8849 header:8857 headers:1 modelbinding:2 parameter:5 path:34442 query:40052]
				}
			}

			ret.Routes = append(ret.Routes, r)
		}
	}

	return ret, merr.ErrorOrNil()
}

func (a APIS) ToKiteBuilderAPIS() ([]kitebuilder.API, error) {
	ret := make([]kitebuilder.API, 0)
	for _, v := range a {
		tmp, err := v.ToKitebuilderAPI()
		if err != nil {
			return nil, err
		}
		ret = append(ret, tmp)
	}
	return ret, nil
}

func CrumbToSchema(c Crumb) kitebuilder.Schema {
	p := kitebuilder.Schema{
		Name: c.Key(),
	}
	switch c.(type) {
	case UUIDCrumb:
		p.Type = "uuid"
	case StaticCrumb:
		p.Type = "string"
		p.Default = c.Value()
	case IntCrumb:
		p.Type = "integer"
		if c.(IntCrumb).Fixed {
			p.Default = c.(IntCrumb).Val
		}
		if c.(IntCrumb).Min != 0 {
			p.Min = float64(c.(IntCrumb).Min)
		}
		if c.(IntCrumb).Max != 0 {
			p.Max = float64(c.(IntCrumb).Max)
		}
	case BoolCrumb:
		p.Type = "bool"
		if c.(BoolCrumb).Fixed {
			p.Default = c.(BoolCrumb).Val
		}
	case FloatCrumb:
		p.Type = "float"
		if c.(FloatCrumb).Fixed {
			p.Default = c.(FloatCrumb).Val
		}
	case RandomStringCrumb:
		p.Type = "string"
	case RegexStringCrumb:
		p.Type = "string"
		p.Pattern = c.(RegexStringCrumb).Regex
	case BasicAuthCrumb:
		p.Type = "string"
	case ObjectCrumb:
		p.Type = "object"
		p.Properties = make(map[string]kitebuilder.Schema)
		for _, v := range c.(ObjectCrumb).Elements {
			p.Properties[v.Key()] = CrumbToSchema(v)
		}
	case ArrayCrumb:
		p.Type = "Array"
		tmp := CrumbToSchema(c.(ArrayCrumb).Element)
		p.Items = &tmp
	case StringCrumbCrumb:
		p.Type = "string"
		p.Default = c.Value()
	}
	return p
}

func CrumbToParameter(c Crumb) kitebuilder.Parameter {
	p := kitebuilder.Parameter{
		Name: c.Key(),
	}

	switch c.(type) {
	case UUIDCrumb:
		p.Type = "uuid"
	case StaticCrumb:
		p.Type = "string"
		p.Default = c.Value()
	case IntCrumb:
		p.Type = "integer"
		if c.(IntCrumb).Fixed {
			p.Default = c.(IntCrumb).Val
		}
		if c.(IntCrumb).Min != 0 {
			p.Minimum = float64(c.(IntCrumb).Min)
		}
		if c.(IntCrumb).Max != 0 {
			p.Maximum = float64(c.(IntCrumb).Max)
		}
	case BoolCrumb:
		p.Type = "bool"
		if c.(BoolCrumb).Fixed {
			p.Default = c.(BoolCrumb).Val
		}
	case FloatCrumb:
		p.Type = "float"
		if c.(FloatCrumb).Fixed {
			p.Default = c.(FloatCrumb).Val
		}
	case RandomStringCrumb:
		p.Type = "string"
	case RegexStringCrumb:
		p.Type = "string"
		p.Pattern = c.(RegexStringCrumb).Regex
	case BasicAuthCrumb:
		p.Type = "basic"
	case ObjectCrumb:
		p.Type = "object"
		tmp := CrumbToSchema(c)
		p.Schema = &tmp
	case ArrayCrumb:
		p.Type = "Array"
		tmp := CrumbToSchema(c)
		p.Items = &tmp
	case StringCrumbCrumb:
		p.Type = "string"
		p.Default = c.Value()
	}

	return p
}

func CrumbToSecurityDefinition(c Crumb) kitebuilder.SecurityDefinition {
	p := kitebuilder.SecurityDefinition{
		Name: c.Key(),
		Type: "apikey",
	}

	switch c.(type) {
	case UUIDCrumb:
		p.Type = "apikey"
	case StaticCrumb:
		p.Type = "apikey"
	case RandomStringCrumb:
		p.Type = "apikey"
	case RegexStringCrumb:
		p.Type = "apikey"
	case BasicAuthCrumb:
		p.Type = "basic"
	}

	return p
}

func (a API) ToKitebuilderAPI() (kitebuilder.API, error) {
	ret := kitebuilder.API{
		ID:                  a.ID,
		URL:                 a.URL,
		SecurityDefinitions: make(map[string]kitebuilder.SecurityDefinition),
		Paths:               make(map[kitebuilder.Path]kitebuilder.Operations),
	}

	for _, v := range a.QueryCrumbs {
		p := CrumbToSecurityDefinition(v)
		p.In = "query"
		ret.SecurityDefinitions[p.Name] = p
	}

	for _, v := range a.BodyCrumbs {
		p := CrumbToSecurityDefinition(v)
		p.In = "body"
		ret.SecurityDefinitions[p.Name] = p
	}

	for _, v := range a.HeaderCrumbs {
		p := CrumbToSecurityDefinition(v)
		p.In = "header"
		ret.SecurityDefinitions[p.Name] = p
	}

	for _, v := range a.CookieCrumbs {
		p := CrumbToSecurityDefinition(v)
		p.In = "cookie"
		ret.SecurityDefinitions[p.Name] = p
	}

	for _, route := range a.Routes {
		ops, ok := ret.Paths[kitebuilder.Path(route.TemplatePath)]
		if !ok {
			ret.Paths[kitebuilder.Path(route.TemplatePath)] = make(kitebuilder.Operations)
			ops = ret.Paths[kitebuilder.Path(route.TemplatePath)]
		}

		// add consumes
		op := ops[kitebuilder.OperationTypes(route.Method)]
		for _, v := range route.ContentType {
			op.Consumes = append(op.Consumes, kitebuilder.ContentType(v))
		}

		// add parameters
		for _, v := range route.QueryCrumbs {
			p := CrumbToParameter(v)
			p.In = "query"
			op.Parameters = append(op.Parameters, p)
		}
		for _, v := range route.BodyCrumbs {
			p := CrumbToParameter(v)
			p.In = "body"
			op.Parameters = append(op.Parameters, p)
		}
		for _, v := range route.HeaderCrumbs {
			p := CrumbToParameter(v)
			p.In = "header"
			op.Parameters = append(op.Parameters, p)
		}
		for _, v := range route.PathCrumbs {
			p := CrumbToParameter(v)
			p.In = "path"
			op.Parameters = append(op.Parameters, p)
		}

		ret.Paths[kitebuilder.Path(route.TemplatePath)][kitebuilder.OperationTypes(route.Method)] = op
	}

	return ret, nil
}
