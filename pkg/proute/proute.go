// proute or Paramaterised-Routes allow for customisable paths with arbitrary parameters that can be generated
// to produce a full request. This is our intermediate representation between kitebuilder and kiterunner.
// This also can further be extended for programmatic use allowing for fuzzable parameters
package proute

import (
	"fmt"

	"github.com/segmentio/ksuid"
)

// API encapsulates all the routes and required headers for the routes
type API struct {
	URL    string  // the URL source of the API
	ID     string  // the Ksuid for this API which can be any UUID really... but KSUID is convenient
	Routes []Route // a list of routes

	// Crumbs to add to all the Routes
	QueryCrumbs []Crumb
	queryParams []KV

	HeaderCrumbs []Crumb
	headers      []KV

	BodyCrumbs []Crumb
	bodyParams []KV

	CookieCrumbs []Crumb
	cookieParams []KV
}

// APIS are multiple APIs, defined as a type for convenience
type APIS []API

func FromAPISlice(a []API) APIS {
	return append(APIS{}, a...)
}

func (a APIS) First(n int) APIS {
	if n == 0 {
		return a
	}

	ret := make(APIS, 0)
	count := 0
	for _, v := range a {
		if count+len(v.Routes) <= n {
			count += len(v.Routes)
			ret = append(ret, v)
		} else {
			// count + len(v.Routes) > n
			// we're over the limit so truncate
			remainder := n - count
			v.Routes = v.Routes[:remainder]

			count += remainder
			ret = append(ret, v)
			return ret
		}
	}
	return ret
}

func (a API) DebugPrint() {
	url := a.URL
	if url == "" {
		url = "<no-url>"
	}

	fmt.Printf("%s {%s}\n", url, a.ID)
	if len(a.HeaderCrumbs) > 0 {
		fmt.Printf("\theaders: %s\n", crumbString(a.HeaderCrumbs))
	}
	if len(a.QueryCrumbs) > 0 {
		fmt.Printf("\tquery: %s\n", crumbString(a.QueryCrumbs))
	}
	if len(a.CookieCrumbs) > 0 {
		fmt.Printf("\tcookie: %s\n", crumbString(a.CookieCrumbs))
	}
	if len(a.BodyCrumbs) > 0 {
		fmt.Printf("\tbody: %s\n", crumbString(a.BodyCrumbs))
	}

	for _, route := range a.Routes {
		p, _ := route.Path(false)
		fmt.Printf("\t%s %s Query(%s) Header(%s)\n", route.Method, p, crumbString(route.QueryCrumbs), crumbString(route.HeaderCrumbs))
	}
}

func (a API) QueryParams(generate bool) []KV {
	if generate || len(a.queryParams) == 0 {
		a.queryParams = a.queryParams[:0]
		for _, v := range a.QueryCrumbs {
			a.queryParams = append(a.queryParams, KV{Key: v.Key(), Value: v.Value()})
		}
	}
	return a.queryParams
}

func (a API) CookieParams(generate bool) []KV {
	if generate || len(a.cookieParams) == 0 {
		a.cookieParams = a.cookieParams[:0]
		for _, v := range a.CookieCrumbs {
			a.cookieParams = append(a.cookieParams, KV{Key: v.Key(), Value: v.Value()})
		}
	}
	return a.cookieParams
}

func (a API) BodyParams(generate bool) []KV {
	if generate || len(a.bodyParams) == 0 {
		a.bodyParams = a.bodyParams[:0]
		for _, v := range a.BodyCrumbs {
			a.bodyParams = append(a.bodyParams, KV{Key: v.Key(), Value: v.Value()})
		}
	}
	return a.bodyParams
}

func (a API) Headers(generate bool) []KV {
	if generate || len(a.headers) == 0 {
		a.headers = a.headers[:0]
		for _, v := range a.HeaderCrumbs {
			a.headers = append(a.headers, KV{Key: v.Key(), Value: v.Value()})
		}
	}
	return a.headers
}

func NewAPI(id string, url string) API {
	if id == "" {
		id = ksuid.New().String()
	}
	return API{
		ID:  id,
		URL: url,
	}
}
