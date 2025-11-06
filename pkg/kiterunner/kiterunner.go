package kiterunner

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/assetnote/kiterunner/pkg/log"
)

// CheckInterval indicates how many requests a worker thread will process before checking for context cancellation
// or target quarantine
const CheckInterval = 10

var (
	// ErrTargetQuarantined indicates that the target has crossed the threshold amount for consecutive non-baseline requests
	// and has consequently been quarantined. This indicates the host should not be scanned anymore
	ErrTargetQuarantined = fmt.Errorf("target quarantined")
	// DefaultRootRoute is the default base route that is used as a canary baseline against all hosts.
	DefaultRootRoute = []*http.Route{{Path: []byte("/"), Method: http.GET}}
)

// Engine provides a scan configuration with a set of routes and a specified configuration
// Calling Run or RunAsync can be done concurrently. Each call will create its own threadpool
// If you wish to access one threadpool from multiple workers, use RunAsync and communicate using
// the provided channels
// The options are non-configurable after instantiation as modifying the routes or config
// during Run or RunAsync may lead to non-deterministic behaviour
type Engine struct {
	config *Config
	routes http.RouteMap
}

// NewEngine will create an engine with the defined routes and options.
// If you require a different set of routes, you should instantiate a new engine
func NewEngine(routes http.RouteMap, opts ...ConfigOption) *Engine {
	e := &Engine{
		config: NewDefaultConfig(),
		routes: routes,
	}
	for _, o := range opts {
		o(e.config)
	}
	return e
}

// Config returns the config for the engine. Modifying this config will modify the config
// for any currently running scans
func (e *Engine) Config() *Config {
	return e.config
}

// handleTarget will process the target provided. This will manage the number of concurrent requests
// that can be sent to a particular host with a semaphore channel
// this function processes preflight check asynchronously
// This function will terminate when the context is cancelled.
// the child goroutine will receive the context cancellation and close the subpathRouteChannel
// the function will wait for the subpathRouteChannel to be closed before exiting to ensure no goroutines are left before exiting
func handleTarget(ctx context.Context, target *http.Target, reqChan chan *ReqMsg, rm http.RouteMap, config *Config) error {
	// we first parse the host header to ensure its populated in case its nil
	target.ParseHostHeader()
	c := target.HTTPClient(config.MaxConnPerHost, config.HTTP.Timeout)
	preflightChecks := len(config.PreflightCheckRoutes)

	// we expect N routes for the number of requests + (M + 1) * baselines + 1 root request
	totalExpectedRequests := rm.FlattenCount() + (len(config.PreflightCheckRoutes) * (len(rm) + 1))
	requestsSent := 0
	defer func() {
		log.Debug().
			Int("expected", totalExpectedRequests).
			Int("sent", requestsSent).
			Int("diff", totalExpectedRequests-requestsSent).
			Bytes("target", target.Bytes()).
			Msg("requests count")
		// decrement the progress bar by the amount of requests we missed by
		config.ProgressBar.AddTotal(int64(requestsSent - totalExpectedRequests))
	}()

	// some random number for a buffer
	// this channel is only closed when the target is cancelled. In a normal request loop where the target is online
	// then this will just be drained and returned to the pool.
	// Responsibility is on the caller to ensure that the channel is empty before returning it to the pool
	subpathRouteChan := acquireSubpathRoutesChan(config.MaxConnPerHost)

	// waitgroup used to ensure we exit this function only after our child scheduler goroutine exits. Otherwise
	// we get panics when channels are closed unexpected
	wg := acquireWaitGroup()
	defer func() {
		wg.Wait()
		releaseWaitGroup(wg)
	}()
	// defer log.Debug().Str("target", target.String()).Msg("target func exiting")

	// run preflight checks async to scheduling the targets. this way we don't get blocked each time
	// we need to traverse to the next depth
	// this goroutine will exit either when the context is cancelled or when it has exhausted all the scheduling
	wg.Add(1)
	go func() {
		defer wg.Done()
		// first do the docroot to ensure that we establish the initial baseline
		{
			base := []byte("/")
			// fetch a channel for us to send responses to
			subpathRespChan := acquireSubpathBaselineChan(preflightChecks)
			for _, req := range config.PreflightCheckRoutes {
				t := acquireReqMsg()
				t.typ = PreflightMsg
				t.Preflight = acquireSubpathBaseline()
				t.Preflight.base = append(t.Preflight.base, base...)
				t.Preflight.target = target
				t.Preflight.route = req
				// responses will come back to here
				t.Preflight.resp = subpathRespChan

				// dispatch our requests
				reqChan <- t
			}

			// send the collector to the main thread to aggregate
			select {
			case <-target.Context().Done():
				// log.Debug().Str("target", target.String()).Msg("target scheduler context cancellation received")
				// context is cancelled drop everything and fuck off
				close(subpathRouteChan)
				return
			case subpathRouteChan <- subpathRoutes{responses: subpathRespChan, routes: nil}:
			}
		}

		for base, routes := range rm {
			subpathRespChan := acquireSubpathBaselineChan(preflightChecks)
			for _, req := range config.PreflightCheckRoutes {
				t := acquireReqMsg()
				t.typ = PreflightMsg
				t.Preflight = acquireSubpathBaseline()
				t.Preflight.base = append(t.Preflight.base, base...)
				t.Preflight.target = target
				t.Preflight.route = req
				t.Preflight.resp = subpathRespChan

				reqChan <- t
			}

			// send the collector to the main thread to aggregate
			select {
			case <-target.Context().Done():
				// log.Debug().Str("target", target.String()).Msg("target scheduler context cancellation received")
				// context is cancelled drop everything and fuck off
				close(subpathRouteChan)
				return
			case subpathRouteChan <- subpathRoutes{responses: subpathRespChan, routes: routes}:
			}
		}
	}()
	expect := len(rm) + 1 // we expect N+1 baselines, since we do the docroot by default always

	chunks := http.AcquireChunkedRoutes()
	defer http.ReleaseChunkedRoutes(chunks)

	// allocation created here because its a variable length growing slice
	// this will share the baselines across all the subpaths for a target
	// preallocate 1 because we should have always at least 1 wildcard response
	// in theory, if we have only 1 wildcard response, then this allocation is on the stack
	baselines := make(WildcardResponses, 0, 1)

	for i := 0; i < expect; i++ {
		spr, ok := <-subpathRouteChan
		if !ok {
			// if the channel was closed, then the context above us was cancelled. so we can just exit
			return nil
		}

		// reset the baselines for a subpath
		// baselines = baselines[:0]

		// we expect N responses for the number of preflight check routes
		for _ = range config.PreflightCheckRoutes {
			requestsSent++
			// ensure we drain all the responses, so don't do the context check here
			resp := <-spr.responses
			if resp.err != nil {
				log.Debug().Err(resp.err).Bytes("target", target.Bytes()).Msg("failed preflight check")
			} else {
				baselines, _ = baselines.UniqueAdd(resp.baseline)
			}
			releaseSubpathBaseline(resp)
		}
		releaseSubpathBaselineChan(spr.responses)

		if len(config.PreflightCheckRoutes) > 0 && len(baselines) == 0 {
			// if none of our preflight requests suceeded, then we're in a situation where the host didnt respond
			// we can cancel the target context so our child goroutine terminates
			// log.Debug().Str("target", target.String()).Msg("failed all preflight checks. cancelling")
			target.Cancel()

			// just drop the messages inflight on the floor and let golang clean it up. Too hard to manage how many
			// messages are actually inflight
			return ErrFailedPreflight{}
		}

		if config.QuarantineThreshold > 0 && target.Quarantined() {
			target.Cancel()
			return ErrTargetQuarantined
		}

		// TODO: implement a better handling of the quarantine reset. This may desync/race against the existing inflight
		// requests. The quarantine should be applied on each subpath not the entire target
		// we should alternatively have a global target quarantine
		target.QuarantineReset()

		// TODO: move this up to the global map of routes, we don't need to chunk every time.
		// this work can be shared across all targets
		*chunks = (*chunks)[:0]
		chunks = http.ChunkRoutes(spr.routes, chunks, config.MaxConnPerHost)
		// we will send out all the routes to the workers. the receiver should drain it for us if a context cancellation arrives
		for _, chunk := range *chunks {
			t := acquireReqMsg()
			t.typ = RequestMsg
			t.Job = acquireJob()
			t.Job.t = target
			t.Job.routes = append(t.Job.routes, chunk...)
			t.Job.wcr = append(t.Job.wcr, baselines...)
			t.Job.client = c

			requestsSent += len(chunk)
			reqChan <- t
		}
	}

	// release resources for this function call
	releaseSubpathRoutesChan(subpathRouteChan)
	return nil
}

// These errors correspond to errors resulting from failing various response validation checks.
var (
	ErrLengthMatch             = fmt.Errorf("failed on content length check")
	ErrScaledLengthMatch       = fmt.Errorf("failed on adjusted content length check")
	ErrWordCountMatch          = fmt.Errorf("failed on word and line count match")
	ErrContentLengthRangeMatch = fmt.Errorf("failed on content length range match")
	ErrBlacklistedStatusCode   = fmt.Errorf("failed with blacklisted status code")
	ErrWhitelistedStatusCode   = fmt.Errorf("failed with not whitelisted status code")
)

// handleRequest is a convenience abstraction for handling the job passed to a job worker
// this will consume the job, send the request and send the job back through its provided source channel
// valid results will be passed to the result channel
// this will process the slice of routes. we perform a slice of routes to avoid having to wait on the channel
// for each individual route
// HandleRequest will early exit from processing a slice of routes if the context is cancelled
func handleRequest(ctx context.Context, j *job, resChan chan *Result, config *Config) error {
	// log.Trace().Str("request", req.String()).Msg("received host")
	req := http.Request{Target: j.t}

routeloop:
	for idx, route := range j.routes {
		// Extra Line added to fix Delay function
		if config.Delay > 0 {
			time.Sleep(config.Delay)
		}
		// Extra Line added to fix Delay function
		
		// periodically check if the channel has terminated so we can exit
		// TODO: not sure how this affects branch prediction on the loop, or whether there's a noticable performance impact
		// i presume not, since 99% of the time we're waiting on network
		// we +1 so we don't do this check on the first loop
		if (idx+1)%CheckInterval == 0 {
			select {
			case <-ctx.Done():
				break routeloop
			default: // need a get out of jail free card
			}

			if config.QuarantineThreshold > 0 && j.t.Quarantined() {
				break routeloop
			}
		}

		req.Route = route

		j.t.HitIncr()
		config.ProgressBar.Incr(1)

		resp, err := http.DoClient(j.client, req, &config.HTTP)
		if err != nil {
			log.Debug().Err(err).
				Bytes("request", req.Target.Bytes()).
				Int("status", resp.StatusCode).
				Str("func", "handleRequest").
				Msg("failed request")
			// do not early exit as all the steps afterwards must be executed

			// TODO: implement alternative queue for quarantined host
			// If the request fails, also consider it for quarantine. The host might be down or we're getting blocked
			v := j.t.QuarantineIncr()
			if v > config.QuarantineThreshold {
				// log.Trace().Bytes("target", j.t.Bytes()).Msg("quarantining host")
				j.t.Quarantine()
			}
			break
		} else {
			// we have to pass all validators before we can consider it a valid request
			for _, v := range config.RequestValidators {
				if err := v.Validate(resp, j.wcr, config); err != nil {
					j.t.QuarantineReset()

					log.Debug().Err(err).
						Bytes("target", j.t.Bytes()).
						Bytes("path", route.Path).
						Int("status", resp.StatusCode).
						Msg("request was not valid. discarding")

					// this request was not a match, so release all chained requests that were part of it
					for r := resp.Next; r != nil; {
						next := r.Next
						http.ReleaseResponse(r)
						r = next
					}
					continue routeloop
				}
			}

			// if QuarantineThreshold requests all fail the check, then we've hit firewalling or a wildcard host
			// so we can just skip over it
			v := j.t.QuarantineIncr()
			if v > config.QuarantineThreshold {
				// log.Trace().Bytes("target", j.t.Bytes()).Msg("quarantining host")
				j.t.Quarantine()
			}

			// We don't expect the result to be deallocated for us. We make a copy of the resp so we never accidentally
			// deallocate it
			log.Trace().Err(err).Bytes("target", j.t.Bytes()).Bytes("path", route.Path).Msg("succesful request. sending to result")
			ret := AcquireResult()
			ret.Target = j.t
			ret.Response = resp
			ret.Route = route
			resChan <- ret
		}
	}

	// done with the request
	releaseJob(j)
	return nil
}

// RunAsync will begin all the concurrent threads for scanning. Inputs should be fed to the tx (to transmit) and results
// are read off rx (receive). This function can fail if the config provided to the engine is invalid
// Each call instantiates its own set of workers, tx, and rx. Hence this is safe to call concurrently
// The Engine will terminate when the context is cancelled, or when the tx channel is closed
// When the rx channel is closed, all results have been returned.
// The caller closing rx may panic and is considered unexpected behaviour
func (e *Engine) RunAsync(ctx context.Context) (tx chan *http.Target, rx chan *Result, err error) {
	if err := e.config.Validate(); err != nil {
		return nil, nil, fmt.Errorf("failed to start. invalid settings: %w", err)
	}

	ctx, _ = context.WithCancel(ctx)

	var (
		maxMessages = e.config.MaxParallelHosts * e.config.MaxConnPerHost
		input       = make(chan *http.Target, e.config.MaxParallelHosts)
		output      = make(chan *Result, maxMessages)

		targetWg  sync.WaitGroup
		requestWg sync.WaitGroup
	)

	for i := 0; i < e.config.MaxParallelHosts; i++ {
		requests := make(chan *ReqMsg, e.config.MaxConnPerHost)

		targetWg.Add(1)
		go func() {
			defer targetWg.Done()
			defer close(requests)
			for {
				select {
				case <-ctx.Done():
					// log.Trace().Err(ctx.Err()).Str("goroutine", "target worker").Msg("context cancellation received")
					return
				case t, ok := <-input:
					if !ok {
						// log.Trace().Err(ctx.Err()).Str("goroutine", "target worker").Msg("tx channel closed")
						return
					}

					if err := handleTarget(ctx, t, requests, e.routes, e.config); err != nil {
						// ignore all the errors
						log.Debug().Err(err).
							Str("goroutine", "target worker").
							Bytes("target", t.Bytes()).
							Msg("failed to handle target")
						if errors.Is(err, ErrTargetQuarantined) {
							log.Info().Bytes("target", t.Bytes()).Msg("Target quarantined")
						}
					}
				}
			}
		}()

		for i := 0; i < e.config.MaxConnPerHost; i++ {
			requestWg.Add(1)
			// this worker will only exit when the requests channel is closed.
			go func() {
				defer requestWg.Done()
				for r := range requests {
					// handle context cancellations
					select {
					case <-ctx.Done():
						// start sinkholing messages after context is cancelled to speed up the process
						if r.typ == PreflightMsg {
							r.Preflight.resp <- r.Preflight
						}
						continue
					default:
					}

					switch r.typ {
					case PreflightMsg:
						// manually inline this because there's nothing to do
						e.config.ProgressBar.Incr(1)
						r.Preflight.baseline, r.Preflight.err = preflightCheck(r.Preflight.route, r.Preflight.target, e.config, r.Preflight.base)
						r.Preflight.resp <- r.Preflight
					case RequestMsg:
						if err := handleRequest(ctx, r.Job, output, e.config); err != nil {
							log.Error().Err(ctx.Err()).Str("goroutine", "request worker").Msg("failed to handle request")
						}
					}

					releaseReqMsg(r)
				}
			}()
		}
	}

	// Start our worker closer thread.
	// this will wait until all the target workers have terminated (either due to context cancellation, or due to tx closing)
	// then this will close the request channel to force the request workers to terminate
	// after request workers terminate, this will close the output channel
	go func() {
		// wait until the targets have all terminated
		targetWg.Wait()
		requestWg.Wait()
		close(output)
		log.Debug().Str("goroutine", "worker closer").Msg("output and requests closed")
	}()

	return input, output, nil
}
