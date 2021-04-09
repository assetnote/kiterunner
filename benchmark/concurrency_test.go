package benchmark

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func doNetwork() {
	time.Sleep(0 * time.Millisecond)
}

type args struct {
	parallelHosts int
	connPerHost   int
	targets       int
	chunks        int
	requests      int
}

type test struct {
	name     string
	input    args
	expected int
}

type reqmsg struct {
	typ    string
	resp   chan interface{}
	routes []int
	target *target
}

type target struct {
	semaphore chan interface{}
}

func NewTarget(size int) *target {
	ret := &target{
		semaphore: make(chan interface{}, size),
	}
	for i := 0; i < size; i++ {
		ret.semaphore <- i
	}
	return ret
}

func (t *target) Acquire() {
	<-t.semaphore
}
func (t *target) Release() {
	t.semaphore <- struct{}{}
}

func BenchmarkModel1(b *testing.B) {
	tests := []test{
		{"simple", args{10, 5, 10, 10, 10}, 1000},
		{"larger", args{100, 5, 100, 10, 10}, 10000},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			var res int
			for i := 0; i < b.N; i++ {
				res = RunModel1(test.input.parallelHosts, test.input.connPerHost, test.input.targets, test.input.chunks, test.input.requests)
			}
			_ = res
		})
	}
}

func BenchmarkModel2(b *testing.B) {
	tests := []test{
		{"simple", args{10, 5, 10, 10, 10}, 1000},
		{"larger", args{100, 5, 100, 10, 10}, 10000},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			var res int
			for i := 0; i < b.N; i++ {
				res = RunModel2(test.input.parallelHosts, test.input.connPerHost, test.input.targets, test.input.chunks, test.input.requests)
			}
			_ = res
		})
	}
}

func BenchmarkModel3(b *testing.B) {
	tests := []test{
		{"simple", args{10, 5, 10, 10, 10}, 1000},
		{"larger", args{100, 5, 100, 10, 10}, 10000},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			var res int
			for i := 0; i < b.N; i++ {
				res = RunModel3(test.input.parallelHosts, test.input.connPerHost, test.input.targets, test.input.chunks, test.input.requests)
			}
			_ = res
		})
	}
}

func BenchmarkModel4(b *testing.B) {
	tests := []test{
		{"simple", args{10, 5, 10, 10, 10}, 1000},
		{"larger", args{100, 5, 100, 10, 10}, 10000},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			var res int
			for i := 0; i < b.N; i++ {
				res = RunModel4(test.input.parallelHosts, test.input.connPerHost, test.input.targets, test.input.chunks, test.input.requests)
			}
			_ = res
		})
	}
}

func TestModel1(b *testing.T) {
	tests := []test{
		{"simple", args{10, 5, 10, 10, 10}, 1000},
		{"larger", args{100, 5, 100, 10, 10}, 10000},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.T) {
			res := RunModel1(test.input.parallelHosts, test.input.connPerHost, test.input.targets, test.input.chunks, test.input.requests)
			assert.Equal(b, res, test.expected)
		})
	}
}

func TestModel2(b *testing.T) {
	tests := []test{
		{"simple", args{10, 5, 10, 10, 10}, 1000},
		{"larger", args{100, 5, 100, 10, 10}, 10000},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.T) {
			res := RunModel2(test.input.parallelHosts, test.input.connPerHost, test.input.targets, test.input.chunks, test.input.requests)
			assert.Equal(b, res, test.expected)
		})
	}
}

func TestModel3(b *testing.T) {
	tests := []test{
		{"simple", args{10, 5, 10, 10, 10}, 1000},
		{"larger", args{100, 5, 100, 10, 10}, 10000},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.T) {
			res := RunModel3(test.input.parallelHosts, test.input.connPerHost, test.input.targets, test.input.chunks, test.input.requests)
			assert.Equal(b, res, test.expected)
		})
	}
}

func TestModel4(b *testing.T) {
	tests := []test{
		{"simple", args{10, 5, 10, 10, 10}, 1000},
		{"larger", args{100, 5, 100, 10, 10}, 10000},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.T) {
			res := RunModel4(test.input.parallelHosts, test.input.connPerHost, test.input.targets, test.input.chunks, test.input.requests)
			assert.Equal(b, res, test.expected)
		})
	}
}

// Model message-passing
//               ┌────────────┐       ┌───────────────────────────────┐
//               │            │       │                               │
//               ▼            │       ▼        (A)                 (B)│
//     ┌─► target_wkr ───► preflight_wkr ─┐  1.┌─► preflight_check ─┐ │
//     │                                  │    │                    │ │
// tx ─┼─► target_wkr ───► preflight_wkr ─┼────┼─► preflight_check ─┼─┘
//     │                                  │    │                    │
//     └─► target_wkr ───► preflight_wkr ─┘    └─► preflight_check ─┘
//           │  │  │ ▲             ▲                 │
//           └──┼──┘ │             │                 │
//              │    └───target request semaphore ◄──┘
//              │               ▲
//              │      ┌─► request_wkr ─┐
//              │      │                │
//       2. (C) └──────┼─► request_wkr ─┼─► rx
//                     │                │
//                     └─► request_wkr ─┘
func RunModel1(parallelHosts int, connPerHost int, targets, chunks, requests int) int {
	var (
		preflightChecks    = 10
		tx                 = make(chan *target, parallelHosts)
		preflightCheckChan = make(chan reqmsg, parallelHosts*5)
		requestChan        = make(chan reqmsg, parallelHosts*connPerHost)
		rx                 = make(chan interface{}, parallelHosts*connPerHost)

		targetWg    sync.WaitGroup
		preflightWg sync.WaitGroup
		requestWg   sync.WaitGroup
	)
	for i := 0; i < parallelHosts; i++ {
		targetWg.Add(1)
		go func() {
			defer targetWg.Done()
			for tg := range tx {
				chunkChan := make(chan interface{}, 3)
				go func() {
					for k := 0; k < chunks; k++ {
						respChan := make(chan interface{}, parallelHosts*preflightChecks)
						for j := 0; j < preflightChecks; j++ {
							tg.Acquire()
							preflightCheckChan <- reqmsg{typ: "preflight", resp: respChan, target: tg}
						}

						for j := 0; j < preflightChecks; j++ {
							_ = <-respChan
						}
						chunkChan <- struct{}{}
					}
					close(chunkChan)
				}()

				for _ = range chunkChan {
					for j := 0; j < requests; j++ {
						tg.Acquire()
						requestChan <- reqmsg{typ: "request", target: tg}
					}
				}
			}
		}()
	}
	for i := 0; i < parallelHosts; i++ {
		preflightWg.Add(1)
		go func() {
			defer preflightWg.Done()
			for msg := range preflightCheckChan {
				doNetwork()
				msg.target.Release()
				msg.resp <- struct{}{}
			}
		}()
	}

	for i := 0; i < parallelHosts*connPerHost; i++ {
		requestWg.Add(1)
		go func() {
			defer requestWg.Done()
			for msg := range requestChan {
				doNetwork()
				msg.target.Release()
				rx <- msg
			}
		}()
	}

	go func() {
		for i := 0; i < targets; i++ {
			tx <- NewTarget(connPerHost)
		}
		close(tx)
		targetWg.Wait()
		close(preflightCheckChan)
		preflightWg.Wait()
		close(requestChan)
		requestWg.Wait()
		close(rx)
	}()

	total := 0
	for tg := range rx {
		total += 1
		_ = tg
	}
	return total
}

// similar to model1, where we spin up a goroutine on demand for each preflight request
func RunModel2(parallelHosts int, connPerHost int, targets, chunks, requests int) int {
	var (
		preflightChecks = 10
		tx              = make(chan *target, parallelHosts)
		requestChan     = make(chan reqmsg, parallelHosts*connPerHost)
		rx              = make(chan interface{}, parallelHosts*connPerHost)

		targetWg  sync.WaitGroup
		requestWg sync.WaitGroup
	)
	for i := 0; i < parallelHosts; i++ {
		targetWg.Add(1)
		go func() {
			defer targetWg.Done()
			for tg := range tx {
				respChan := make(chan interface{}, parallelHosts*preflightChecks)
				for k := 0; k < chunks; k++ {
					for j := 0; j < preflightChecks; j++ {
						go func() {
							tg.Acquire()
							doNetwork()
							tg.Release()
							respChan <- struct{}{}
						}()
					}

					for j := 0; j < preflightChecks; j++ {
						_ = <-respChan
					}

					for j := 0; j < requests; j++ {
						tg.Acquire()
						requestChan <- reqmsg{typ: "request", target: tg}
					}
				}
			}
		}()
	}

	for i := 0; i < parallelHosts*connPerHost; i++ {
		requestWg.Add(1)
		go func() {
			defer requestWg.Done()
			for msg := range requestChan {
				doNetwork()
				msg.target.Release()
				rx <- msg
			}
		}()
	}

	go func() {
		for i := 0; i < targets; i++ {
			tx <- NewTarget(connPerHost)
		}
		close(tx)
		targetWg.Wait()
		close(requestChan)
		requestWg.Wait()
		close(rx)
	}()

	total := 0
	for tg := range rx {
		total += 1
		_ = tg
	}
	return total
}

// each target gets a dedicated set of threads
//                     ┌─► request_wkr ──┬───┐
//     ┌─► target_wkr ─┼─► request_wkr ──┤   │
//     │         ▲     └─► request_wkr ──┴─┐ │
//     │         └─────────────────────────┘ │
//     │               ┌─► request_wkr ──┬───┼─────► rx
// tx ─┼─► target_wkr ─┼─► request_wkr ──┤   │
//     │         ▲     └─► request_wkr ──┴─┐ │
//     │         └─────────────────────────┘ │
//     │               ┌─► request_wkr ──┬───┘
//     └─► target_wkr ─┼─► request_wkr ──┤
//               ▲     └─► request_wkr ──┴─┐
//               └─────────────────────────┘
func RunModel3(parallelHosts int, connPerHost int, targets, chunks, requests int) int {
	var (
		preflightChecks = 10
		tx              = make(chan interface{}, parallelHosts)
		rx              = make(chan interface{}, parallelHosts*connPerHost)

		allReqChans = make([]chan reqmsg, 0)

		targetWg  sync.WaitGroup
		requestWg sync.WaitGroup
	)
	for i := 0; i < parallelHosts; i++ {
		requestChan := make(chan reqmsg, connPerHost)
		allReqChans = append(allReqChans, requestChan)

		targetWg.Add(1)
		go func() {
			defer targetWg.Done()
			for _ = range tx {
				respChan := make(chan interface{}, preflightChecks)

				for k := 0; k < chunks; k++ {
					for j := 0; j < preflightChecks; j++ {
						requestChan <- reqmsg{typ: "preflight", resp: respChan}
					}

					for j := 0; j < preflightChecks; j++ {
						_ = <-respChan
					}

					for j := 0; j < requests; j++ {
						requestChan <- reqmsg{typ: "request", resp: nil}
					}
				}
			}
		}()

		for i := 0; i < connPerHost; i++ {
			requestWg.Add(1)
			go func() {
				defer requestWg.Done()
				for tg := range requestChan {
					doNetwork()
					switch tg.typ {
					case "preflight":
						tg.resp <- tg
					case "request":
						rx <- tg
					}
				}
			}()
		}
	}

	go func() {
		for i := 0; i < targets; i++ {
			tx <- i
		}
		close(tx)
		targetWg.Wait()
		for _, v := range allReqChans {
			close(v)
		}
		requestWg.Wait()
		close(rx)
	}()

	total := 0
	for tg := range rx {
		total += 1
		_ = tg
	}
	return total
}

func chunkBy(items []int, chunkSize int) (chunks [][]int) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}

	return append(chunks, items)
}

// each target gets a dedicated set of threads
//                     ┌─► request_wkr ──┬───┐
//     ┌─► target_wkr ─┼─► request_wkr ──┤   │
//     │         ▲     └─► request_wkr ──┴─┐ │
//     │         └─────────────────────────┘ │
//     │               ┌─► request_wkr ──┬───┼─────► rx
// tx ─┼─► target_wkr ─┼─► request_wkr ──┤   │
//     │         ▲     └─► request_wkr ──┴─┐ │
//     │         └─────────────────────────┘ │
//     │               ┌─► request_wkr ──┬───┘
//     └─► target_wkr ─┼─► request_wkr ──┤
//               ▲     └─► request_wkr ──┴─┐
//               └─────────────────────────┘
func RunModel4(parallelHosts int, connPerHost int, targets, chunks, requestCount int) int {
	var (
		preflightChecks = 10
		tx              = make(chan interface{}, parallelHosts)
		rx              = make(chan interface{}, parallelHosts*connPerHost)

		allReqChans = make([]chan reqmsg, 0)

		targetWg  sync.WaitGroup
		requestWg sync.WaitGroup
	)

	requests := make([]int, 0)
	for i := 0; i < requestCount; i++ {
		requests = append(requests, i)
	}

	for i := 0; i < parallelHosts; i++ {
		requestChan := make(chan reqmsg, connPerHost)
		allReqChans = append(allReqChans, requestChan)

		targetWg.Add(1)
		go func() {
			defer targetWg.Done()
			for _ = range tx {
				respChan := make(chan interface{}, preflightChecks)

				for k := 0; k < chunks; k++ {
					for j := 0; j < preflightChecks; j++ {
						requestChan <- reqmsg{typ: "preflight", resp: respChan}
					}

					for j := 0; j < preflightChecks; j++ {
						_ = <-respChan
					}

					for _, v := range chunkBy(requests, connPerHost) {
						requestChan <- reqmsg{typ: "request", resp: nil, routes: v}
					}
				}
			}
		}()

		for i := 0; i < connPerHost; i++ {
			requestWg.Add(1)
			go func() {
				defer requestWg.Done()
				for tg := range requestChan {
					doNetwork()
					switch tg.typ {
					case "preflight":
						tg.resp <- tg
					case "request":
						for _, v := range tg.routes {
							rx <- v
						}
					}
				}
			}()
		}
	}

	go func() {
		for i := 0; i < targets; i++ {
			tx <- i
		}
		close(tx)
		targetWg.Wait()
		for _, v := range allReqChans {
			close(v)
		}
		requestWg.Wait()
		close(rx)
	}()

	total := 0
	for tg := range rx {
		total += 1
		_ = tg
	}
	return total
}
