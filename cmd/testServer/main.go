package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

const (
	responseSize = 1024
)

var (
	requestCount count32
)

type count32 struct {
	val uint32
}

func (c *count32) increment() {
	atomic.AddUint32(&c.val, 1)
}

func (c *count32) get() uint32 {
	return atomic.LoadUint32(&c.val)
}

func PreRequest() {
	time.Sleep(0* time.Millisecond)
	requestCount.increment()
}

func Index(ctx *fasthttp.RequestCtx) {
	PreRequest()

	ctx.WriteString("Welcome!")
}

func ASDFResponder(ctx *fasthttp.RequestCtx) {
	PreRequest()

	ctx.WriteString("asdf!")
}

func Hello(ctx *fasthttp.RequestCtx) {
	PreRequest()

	fmt.Fprintf(ctx, "Hello, %s!\n", ctx.UserValue("name"))
}

func WildcardResponder(ctx *fasthttp.RequestCtx) {
	PreRequest()

	fmt.Fprintf(ctx, "get %s\n", ctx.RequestURI())
	// log.Info().Msgf("got: %s", ctx.RequestURI())
	ctx.SetStatusCode(200)
}

func RedirectResponder(ctx *fasthttp.RequestCtx) {
	PreRequest()

	fmt.Fprintf(ctx, "go to, %s!\n", ctx.UserValue("dest"))
	ctx.SetStatusCode(302)
	ctx.Response.Header.Add("location", "/"+ctx.UserValue("dest").(string))
}

func UserWildcardResponder(ctx *fasthttp.RequestCtx) {
	PreRequest()

	log.Info().
		Bytes("method",ctx.Method()).
		Bytes("uri", ctx.RequestURI()).Msg("got user request")
	switch string( ctx.Method() ) {
	case "GET":
		fmt.Fprintf(ctx, "get %s user woo\n", ctx.RequestURI())
	default:
		ctx.SetStatusCode(302)
		ctx.Response.Header.Add("location", string(ctx.RequestURI()))
	}
}


func APIWildcardResponder(ctx *fasthttp.RequestCtx) {
	PreRequest()

	fmt.Fprintf(ctx, "get %s\n", ctx.RequestURI())
}

func StatsFunc(end <-chan bool) {
	// rolling average
	lastRequest := time.Now()
	lastRequestCount := requestCount.get()
	rpsPeak := float64(0)
	for {
		select {
		case <-end:
			fmt.Println("\nTerminating.")
			return
		default:
			timeDiff := time.Since(lastRequest).Seconds()
			curRequestCount := requestCount.get()
			requestCountDiff := curRequestCount - lastRequestCount
			rps := float64(requestCountDiff) / timeDiff
			if rps > rpsPeak {
				rpsPeak = rps
			}

			fmt.Printf("Total Requests: %d. Requests since last checkin: %d. RPS: %f. Peak: %f\t\t\t\t\r", curRequestCount, requestCountDiff, rps, rpsPeak)
			lastRequest = time.Now()
			lastRequestCount = curRequestCount
			time.Sleep(1 * time.Second)
		}
	}
}

func main() {
	var portRange string
	flag.StringVar(&portRange, "p", "14000-14500", "Range of ports to start servers on")
	flag.Parse()

	flagParts := strings.Split(portRange, "-")
	if len(flagParts) != 2 {
		log.Fatal().Msg("Invalid portRange. Format should be <int>-<int>")
	}

	startPort, err := strconv.Atoi(flagParts[0])
	if err != nil {
		log.Fatal().Msgf("Unable to parse port: %s", err)
	}

	endPort, err := strconv.Atoi(flagParts[1])
	if err != nil {
		log.Fatal().Msgf("Unable to parse port: %s", err)
	}

	r := router.New()
	r.GET("/", Index)
	r.GET("/_search", ASDFResponder)
	r.GET("/hello/:name", Hello)
	r.GET("/redir/{dest:*}", RedirectResponder)
	r.GET("/api/{req:*}", APIWildcardResponder)
	r.GET("/api/user/{req:*}", UserWildcardResponder)
	r.POST("/api/user/{req:*}", UserWildcardResponder)
	r.Handle("*", "/{req:*}", WildcardResponder)

	var wg sync.WaitGroup
	// Need just 1 more port for integration tests to pass
	wg.Add(1)
	go func(port int) {
		Host := fmt.Sprintf(":%d", port)
		// log.Printf("Starting server on %s", Host)
		log.Fatal().Err(fasthttp.ListenAndServe(Host, r.Handler)).Msg("failed to start server")
		wg.Done()
	}(9200)

	for i := startPort; i < endPort; i++ {
		wg.Add(1)
		go func(port int) {
			Host := fmt.Sprintf(":%d", port)
			// log.Printf("Starting server on %s", Host)
			log.Fatal().Err(fasthttp.ListenAndServe(Host, r.Handler)).Msg("failed to start server")
			wg.Done()
		}(i)
	}
	statsFunc := make(chan bool, 0)

	go StatsFunc(statsFunc)
	wg.Wait()

	statsFunc <- true
	close(statsFunc)
}
