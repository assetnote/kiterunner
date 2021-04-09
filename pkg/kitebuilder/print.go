package kitebuilder

import (
	"fmt"
	"strings"

	"github.com/assetnote/kiterunner/pkg/log"
)

func PrintAPIs(apis []API) {
	headerTypes := make(map[string]int)
	routes := 0
	routeDistribution := make(map[int]int)
	parameterTypes := make(map[string]int)
	for _, api := range apis {
		url := api.URL
		if url == "" {
			url = "<no-url>"
		}

		securityOpts := make([]string, 0)
		for _, v := range api.SecurityDefinitions {
			securityOpts = append(securityOpts, fmt.Sprintf("{%s:%s(%s)}", v.In, v.Name, v.Type))
		}
		log.Info().Msgf("%s [%s] %s", url, api.ID, strings.Join(securityOpts, " "))

		apiRoutes := 0

		for path, ops := range api.Paths {
			for method, op := range ops {
				params := make([]string, 0)
				for _, p := range op.Parameters {
					params = append(params, fmt.Sprintf("{%s:%s(%s)}", p.In, p.Name, p.Type))
					parameterTypes[p.Type] += 1
					if p.Schema != nil && !p.Schema.IsZero() {
						if p.Schema.Type == "" {
							// v, _:= json.Marshal(p.Schema)
							for _, v := range p.Schema.Properties {
								if v.Type == "string" {
									headerTypes[strings.ToLower(v.Format)] += 1
								}
							}
							for _, v := range p.Schema.AllOf {
								if v.Type == "string" {
									headerTypes[strings.ToLower(v.Format)] += 1
								}
							}
						}
					}
				}
				_, _ = path, method
				log.Info().Msgf("\t%s %s %s", method, path, strings.Join(params, " "))
				routes += 1
				apiRoutes += 1
			}
		}
		routeDistribution[apiRoutes] += 1
	}
	log.Info().Interface("v", headerTypes).Msg("schema types")
	log.Info().Interface("v", parameterTypes).Msg("parameter types")
	log.Info().Interface("v", routeDistribution).Msg("route api distribution")
	log.Info().
		Int("apis", len(apis)).
		Int("routes", routes).
		Msg("analysis complete")
}
