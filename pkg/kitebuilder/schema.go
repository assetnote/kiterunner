package kitebuilder

import (
	"reflect"
)

type Path string

type API struct {
	ID                  string                        `json:"ksuid,omitempty"`
	URL                 string                        `json:"url"`
	SecurityDefinitions map[string]SecurityDefinition `json:"securityDefinitions"`
	Paths               map[Path]Operations           `json:"paths"`
}

type SecurityDefinition struct {
	In   string `json:"in"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type OperationTypes string

var (
	GET     OperationTypes = "get"
	DELETE  OperationTypes = "delete"
	HEAD    OperationTypes = "head"
	OPTIONS OperationTypes = "options"
	PATCH   OperationTypes = "patch"
	POST    OperationTypes = "post"
	PUT     OperationTypes = "put"
)

// Operations is a map of the verb to the operation data. technically we can have "parameters" here, but then we have
// dual types for this one object which is fucked
type Operations map[OperationTypes]Operation

type ContentType string

type Operation struct {
	Description string        `json:"description,omitempty"`
	OperationID string        `json:"operationId,omitempty"`
	Parameters  []Parameter   `json:"parameter,omitempty"`
	Consumes    []ContentType `json:"consume,omitempty"`
	Produces    []ContentType `json:"produce,omitempty"`
}

// Typer provides an interface that makes parameter and schema generic. It provides a list of fields
// that allow both to be accessed as the same kind of object when reconstructing proute crumbs
// It doesnt totally fit over both, but its close/good enough with nillable types for determining whats
// possible and whats not possible
type Typer interface {
	GetType() string
	GetName() string
	GetExample() interface{}
	GetFormat() string
	GetPattern() string
	GetDefault() interface{}

	GetIn() string // exclusive to parameter, so a schema should return empty

	GetMinimum() float64
	GetMaximum() float64

	GetProperties() map[string]Schema
	GetAdditionalProperties() *Schema
	GetSchema() *Schema
	GetItems() *Schema
	GetAllOf() []Schema
}

type Parameter struct {
	Description string      `json:"description,omitempty"`
	In          string      `json:"in,omitempty"`
	Name        string      `json:"name,omitempty"`
	Required    interface{} `json:"required,omitempty"`
	// some schemas use "true" instead of true. So we decided to ignore required entirely anyway

	Schema *Schema `json:"schema,omitempty"` // if in == "body"

	// if in != "body"
	Type            string      `json:"type,omitempty"`
	AllowEmptyValue bool        `json:"allowEmptyValue,omitempty"`
	Pattern         string      `json:"pattern,omitempty"`
	Format          string      `json:"format,omitempty"`
	Example         interface{} `json:"example,omitempty"`
	// could really be anything. we should handle this with care.

	Minimum   float64 `json:"minimum,omitempty"`
	Maximum   float64 `json:"maximum,omitempty"`
	MaxLength uint64  `json:"maxLength,omitempty"`
	MaxItems  uint64  `json:"maxItems,omitempty"`
	MinLength uint64  `json:"minLength,omitempty"`
	MinItems  uint64  `json:"minItems,omitempty"`

	Enum    []interface{} `json:"enum,omitempty"` // can be [1,2,3] ["a", "b", "c"]
	Default interface{}   `json:"default,omitempty"`

	Items *Schema `json:"items,omitempty"`
}

var _ Typer = &Parameter{}

func (p Parameter) GetType() string                  { return p.Type }
func (p Parameter) GetName() string                  { return p.Name }
func (p Parameter) GetIn() string                    { return p.In }
func (p Parameter) GetExample() interface{}          { return p.Example }
func (p Parameter) GetMinimum() float64              { return p.Minimum }
func (p Parameter) GetMaximum() float64              { return p.Maximum }
func (p Parameter) GetDefault() interface{}          { return p.Default }
func (p Parameter) GetFormat() string                { return p.Format }
func (p Parameter) GetPattern() string               { return p.Pattern }
func (p Parameter) GetProperties() map[string]Schema { return nil }
func (p Parameter) GetSchema() *Schema               { return p.Schema }
func (p Parameter) GetAdditionalProperties() *Schema { return nil }
func (p Parameter) GetItems() *Schema                { return p.Items }
func (p Parameter) GetAllOf() []Schema               { return nil }

type Schema struct {
	Properties       map[string]Schema `json:"properties,omitempty"`
	Type             string            `json:"type,omitempty" yaml:"type,omitempty"`
	Title            string            `json:"title,omitempty" yaml:"title,omitempty"`
	Format           string            `json:"format,omitempty" yaml:"format,omitempty"`
	Description      string            `json:"description,omitempty" yaml:"description,omitempty"`
	Enum             []interface{}     `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default          interface{}       `json:"default,omitempty" yaml:"default,omitempty"`
	Example          interface{}       `json:"example,omitempty" yaml:"example,omitempty"`
	Name             string            `json:"name,omitempty"`
	CollectionFormat string            `json:"collectionFormat,omitempty"`

	// Array-related, here for struct compactness
	UniqueItems bool `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	// Number-related, here for struct compactness
	ExclusiveMin bool `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	ExclusiveMax bool `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	// Properties
	Nullable bool `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	ReadOnly bool `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	// can be either []string or bool. We don't actually pay attention to this field
	WriteOnly       bool        `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
	AllowEmptyValue bool        `json:"allowEmptyValue,omitempty" yaml:"allowEmptyValue,omitempty"`
	XML             interface{} `json:"xml,omitempty" yaml:"xml,omitempty"`
	Deprecated      bool        `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	// Number
	Min        float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Max        float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	MultipleOf float64 `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`

	// String
	MinLength uint64 `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	MaxLength uint64 `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	Pattern   string `json:"pattern,omitempty" yaml:"pattern,omitempty"`

	// Array
	MinItems             uint64  `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	MaxItems             uint64  `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	Items                *Schema `json:"items,omitempty" yaml:"items,omitempty"`
	AdditionalProperties *Schema `json:"additional_properties,omitempty"`

	// Object
	Required interface{} `json:"required,omitempty" yaml:"required,omitempty"`
	// can be either []string or bool. We don't actually pay attention to this field

	MinProps uint64   `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	MaxProps uint64   `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	AllOf    []Schema `json:"allOf,omitempty"`
}

var _ Typer = &Schema{}

func (s Schema) GetType() string                  { return s.Type }
func (s Schema) GetName() string                  { return s.Name }
func (s Schema) GetIn() string                    { return "" }
func (s Schema) GetExample() interface{}          { return s.Example }
func (s Schema) GetMinimum() float64              { return s.Min }
func (s Schema) GetMaximum() float64              { return s.Max }
func (s Schema) GetDefault() interface{}          { return s.Default }
func (s Schema) GetFormat() string                { return s.Format }
func (s Schema) GetPattern() string               { return s.Pattern }
func (s Schema) GetProperties() map[string]Schema { return s.Properties }
func (s Schema) GetSchema() *Schema               { return nil }
func (s Schema) GetAdditionalProperties() *Schema { return s.AdditionalProperties }
func (s Schema) GetItems() *Schema                { return s.Items }
func (s Schema) GetAllOf() []Schema               { return s.AllOf }

func (s Schema) IsZero() bool {
	s2 := Schema{}
	return reflect.DeepEqual(&s2, s)
}
