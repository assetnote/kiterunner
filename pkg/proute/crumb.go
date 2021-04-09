package proute

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"mime/multipart"
	"net/url"
	"strconv"
	"strings"

	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/beevik/etree"
	"github.com/francoispqt/gojay"
	"github.com/google/uuid"
	"github.com/lucasjones/reggen"
)

const (
	DefaultFormDataBoundary = "hahahahahformboundaryhahahaha"
)

var (
	AllCrumbs = []Crumb{
		UUIDCrumb{},
		StaticCrumb{},
		IntCrumb{},
		BoolCrumb{},
		FloatCrumb{},
		RandomStringCrumb{},
		RegexStringCrumb{},
		BasicAuthCrumb{},
		ArrayCrumb{},
		ObjectCrumb{},
		StringCrumbCrumb{},
	}
)

// Crumb is a piece of a route, can be dynamically rendered or not
type Crumb interface {
	Value(...CrumbOption) string // we pass opts so at runtime we can determine if we want to configure any behaviour
	RawValue(...CrumbOption) interface{}
	Key() string
	protoCrumb() *ProtoCrumb
}

// CrumbOptions provides options that can be used by any crumb when generating the values
// TODO: add rng as a source from the crumb options
type CrumbOptions struct {
	ContentType      ContentType
	FormDataBoundary string
	IsChild          bool
	Random           bool
}
type CrumbOption func(o *CrumbOptions)

func DefaultCrumbOptions() *CrumbOptions {
	return &CrumbOptions{
		ContentType:      ContentTypeFormEncoded,
		FormDataBoundary: DefaultFormDataBoundary,
	}
}

func NewCrumbOptions(opts ...CrumbOption) *CrumbOptions {
	o := DefaultCrumbOptions()
	for _, v := range opts {
		v(o)
	}
	return o
}

func CrumbOptContentType(v ContentType) CrumbOption {
	return func(o *CrumbOptions) {
		o.ContentType = v
	}
}

func CrumbOptFormDataBoundary(v string) CrumbOption {
	return func(o *CrumbOptions) {
		o.FormDataBoundary = v
	}
}

func CrumbOptIsChild(v bool) CrumbOption {
	return func(o *CrumbOptions) {
		o.IsChild = v
	}
}

func (p UUIDCrumb) Key() string {
	return p.Name
}

func (p UUIDCrumb) Value(...CrumbOption) string {
	return uuid.New().String()
}

func (p UUIDCrumb) RawValue(...CrumbOption) interface{} {
	return uuid.New().String()
}

func (p UUIDCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_UuidCrumb{UuidCrumb: &p}}
}

func (s StaticCrumb) Key() string {
	return s.K
}

func (s StaticCrumb) Value(...CrumbOption) string {
	return s.V
}

func (s StaticCrumb) RawValue(...CrumbOption) interface{} {
	return s.V
}

func (p StaticCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_StaticCrumb{StaticCrumb: &p}}
}

func (i IntCrumb) Value(opts ...CrumbOption) string {
	return strconv.FormatInt(i.RawValue(opts...).(int64), 10)
}

func (i IntCrumb) Key() string {
	return i.Name
}

func (i IntCrumb) RawValue(...CrumbOption) interface{} {
	if i.Fixed {
		return i.Val
	}
	if i.Min >= i.Max { // avoid panicking
		i.Min = 0
	}
	// if our max and min aren't valid, pick some reasonable value (not 64bit)
	if !(i.Max > i.Min) {
		i.Max = 1000000
		i.Min = 0
	}
	return rand.Int63n(i.Max-i.Min) + i.Min
}

func (p IntCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_IntCrumb{IntCrumb: &p}}
}

func (b BoolCrumb) Value(opts ...CrumbOption) string {
	return strconv.FormatBool(b.RawValue(opts...).(bool))
}

func (b BoolCrumb) Key() string {
	return b.Name
}

func (b BoolCrumb) RawValue(...CrumbOption) interface{} {
	if b.Fixed {
		return b.Val
	}
	return rand.Intn(1) == 1
}

func (p BoolCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_BoolCrumb{BoolCrumb: &p}}
}

func (i FloatCrumb) Value(opts ...CrumbOption) string {
	return strconv.FormatFloat(i.RawValue(opts...).(float64), 'g', -1, 64)
}

func (i FloatCrumb) Key() string {
	return i.Name
}

func (i FloatCrumb) RawValue(...CrumbOption) interface{} {
	if i.Fixed {
		return i.Val
	}
	return rand.Float64()
}

func (p FloatCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_FloatCrumb{FloatCrumb: &p}}
}

func (s RandomStringCrumb) Key() string {
	return s.Name
}

// Value(...CrumbOption) will return a random string value. For maximum compatability, this return a number
func (s RandomStringCrumb) Value(...CrumbOption) string {
	if len(s.Charset) == 0 {
		s.Charset = ASCIINum
	}
	if s.Length == 0 {
		s.Length = 8 // default length
	}
	return RandomString(nil, s.Charset, s.Length)
}

func (s RandomStringCrumb) RawValue(opts ...CrumbOption) interface{} {
	return s.Value(opts...)
}

func (p RandomStringCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_RandomStringCrumb{RandomStringCrumb: &p}}
}

func (r RegexStringCrumb) Validate() error {
	_, err := reggen.Generate(r.Regex, 1)
	return err
}

func (r RegexStringCrumb) Value(...CrumbOption) string {
	str, err := reggen.Generate(r.Regex, 10)
	if err != nil {
		log.Debug().Str("regex", r.Regex).Err(err).Str("name", r.Name).Msg("failed to generate regex string crumb value")
		str = "1" // use some simple default thats both a string and a numeric
	}
	return str
}

func (r RegexStringCrumb) RawValue(opts ...CrumbOption) interface{} {
	return r.Value(opts...)
}

func (r RegexStringCrumb) Key() string {
	return r.Name
}

func (p RegexStringCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_RegexStringCrumb{RegexStringCrumb: &p}}
}

func (b BasicAuthCrumb) Value(...CrumbOption) string {
	var (
		user = b.User
		pass = b.Password
	)

	if b.Random {
		if user == "" {
			user = RandomString(nil, ASCIIAlphaNum, 16)
		}
		if pass == "" {
			pass = RandomString(nil, ASCIIAlphaNum, 16)
		}
	}
	unencoded := fmt.Sprintf("%s:%s", user, pass)
	return fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(unencoded)))
}

func (b BasicAuthCrumb) RawValue(opts ...CrumbOption) interface{} {
	return b.Value(opts...)
}

func (b BasicAuthCrumb) Key() string {
	return "Authorization"
}

func (p BasicAuthCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_BasicAuthCrumb{BasicAuthCrumb: &p}}
}

// MarshalXMLCrumb will create an XML object containing just that crumb. The object name is set to the crumb name
// and the child value is set to the child value of the crumb
func MarshalXMLCrumb(c Crumb, opts ...CrumbOption) string {
	co := NewCrumbOptions(opts...)
	doc := etree.NewDocument()
	// append the header if we're the root
	if !co.IsChild {
		doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
	}
	root := doc.CreateElement(c.Key())
	root.SetText(c.Value(append(opts, CrumbOptIsChild(true))...))
	ret, err := doc.WriteToString()
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal xml")
	}
	return ret
}

type XMLer interface {
	AddChild(t etree.Token)
	SetText(string)
	CreateElement(tag string) *etree.Element
}

type ObjectCrumb struct {
	Name     string
	Elements []Crumb
}

// XMLAddChildCrumb is a dirty method of building an XML document using strings
// if the child is to be an object, we need to unmarshal it to get a etree elemtn we can append
// otherwise it'll encode our xml object. Fuck this is stupid
func XMLAddChildCrumb(xml XMLer, c Crumb, nameOverride string, opts ...CrumbOption) error {
	switch c.(type) {
	case ObjectCrumb, ArrayCrumb:
		// these xml objects already have a Element created so don't create one to nest
		childdoc := etree.NewDocument()
		// this is dirty and painful. there's no good way to build XML. So we marshal and unmarshal the XML
		// to create a child element
		childstr := c.Value(append(opts, CrumbOptIsChild(true))...)
		err := childdoc.ReadFromString(childstr)
		if err != nil {
			log.Error().Str("childstr", childstr).Err(err).Msg("failed to unmarshal child")
			return err
		}
		for _, v := range childdoc.Element.ChildElements() {
			xml.AddChild(v)
		}
	default:
		// otherwise these are bare values, so they need to have an element created to nest in
		key := c.Key()
		if nameOverride != "" {
			key = nameOverride
		}
		child := xml.CreateElement(key)
		child.SetText(c.Value(append(opts, CrumbOptIsChild(true))...))
	}
	return nil
}

func (o ObjectCrumb) Value(opts ...CrumbOption) string {
	co := NewCrumbOptions(opts...)

	switch co.ContentType {
	case ContentTypeXML:
		doc := etree.NewDocument()
		// append the header if we're the root
		if !co.IsChild {
			doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
		}
		root := doc.CreateElement(o.Name)

		for _, v := range o.Elements {
			err := XMLAddChildCrumb(root, v, "", opts...)
			if err != nil {
				log.Error().Err(err).Msg("failed to unmarshal child XML for object")
			}
		}
		ret, err := doc.WriteToString()
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal xml")
		}
		return ret
	case ContentTypeJSON:
		b := strings.Builder{}
		enc := gojay.BorrowEncoder(&b)
		defer enc.Release()

		if err := enc.Encode(o); err != nil {
			log.Error().Err(err).Msg("failed to marshal object")
		}
		return b.String()
	case ContentTypeFormData:
		// only produce full formdata if we're the root
		if !co.IsChild {
			b := &strings.Builder{}
			w := multipart.NewWriter(b)
			if err := w.SetBoundary(co.FormDataBoundary); err != nil {
				log.Error().Err(err).Str("context", "objectCrumb Formdata Value").Msg("failed to set boundary")
			}
			for _, v := range o.Elements {
				w.WriteField(v.Key(), v.Value(append(opts, CrumbOptIsChild(true))...))
			}
			if err := w.Close(); err != nil {
				log.Error().Err(err).Str("context", "objectCrumb Formdata Value").Msg("failed to close multipart")
			}
			return b.String()
		}
		fallthrough
	case ContentTypeFormEncoded:
		fallthrough
	default:
		// there's a weird case if the element is an array type, and its children are objects
		// we will get json objects in the array, but i guess thats to be expected
		// similarly, if the child is an object type, we will get nested KV instead of json types
		// its weird, what if they expect a json body as part of a post body? what happens? where is god?
		data := url.Values{}
		for _, v := range o.Elements {
			data.Set(v.Key(), v.Value(append(opts, CrumbOptIsChild(true))...))
		}
		return data.Encode()
	}
	return ""
}

func (o ObjectCrumb) Key() string {
	return o.Name
}

func (o ObjectCrumb) RawValue(opts ...CrumbOption) interface{} {
	return o
}

func (o ObjectCrumb) MarshalJSONObject(enc *gojay.Encoder) {
	for _, v := range o.Elements {
		enc.AddInterfaceKey(v.Key(), v.RawValue())
	}
}

func (o ObjectCrumb) IsNil() bool {
	return o.Name == "" && len(o.Elements) == 0
}

func (p ObjectCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_ObjectCrumb{ObjectCrumb: p.protoObjectCrumb()}}
}

func (p ObjectCrumb) protoObjectCrumb() *ProtoObjectCrumb {
	return &ProtoObjectCrumb{
		Name:     p.Name,
		Elements: FromCrumbs(p.Elements),
	}
}

type ArrayCrumb struct {
	Name    string
	Element Crumb
}

func (a ArrayCrumb) Value(opts ...CrumbOption) string {
	co := NewCrumbOptions(opts...)
	e := a.Element

	switch co.ContentType {
	case ContentTypeXML:
		doc := etree.NewDocument()
		// only append the header if we're the root
		if !co.IsChild {
			doc.CreateProcInst("xml", `version="1.0" encoding="UTF-8"`)
		}
		root := doc.CreateElement(a.Name)
		// we need to have a name for this child because XML is dumb
		// and not all swagger specs will specify this. so we'll use the parent as the example
		// might be wrong per the spec, but its something that should parse the xml parser
		childKey := e.Key()
		if childKey == "" {
			childKey = a.Name
		}

		if err := XMLAddChildCrumb(root, e, childKey, opts...); err != nil {
			log.Error().Err(err).Msg("failed to unmarshal child XML for array")
		}

		ret, err := doc.WriteToString()
		if err != nil {
			log.Error().Err(err).Msg("failed to marshal xml")
		}
		return ret
	case ContentTypeJSON:
		b := strings.Builder{}
		enc := gojay.BorrowEncoder(&b)
		defer enc.Release()

		if err := enc.Encode(a); err != nil {
			log.Error().Err(err).Msg("failed to marshal object")
		}
		return b.String()
	case ContentTypeFormData:
		// only encode the entire body if we're the root. otherwise its just form encoded data
		if !co.IsChild {
			b := &strings.Builder{}
			w := multipart.NewWriter(b)
			if err := w.SetBoundary(co.FormDataBoundary); err != nil {
				log.Error().Err(err).Str("context", "objectCrumb Formdata Value").Msg("failed to set boundary")
			}
			childKey := e.Key()
			if childKey == "" {
				childKey = a.Name
			}

			w.WriteField(childKey, e.Value(append(opts, CrumbOptIsChild(true))...))
			if err := w.Close(); err != nil {
				log.Error().Err(err).Str("context", "objectCrumb Formdata Value").Msg("failed to close multipart")
			}
			return b.String()
		}
		fallthrough
	case ContentTypeFormEncoded:
		fallthrough
	default:
		// there's a weird case if the element is an array type, and its children are objects
		// we will get json objects in the array, but i guess thats to be expected
		// similarly, if the child is an object type, we will get nested KV instead of json types
		// its weird, what if they expect a json body as part of a post body? what happens? where is god?

		// We also need to similarly handle where we need a child key name. Not sure what we're supposed to expect
		// but lets just reuse our name if the child is blank
		childKey := e.Key()
		if childKey == "" {
			childKey = a.Name
		}

		data := url.Values{}
		data.Set(childKey, e.Value(append(opts, CrumbOptIsChild(true))...))
		return data.Encode()
	}
}

func (a ArrayCrumb) RawValue(opts ...CrumbOption) interface{} {
	return a
}

func (a ArrayCrumb) Key() string {
	return a.Name
}

func (a ArrayCrumb) MarshalJSONArray(enc *gojay.Encoder) {
	if a.Element == nil {
		return
	}
	enc.AddInterface(a.Element.RawValue())
}

func (a ArrayCrumb) IsNil() bool {
	return a.Name == "" && a.Element == nil
}

func (p ArrayCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_ArrayCrumb{ArrayCrumb: p.protoArrayCrumb()}}
}

func (p ArrayCrumb) protoArrayCrumb() *ProtoArrayCrumb {
	return &ProtoArrayCrumb{
		Name:    p.Name,
		Element: p.Element.protoCrumb(),
	}
}

type StringCrumbCrumb struct {
	Name  string
	Child Crumb
}

func (s StringCrumbCrumb) Value(option ...CrumbOption) string {
	tmp := s.Child.Value(option...)
	v, err := json.Marshal(tmp)
	if err != nil {
		log.Fatal().Str("v", tmp).Err(err).Msg("failed to encode json string")
	}

	return string(v)
}

func (s StringCrumbCrumb) Key() string {
	return s.Name
}

func (s StringCrumbCrumb) RawValue(option ...CrumbOption) interface{} {
	return s.Value(option...)
}

func (p StringCrumbCrumb) protoCrumb() *ProtoCrumb {
	return &ProtoCrumb{&ProtoCrumb_StringCrumbCrumb{StringCrumbCrumb: p.protoStringCrumbCrumb()}}
}

func (p StringCrumbCrumb) protoStringCrumbCrumb() *ProtoStringCrumbCrumb {
	return &ProtoStringCrumbCrumb{
		Name:  p.Name,
		Child: p.Child.protoCrumb(),
	}
}
