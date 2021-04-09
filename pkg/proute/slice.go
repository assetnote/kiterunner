package proute

import (
	"bufio"
	"io"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/segmentio/ksuid"
)

type PRouteOptions struct {
	HeaderCrumbs []Crumb
	QueryCrumbs  []Crumb
	PathCrumbs   []Crumb
	BodyCrumbs   []Crumb
	Method       string
	ID string
	ContentType  []ContentType
}
type PRouteOption func(o *PRouteOptions)

func OptHeader(c Crumb) PRouteOption {
	return func(o *PRouteOptions) {
		o.HeaderCrumbs = append(o.HeaderCrumbs, c)
	}
}
func OptQuery(c Crumb) PRouteOption {
	return func(o *PRouteOptions) {
		o.QueryCrumbs = append(o.QueryCrumbs, c)
	}
}
func OptPath(c Crumb) PRouteOption {
	return func(o *PRouteOptions) {
		o.PathCrumbs = append(o.PathCrumbs, c)
	}
}
func OptBody(c Crumb) PRouteOption {
	return func(o *PRouteOptions) {
		o.BodyCrumbs = append(o.BodyCrumbs, c)
	}
}
func OptID(ID string) PRouteOption {
	return func(o *PRouteOptions) {
		o.ID = ID
	}
}
func OptMethod(method string) PRouteOption {
	return func(o *PRouteOptions) {
		o.Method = method
	}
}
func OptContentType(v string) PRouteOption {
	return func(o *PRouteOptions) {
		o.ContentType = append(o.ContentType, ContentType(v))
	}
}

func FromStringSliceReader(r io.Reader, source string, opts...PRouteOption) (API, error) {
	scanner := bufio.NewScanner(r)

	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, strings.Trim(scanner.Text(), " "))
	}
	return FromStringSlice(lines, source, opts...)
}

// FromStringSlice will convert a string slice of paths into a Proute API. You can provide options to configure
// all the proutes at the same time, but not any individually
func FromStringSlice(in []string, source string, opts ...PRouteOption) (API, error) {
	ret := API{
		URL: source,
	}

	o := &PRouteOptions{
		Method: "GET",
		ID: ksuid.New().String(),
	}

	for _, v := range opts {
		v(o)
	}
	ret.ID = o.ID

	for _, v := range in {
		if len(v) == 0 {
			continue
		}
		if v[0] != '/' {
			v = "/" + v
		}

		route := Route{
			TemplatePath: v,
			Method:       o.Method,
			ContentType:  o.ContentType,
			HeaderCrumbs: o.HeaderCrumbs,
			PathCrumbs:   o.PathCrumbs,
			BodyCrumbs:   o.BodyCrumbs,
			QueryCrumbs:  o.QueryCrumbs,
		}
		ret.Routes = append(ret.Routes, route)
	}
	return ret, nil
}

func (a APIS) EncodeStringSlice(w io.Writer) (error) {
	var merr *multierror.Error
	for _, api := range a {
		for _, route := range api.Routes {
			path, err:= route.Path(true)
			if err != nil {
				multierror.Append(merr, err)
				continue
			}
			w.Write([]byte(path + "\n"))
		}
	}
	return merr.ErrorOrNil()
}