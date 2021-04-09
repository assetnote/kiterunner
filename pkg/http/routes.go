package http

// getDepth will return up to the N'th+1 slash in the path.
// e.g. /foo/bar/baz depth 2 will return /foo/bar
// If a prefix slash is not included, we will prepend the prefix slash,
// e.g. foo/bar/baz depth 2 will return /foo/bar
// If there is insufficient path elements, we will return the input element
func getDepth(path string, depth int64) (ret string) {
	// if we wanted no depth, then the basepath is empty
	if depth == 0 {
		return ""
	}

	// not sure how this is possible but sure
	if len(path) == 0 {
		return path
	}

	if path[0] != '/' {
		path = "/" + path
	}
	var hits int64 = 0
	for i, v := range path {
		if v == '/' {
			hits++
		}

		if hits == depth+1 {
			return path[0:i]
			break
		}
	}

	return path
}

type RouteMap map[string][]*Route

func (r RouteMap) FlattenCount() int {
	ret := 0
	for _, v := range r {
		ret += len(v)
	}
	return ret
}

func (r RouteMap) Flatten() []*Route {
	ret := make([]*Route, 0)
	for _, v := range r {
		ret = append(ret, v...)
	}
	return ret
}

// GroupRouteDepth will collate the routes into the corresponding depth of path
func GroupRouteDepth(routes []*Route, depth int64) RouteMap {
	ret := make(map[string][]*Route)
	for _, v := range routes {
		basePath := getDepth(string(v.Path), depth)
		ret[basePath] = append(ret[basePath], v)
	}
	return ret
}

// UniqueSource will unique the routes based on the ID
func UniqueSource(routes []*Route) []*Route {
	ret := make([]*Route, 0)
	seen := make(map[string]interface{})
	for _, v := range routes {
		if _, ok := seen[v.Source]; ok {
			continue
		}
		seen[v.Source] = struct{}{}
		ret = append(ret, v)
	}
	return ret
}

// FilterSource will only return the routes that exist in the map provided
func FilterSource(routes []*Route, want map[string]interface{}) []*Route {
	ret := make([]*Route, 0)
	for _, v := range routes {
		if _, ok := want[v.Source]; !ok {
			continue
		}
		ret = append(ret, v)
	}
	return ret
}
