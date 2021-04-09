package kiterunner

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.SetLevelString("error")
}

func MakeTargets(count int) []*http.Target {
	ret := make([]*http.Target, 0)
	for i := 0; i < count; i++ {
		ret = append(ret, &http.Target{
			IP:       "localhost",
			Hostname: "localhost",
			Port:     i + 14000,
		})
	}
	return ret
}

func MakeRoutes(count int) map[string][]*http.Route {
	ret := make([]*http.Route, count)
	for i := 0; i < count; i++ {
		ret[i] = &http.Route{
			Path: []byte("/" + strconv.Itoa(i)),
		}
	}
	return map[string][]*http.Route{
		"": ret,
	}
}

func MakeRedirectRoutes(buckets int, routes int) map[string][]*http.Route {
	rr := make(map[string][]*http.Route)
	for i := 0; i < buckets; i++ {
		ret := make([]*http.Route, routes)
		for j := 0; j < routes; j++ {
			ret[j] = &http.Route{
				Path: []byte(fmt.Sprintf("/redir/%d-%d", i, j)),
			}
		}
		rr[strconv.Itoa(i)] = ret
	}
	return rr
}

// KiterunnerRunAsync demonstrates how to perform an RunAsync call and how to process results
// We recommend releasing the Result after processing is complete to avoid unnecessary allocations for future
// results. the penalty invoked is just the cost of allocating the memory for a new Result, which is relatively
// nominal compared to the number of requests sent
func Example_kiterunnerRunAsync() {
	routes := MakeRedirectRoutes(1, 1)
	e := NewEngine(routes,
		ReadBody(false),
		// adjusting this significantly affects allocations
		MaxParallelHosts(1),
		MaxConnPerHost(2),
	)
	ctx := context.Background()
	targets := MakeTargets(1)

	for _, v := range targets {
		// set the context so they can be cancelled if the task is cancelled
		v.SetContext(ctx)
		// parse the host header so we don't send garbage to the client
		v.ParseHostHeader()
	}

	// start the loop and get your communication channels
	tx, rx, err := e.RunAsync(ctx)
	if err != nil { // handle err
	}

	var res []*Result
	for _, v := range targets {
		tx <- v
	}
	close(tx)

	for r := range rx {
		res = append(res, r)
		// OR if you don't need results
		r.Release()
	}

	// rx will close when tx is closed
}

func TestKiterunnerEngineRunGoland1Async(t *testing.T) {
	routes := MakeRedirectRoutes(1, 1)
	e := NewEngine(routes,
		ReadBody(false),
		// adjusting this significantly affects allocations
		MaxParallelHosts(1),
		MaxConnPerHost(2),
		ReadHeaders(false),
		SkipPreflight(true),

		// have no successful statuscodes to simulate all requests failing
		// kiterunner.SuccessStatusCodes([]int{999}),
	)
	ctx := context.Background()
	targets := MakeTargets(1)
	// create all the clients
	for _, v := range targets {
		v.SetContext(ctx)
		v.HTTPClient(e.Config().MaxConnPerHost, e.Config().HTTP.Timeout)
	}
	expected := len(targets) * 1
	// warmup

	tx, rx, err := e.RunAsync(ctx)
	if err != nil {
		t.Fatalf("failed benchmark: %v", err)
	}

	var res []*Result
	for _, v := range targets {
		tx <- v
	}
	close(tx)
	for r := range rx {
		res = append(res, r)
	}
	assert.Len(t, res, expected)
}

func BenchmarkKiterunnerEngineRunGoland1Async(b *testing.B) {
	b.ReportAllocs()

	targetCount := 1
	// routes := MakeRedirectRoutes(1, 1)
	routes := MakeRoutes(1)
	e := NewEngine(routes,
		ReadBody(false),
		// adjusting this significantly affects allocations
		MaxParallelHosts(targetCount),
		MaxConnPerHost(5),
		ReadHeaders(false),
		SkipPreflight(true),

		// have no successful statuscodes to simulate all requests failing
		// kiterunner.SuccessStatusCodes([]int{999}),
	)
	ctx := context.Background()
	targets := MakeTargets(targetCount)
	// create all the clients
	for _, v := range targets {
		v.SetContext(ctx)
		v.HTTPClient(e.Config().MaxConnPerHost, e.Config().HTTP.Timeout)
	}
	expected := len(targets) * 1
	// warmup

	b.ResetTimer()

	tx, rx, err := e.RunAsync(ctx)
	if err != nil {
		b.Fatalf("failed benchmark: %v", err)
	}

	var res *Result
	for i := 0; i < b.N; i++ {
		for _, v := range targets {
			tx <- v
		}
		for j := 0; j < expected; j++ {
			res = <-rx
			res.Release()
		}
	}
	_ = res
}

func BenchmarkKiterunnerEngineRunGoland1(b *testing.B) {
	b.ReportAllocs()

	e := NewEngine(MakeRedirectRoutes(1, 1),
		ReadBody(false),
		// adjusting this significantly affects allocations
		MaxParallelHosts(1),
		MaxConnPerHost(5),
		ReadHeaders(false),

		// have no successful statuscodes to simulate all requests failing
		AddRequestFilter(NewStatusCodeWhitelist([]int{999})),
	)
	ctx := context.Background()
	targets := MakeTargets(1)
	// create all the clients
	for _, v := range targets {
		v.SetContext(ctx)
		v.HTTPClient(e.Config().MaxConnPerHost, e.Config().HTTP.Timeout)
	}
	// warmup

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var count int32
		err := e.RunCallbackNoResult(ctx, targets, func(*Result, *Config) {
			atomic.AddInt32(&count, 1)
		})
		if err != nil {
			b.Fatalf("failed benchmark: %v", err)
		}
		if count > 0 {
			b.Fatal("too many results", count)
		}
	}
}

func BenchmarkKiterunnerEngineRunGolandSmall100(b *testing.B) {
	b.ReportAllocs()

	e := NewEngine(MakeRedirectRoutes(50, 10),
		ReadBody(false),
		ReadHeaders(false),
	)
	ctx := context.Background()
	targets := MakeTargets(10)
	// create all the clients
	for _, v := range targets {
		v.SetContext(ctx)
		v.HTTPClient(e.Config().MaxConnPerHost, e.Config().HTTP.Timeout)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		res, err := e.Run(ctx, targets)
		if err != nil {
			b.Fatalf("failed benchmark: %v", err)
		}
		_ = res
	}
}

func BenchmarkKiterunnerEngineRun(b *testing.B) {
	tests := []struct {
		name    string
		input   int
		buckets int
		routes  int
	}{
		{"singular-1,1,1", 1, 1, 1},
		{"singular-1,5,10", 1, 5, 10},
		{"singular-1,1,100", 1, 1, 100},
		{"singular-1,100,1", 1, 100, 1},
		{"tiny-50,5,10", 50, 5, 10},
		{"tiny-50,1,100", 50, 1, 100},
		{"tiny-50,100,1", 50, 100, 1},
		{"small-100", 100, 5, 10},
		{"large-500", 500, 5, 10},
		// {"huge-1000", 1000},
	}

	max := 500
	ctx := context.Background()
	targets := MakeTargets(max)
	e := NewEngine(MakeRedirectRoutes(5, 10),
		ReadBody(false),
		ReadHeaders(false),
	)
	for _, v := range targets {
		v.SetContext(ctx)
		v.HTTPClient(e.Config().MaxConnPerHost, e.Config().HTTP.Timeout)
	}
	for _, test := range tests {
		e := NewEngine(MakeRedirectRoutes(test.buckets, test.routes))
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			e.Config().MaxParallelHosts = test.input/2 + 1
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				res, err := e.Run(ctx, targets[0:test.input])
				if err != nil {
					b.Fatalf("failed benchmark: %v", err)
				}
				_ = res
				// don't actually do the counts here since benchmarks can abort at any point
				// if len(res) != test.input*count {
				//	b.Fatalf("failed benchmark: unexpected count: got %d want %d", len(res), test.input*count)
				// }
			}
		})
	}
}
