package proute

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestObjectCrumb_Value(t *testing.T) {
	type fields struct {
		Name     string
		Elements []Crumb
	}
	type args struct {
		opts []CrumbOption
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{"json empty", fields{"", []Crumb{}}, args{[]CrumbOption{CrumbOptContentType(ContentTypeJSON)}}, "{}"},
		{"json simple int", fields{"", []Crumb{IntCrumb{Name: "key", Val: 123, Fixed: true}}}, args{[]CrumbOption{CrumbOptContentType(ContentTypeJSON)}}, `{"key":123}`},
		{"json nested array", fields{"", []Crumb{IntCrumb{Name: "key", Val: 123, Fixed: true}, ArrayCrumb{Name: "array", Element: IntCrumb{Val: 123, Fixed: true}}}}, args{[]CrumbOption{CrumbOptContentType(ContentTypeJSON)}}, `{"key":123,"array":[123]}`},
		{"json dupe keys", fields{"", []Crumb{IntCrumb{Name: "key", Val: 123, Fixed: true}, ArrayCrumb{Name: "key", Element: IntCrumb{Val: 123, Fixed: true}}}}, args{[]CrumbOption{CrumbOptContentType(ContentTypeJSON)}}, `{"key":123,"key":[123]}`},
		{"xml simple int", fields{"root", []Crumb{IntCrumb{Name: "key", Val: 123, Fixed: true}}}, args{[]CrumbOption{CrumbOptContentType(ContentTypeXML)}}, `<?xml version="1.0" encoding="UTF-8"?><root><key>123</key></root>`},
		{"xml nestedArray", fields{"root", []Crumb{IntCrumb{Name: "key", Val: 123, Fixed: true}, ArrayCrumb{Name: "array", Element: IntCrumb{Name: "", Val: 123, Fixed: true}}}}, args{[]CrumbOption{CrumbOptContentType(ContentTypeXML)}}, `<?xml version="1.0" encoding="UTF-8"?><root><key>123</key><array><array>123</array></array></root>`},
		{"xml nestedArrayObject", fields{"root", []Crumb{IntCrumb{Name: "key", Val: 123, Fixed: true}, ArrayCrumb{Name: "array", Element: ObjectCrumb{Name: "innerobj", Elements:[]Crumb{StaticCrumb{K: "innerkey", V:"innerv"}}}}}}, args{[]CrumbOption{CrumbOptContentType(ContentTypeXML)}}, `<?xml version="1.0" encoding="UTF-8"?><root><key>123</key><array><innerobj><innerkey>innerv</innerkey></innerobj></array></root>`},
		{"formdata simple int", fields{"", []Crumb{IntCrumb{Name: "key", Val: 123, Fixed: true}}}, args{[]CrumbOption{CrumbOptContentType(ContentTypeFormData)}}, "--hahahahahformboundaryhahahaha\r\nContent-Disposition: form-data; name=\"key\"\r\n\r\n123\r\n--hahahahahformboundaryhahahaha--\r\n"},
		{"formdata nested array", fields{"", []Crumb{IntCrumb{Name: "key", Val: 123, Fixed: true}, ArrayCrumb{Name: "array", Element: IntCrumb{Val: 123, Fixed: true}}}}, args{[]CrumbOption{CrumbOptContentType(ContentTypeFormData)}}, "--hahahahahformboundaryhahahaha\r\nContent-Disposition: form-data; name=\"key\"\r\n\r\n123\r\n--hahahahahformboundaryhahahaha\r\nContent-Disposition: form-data; name=\"array\"\r\n\r\narray=123\r\n--hahahahahformboundaryhahahaha--\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := ObjectCrumb{
				Name:     tt.fields.Name,
				Elements: tt.fields.Elements,
			}
			got := o.Value(tt.args.opts...)
			assert.Equal(t, tt.want, got)
		})
	}
}
