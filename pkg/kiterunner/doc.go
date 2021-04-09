/*
Package kiterunner provides the core request loop for performing highly concurrent low allocation requests.

This package should be used where you want to make a large number of requests across a large number of targets quickly,
concurrently and with 0 allocations.

The 0 allocation benchmark comes from BenchmarkKiterunnerEngineRunGoland1Async located in benchmark_integration_test.go
the benchmark assumes the following
 - all your targets have been allocated
 - all your routes have been allocated
 - the caller releases Results back into the pool after they have completed reading the result
Hence the 0 allocations corresponds to 0 additional allocations performed while performing requests. On a 2020 Macbook
13" this yields the following results.

	❯ ulimit -n 20000 && go test '-bench=BenchmarkKiterunnerEngineRunGoland1Async$' -v ./...   -run='^$'  -count=10 -benchtime=10000x -benchmem
	goos: darwin
	goarch: amd64
	pkg: github.com/assetnote/kiterunner/pkg/kiterunner
	BenchmarkKiterunnerEngineRunGoland1Async
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             75670 ns/op               5 B/op          0 allocs/op
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             79782 ns/op               5 B/op          0 allocs/op
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             88210 ns/op               6 B/op          0 allocs/op
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             88506 ns/op               6 B/op          0 allocs/op
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             93852 ns/op               6 B/op          0 allocs/op
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             78248 ns/op               6 B/op          0 allocs/op
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             74890 ns/op               6 B/op          0 allocs/op
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             74421 ns/op               7 B/op          0 allocs/op
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             74885 ns/op               6 B/op          0 allocs/op
	BenchmarkKiterunnerEngineRunGoland1Async-8         10000             74722 ns/op               6 B/op          0 allocs/op
	PASS
	ok      github.com/assetnote/kiterunner/pkg/kiterunner  8.170s

The allocation savings primarily derive from aggressively using sync.Pools for objects that are re-used across targets
and mis-using channels as low cost, cross goroutine communication buffers. These optimisations have resulted in slightly
difficult to understand code in the handleTarget function. Future attempts to develop on this codebase should ensure
that the baseline benchmarks aren't exceeded per commit, or where they are, a justifiable reason for the performance
degradation is provided

The concurrency model elected for the Async loop is defined in RunAsync. This uses 3 goroutine worker separations
 - One goroutine per target to supervise the scheduling of requests
 - One goroutine for scheduling preflight requests. Spawned by the target thread
 - N goroutines (Max Conn Per Host) per target for performing requests.
    - request_wkr threads process both preflight requests and normal requests
		- preflight requests sent to request_wkr include the response channel where results are sent
		- normal requests are sent to the rx channel created a RunAsync time

This is illustrated in the following flow diagram

             (A)               (B)
             ┌─► preflight_wkr ──┬─► request_wkr ──┐
       ┌─► target_wkr─────(D)────┼─► request_wkr ──┼───┐
       │         ▲               └─► request_wkr ──┤  (E)
       │         └─────────────────────────────(C)─┘   │
       │     ┌─► preflight_wkr ──┬─► request_wkr ──┐   │
   tx ─┼─ target_wkr─────────────┼─► request_wkr ──┼───┼──►rx
       │        ▲                └─► request_wkr ──┤   │
       │        └──────────────────────────────────┘   │
       │     ┌─► preflight_wkr ──┬─► request_wkr ──┐   │
       └─► target_wkr────────────┼─► request_wkr ──┼───┘
                 ▲               └─► request_wkr ──┤
                 └─────────────────────────────────┘
 (A) - target_wkr    - spawns preflight worker
 (B) - preflight_wkr - schedules preflight requests for given baseline
 (C) - request_wkr   - performs request and sends response to target_wkr for aggregation
 (D) - target_wkr    - schedules a batch of requests for given baseline
 (E) - request_wkr   - results are sent to rx channel for aggregation

This concurrency model was selected based on the benchmarks run in github.com/assetnote/kiterunner/benchmark/concurrency_test.go
where this was determined to provide a compromise between:
 - Allowing for preflight requests to occur concurrently with normal requests
     - The target_wkr does not need to wait for a directory to complete before performing the next preflight requests
 - Restricting the number of requests sent to a given host at any one time
     - We only use the pool of request_wkr for each given target to make requests
 - Minimising inter-thread waits on channels
     - We batch up the requests on a directory so the request_wkr doesnt need to read from the channel for each subsequent
       request in a directory they already have data for
 - Avoiding a single slow target from blocking other targets from processing

Kiterunner Wrappers

kiterunner_wrappers.go contains a set of synchronous wrapper functions for RunAsync. These are provided as utilities
for callers who don't wish to have the complexity of managing a channel based function call.
Using the functions that return Results will avoid releasing the Result back into the
pool, consuming requests, but allowing you to aggregate the results.
*/
package kiterunner
