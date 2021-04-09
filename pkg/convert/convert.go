package convert

// IntMapToSlice will return all the keys for the given intmap
func IntMapToSlice(m map[int]interface{}) (ret []int) {
	for k := range m {
		ret = append(ret, k)
	}
	return ret
}

// IntSliceToMap will return a map with keys created from the slice
func IntSliceToMap(v []int) map[int]interface{} {
	ret := make(map[int]interface{})
	for _, vv := range v {
		ret[vv] = struct{}{}
	}
	return ret
}

// StringMapToSlice will return all the keys for the given string map
func StringMapToSlice(m map[string]interface{}) (ret []string) {
	for k := range m {
		ret = append(ret, k)
	}
	return ret
}

// UniqueStrings will remove duplicates preserving order of the input
func UniqueStrings(in []string) (out []string) {
	seen := make(map[string]interface{})
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return
}
