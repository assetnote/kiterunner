package proute

import (
	"fmt"
	"strings"
)

func crumbString(in []Crumb) string {
	tmp := make([]string, 0)
	for _, v := range in {
		tmp = append(tmp, fmt.Sprintf("{%s:%s}", v.Key(), v.Value()))
	}
	return strings.Join(tmp, " ")
}
