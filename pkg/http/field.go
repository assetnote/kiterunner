package http

import (
	"strings"
)

type HeaderField struct {
	Key Field
	Value Field
}

type FieldType int
const (
	String FieldType = iota
	UUID
	Int
	Date
	Timestamp
	Format
)

type Field struct {
	Key string `toml:"key" json:"key" mapstructure:"key"`
	Type FieldType `toml:"type" json:"type" mapstructure:"type"`
}

func (f *Field) Bytes() []byte {
	if len(f.Key) == 0 {
		return nil
	}
	return []byte(f.Key)
}

// StringToField will break down a URL path into fields
// This is a convenience function to convert /foo/bar into []Field{{"foo"}, {"bar"}}
func StringToFields(in string) (ret []Field) {
	for _, v := range strings.Split(in, "/") {
		ret = append(ret, Field{Key: v})
	}
	return ret
}

// Write the string to the specified writer
func (f *Field) AppendBytes(dst []byte) []byte {
	return append(dst, f.Key...)
}