package kiterunner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/valyala/bytebufferpool"
)

type Result struct {
	Target   *http.Target
	Route    *http.Route
	Response http.Response
}

func (r *Result) String() string {
	return fmt.Sprintf("result: %v [%v] [%v]", r.Target, r.Route, r.Response)
}

func (r *Result) reset() {
	r.Target = nil
	r.Route = nil
	r.Response.Reset()
}

// Release will place the called result back into the pool. After release it is not safe for use
func (r *Result) Release() {
	ReleaseResult(r)
}

var (
	resultPool sync.Pool
)

// AcquireResult retrieves a host from the shared header pool
func AcquireResult() *Result {
	v := resultPool.Get()
	if v == nil {
		return &Result{}
	}
	return v.(*Result)
}

// ReleaseResult releases a host into the shared header pool
func ReleaseResult(h *Result) {
	h.reset()
	resultPool.Put(h)
}

func leftpadAppendBytes(buf []byte, in []byte, pad int) []byte {
	padding := pad - len(in)
	if padding > 0 {
		buf = append(buf, bytes.Repeat([]byte(" "), padding)...)
	}
	return append(buf, in...)
}

func rightpadAppendBytes(buf []byte, in []byte, pad int) []byte {
	padding := pad - len(in)
	buf = append(buf, in...)
	if padding > 0 {
		buf = append(buf, bytes.Repeat([]byte(" "), padding)...)
	}
	return buf
}

type Attribute int

// Foreground text colors
const (
	FgBlack Attribute = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite

	FgBrightBlack Attribute = 90
)

func getColor(sc int) Attribute {
	if 0 <= sc && sc <= 299 {
		return FgGreen
	} else if 300 <= sc && sc <= 399 {
		return FgCyan
	} else if 400 <= sc && sc <= 499 {
		return FgYellow
	} else if 500 <= sc && sc <= 599 {
		return FgRed
	}
	return FgWhite
}

func appendColorStatusCode(buf []byte, sc int) []byte {
	color := FgWhite
	buf = appendColorStart(buf, color)
	buf = append(buf, strconv.Itoa(sc)...)
	buf = appendColorEnd(buf)
	return buf
}

func appendColorStart(buf []byte, code Attribute) []byte {
	buf = append(buf, "\x1b["...)
	buf = append(buf, strconv.Itoa(int(code))...)
	buf = append(buf, "m"...)
	return buf
}

func appendColorEnd(buf []byte) []byte {
	buf = append(buf, "\x1b[0m"...)
	return buf
}

func appendColor(buf []byte, in []byte, code Attribute) []byte {
	buf = appendColorStart(buf, code)
	buf = append(buf, in...)
	buf = appendColorEnd(buf)
	return buf
}

func appendPaddedIntColor(b []byte, v int, padding int) []byte {
	if v == 0 {
		b = appendColorStart(b, FgBrightBlack)
	} else {
		b = appendColorStart(b, FgWhite)
	}
	b = leftpadAppendBytes(b, []byte(strconv.Itoa(v)), padding)
	b = appendColorEnd(b)
	return b
}

// <METHOD> <STATUSCODE> [lines, words, lines] <URL> [redirects ...]
func (r *Result) AppendPrettyBytes(b []byte) []byte {
	color := getColor(r.Response.StatusCode)

	// colour most of the line with our result
	b = appendColorStart(b, color)

	// Append our method
	b = rightpadAppendBytes(b, r.Route.Method, len("OPTIONS")) // this is the longest host header we got. so pad to there
	b = append(b, " "...)

	b = append(b, strconv.Itoa(r.Response.StatusCode)...)
	b = append(b, " "...)

	// append the sizes
	b = append(b, "["...)
	b = appendPaddedIntColor(b, r.Response.BodyLength, 7)
	b = append(b, ","...)
	b = appendPaddedIntColor(b, r.Response.Words, 5)
	b = append(b, ","...)
	b = appendPaddedIntColor(b, r.Response.Lines, 4)
	b = append(b, "]"...)
	b = appendColorStart(b, color)
	b = append(b, " "...)

	// Append the path
	b = r.Target.AppendBytes(b)
	b = r.Route.AppendPath(b)
	b = append(b, " "...)

	b = appendColorEnd(b)

	b = r.Response.Next.AppendRedirectChain(b)

	// append the KSUID for lookup
	b = append(b, r.Route.Source...)

	return b
}

// <METHOD> <STATUSCODE> <URL> [redirects ...]
func (r *Result) AppendBytes(b []byte) []byte {
	// Append our method
	b = rightpadAppendBytes(b, r.Route.Method, len("OPTIONS")) // this is the longest host header we got. so pad to there
	b = append(b, " "...)

	b = append(b, strconv.Itoa(r.Response.StatusCode)...)
	b = append(b, " "...)

	// append the sizes
	b = append(b, "["...)
	b = leftpadAppendBytes(b, []byte(strconv.Itoa(r.Response.BodyLength)), 7)
	b = append(b, ","...)
	b = leftpadAppendBytes(b, []byte(strconv.Itoa(r.Response.Words)), 5)
	b = append(b, ","...)
	b = leftpadAppendBytes(b, []byte(strconv.Itoa(r.Response.Lines)), 4)
	b = append(b, "]"...)
	b = append(b, " "...)

	// Append the path
	b = r.Target.AppendBytes(b)
	b = r.Route.AppendPath(b)
	b = append(b, " "...)

	b = r.Response.AppendRedirectChain(b)

	// append the ksuid for lookup
	b = append(b, r.Route.Source...)

	return b
}

// LogResultsChan will output the results using the configured logger
func LogResultsChan(ctx context.Context, res chan *Result, config *Config) {
	for {
		select {
		case <-ctx.Done():
			// log.Trace().Err(ctx.Err()).Str("goroutine", "log results").Msg("context cancellation received")
			return
		case v, ok := <-res:
			if !ok {
				return
			}

			LogResult(v, config)
		}
	}
}

// LogResults will output the results using the configured logger
func LogResults(res []*Result, config *Config) {
	for _, v := range res {
		LogResult(v, config)
	}
}

func LogResult(r *Result, config *Config) {
	// We log our results to stdout. This way we can log garbage to stderr
	if log.GetLevel() == log.DebugLevel {
		msg := bytebufferpool.Get()
		msg.B = append(msg.B, "\r"...)
		msg.B = r.AppendPrettyBytes(msg.B)
		msg.B = append(msg.B, "\n"...)
		for _, v := range r.Response.Headers {
			msg.B = v.AppendBytes(msg.B)
			msg.B = append(msg.B, "\n"...)
		}
		msg.B = append(msg.B, r.Response.Body...)
		os.Stdout.Write(msg.B)
		bytebufferpool.Put(msg)
		return
	}

	switch log.GetLogFormat() {
	case log.Text:
		msg := bytebufferpool.Get()
		msg.B = append(msg.B, "\r"...)
		msg.B = r.AppendBytes(msg.B)
		msg.B = append(msg.B, "\n"...)
		os.Stdout.Write(msg.B)
		bytebufferpool.Put(msg)
	case log.Pretty:
		msg := bytebufferpool.Get()
		msg.B = append(msg.B, "\r"...)
		msg.B = r.AppendPrettyBytes(msg.B)
		msg.B = append(msg.B, "\n"...)
		// for _, v := range r.Response.Headers {
		// 	msg.B = v.AppendBytes(msg.B)
		// 	msg.B = append(msg.B, "\n"...)
		// }
		// msg.B = append(msg.B, r.Response.Body...)
		os.Stdout.Write(msg.B)
		bytebufferpool.Put(msg)
	case log.JSON:
		log.Stdout.Log().
			Str("method", string(r.Route.Method)).
			Bytes("target", r.Target.Bytes()).
			Bytes("path", r.Route.Path).
			Array("responses", (&r.Response).Flatten()).
			Msg("")
	}
}
