package proute

import (
	"strings"

	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/hashicorp/go-multierror"
)

func APIsToKiterunnerRoutes(api []API) ([]*http.Route, error) {
	var merr *multierror.Error
	ret := make([]*http.Route, 0)
	for _, v := range api {
		tmp, err := ToKiterunnerRoutes(v)
		if err != nil {
			multierror.Append(merr, err)
		}
		ret = append(ret, tmp...)
	}
	return ret, merr.ErrorOrNil()
}

func ToKiterunnerRoutes(api API) ([]*http.Route, error) {
	var merr *multierror.Error

	ret := make([]*http.Route, 0)
	for _, v := range api.Routes {
		// Skip these options since we don't actually care about the content here
		r, err := v.ToKiterunner(api.Headers(true)...)
		if err != nil {
			multierror.Append(merr, err)
		}
		switch string(r.Method) {
		case "HEAD", "OPTIONS", "CONNECT", "TRACE":
			// we're biased. skip these since they're noisy
			continue
		case "GET", "POST", "PUT", "DELETE", "PATCH":
			// these guys are alright
		}

		r.Source = api.ID
		ret = append(ret, r)
	}
	return ret, merr.ErrorOrNil()
}

func (r Route) ToKiterunner(extraHeaders ...KV) (*http.Route, error) {
	var err error

	method := strings.TrimSpace(strings.ToUpper(r.Method))
	switch method {
	case "HEAD", "OPTIONS", "CONNECT", "TRACE":
		// we're biased. skip these since they're noisy
	case "GET", "POST", "PUT", "DELETE", "PATCH":
		// these guys are alright
	default:
		method = "GET"
	}

	ret := &http.Route{
		Path:   []byte(r.path),
		Method: http.Method(method),
	}
	for _, h := range r.Headers(true) {
		ret.Headers = append(ret.Headers, http.Header{h.Key, h.Value})
	}

	for _, h := range extraHeaders {
		ret.Headers = append(ret.Headers, http.Header{h.Key, h.Value})
	}

	ct := ContentTypeFormEncoded
	if len(r.ContentType) > 0 {
		ct = r.ContentType[0]
		// Overwrite the form-data format with a proper boundary header
		if strings.Contains(string(ct), "form-data") {
			ct = "multipart/form-data; boundary=" + DefaultFormDataBoundary
		}
	}
	ret.Body = r.Body(true, ct)

	// only add content type if there is a body specified
	// or if its a non-get type
	if len(ret.Body) > 0 || (string(ret.Method) != string(http.GET)) {
		ret.Headers = append(ret.Headers, http.Header{"Content-Type", string(ct)})
	}

	tmp, err := r.Path(false)
	if err != nil {
		return ret, err
	}
	ret.Path = []byte( tmp )

	tmp, err = r.Query(false)
	ret.Query = []byte(tmp)
	return ret, err
}
