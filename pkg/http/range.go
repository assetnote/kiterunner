package http

import (
	"fmt"
	"strconv"
	"strings"
)

type Range struct {
	Min int
	Max int
}

func (r Range) String() string {
	if r.Min == r.Max && r.Max == 0 {
		return ""
	}
	if r.Min == r.Max {
		return strconv.Itoa(r.Min)
	}
	return fmt.Sprintf("%d-%d", r.Min, r.Max)
}

// RangeFromString will return a range from a string like 5-10
func RangeFromString(in string) (ret Range, err error) {
	if !strings.Contains(in, "-") {
		// treat it as a single value
		ret.Min, err = strconv.Atoi(in)
		if err != nil {
			return ret, fmt.Errorf("unable to parse range: %w", err)
		}
		ret.Max = ret.Min
		return ret, nil
	}
	v := strings.SplitN(in, "-", -1)
	if len(v) != 2 {
		return ret, fmt.Errorf("unexpected format for range")
	}

	ret.Min, err = strconv.Atoi(v[0])
	if err != nil {
		return ret, fmt.Errorf("unable to parse range min: %w", err)
	}

	ret.Max, err = strconv.Atoi(v[1])
	if err != nil {
		return ret, fmt.Errorf("unable to parse range max: %w", err)
	}

	if ret.Min > ret.Max {
		return ret, fmt.Errorf("invalid range. min is not lower than max")
	}
	return ret, nil
}