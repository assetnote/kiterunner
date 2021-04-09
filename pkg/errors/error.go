package errors

import (
	"errors"
	"fmt"

	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/hashicorp/go-multierror"
)

// prefixfromDepth will create the indent prefix for a certain depth
// of string, e.g. 2 will yield "  " * 2 -> "    "
func prefixFromDepth(depth int) string {
	var p []byte
	for i := 0; i < depth; i++ {
		p = append(p, "  "...)
	}
	return string(p)
}

// PrintError will attempt to traverse the nested error and
// recursively print out any nested ParserErrors found
// If a multierror.Error is found, we will recurisvely print out
// each error found
func PrintError(err error, depth int) {
	var (
		merr *multierror.Error
		perr *ParserError
	)

	if errors.As(err, &merr) {
		for _, v := range merr.Errors {
			PrintError(v, depth+1)
		}
	} else if errors.As(err, &perr) {
		perr.LogError(depth)
	} else {
		log.Debug().Err(err).Msg(prefixFromDepth(depth) + "error")
	}
}

// ParserError encapsulates the contextual error relating to parsing an API schema
// The fields can be arbitrarily used to represent whatever information you wish
type ParserError struct {
	ID      string // ID corresponds to the KSUID for the API allowing you to backref which API the error came from
	Method  string // Method corresponds to the method for the request at the top level of the API
	Route   string // Route corresponds to the route for the request
	RawJSON []byte // RawJSON optionally can include the raw json for the component if there was a JSON parsing error
	Err     error // Err includes the error. If this is a wrapped error, then Error() will not print out this field
	Context string // Context is an arbitrary context field that you can use to add helpful text explaining the error
}

// Error will return the string representation of the error. If the error
// is wrapped, we omit printing the wrapped error, allowing the user to determine
// how to display the wrapped error.
func (p *ParserError) Error() string {
	if err := errors.Unwrap(p.Err); err != nil {
		// have a nested error. so we shouldn't include all the context
		return fmt.Sprintf("parserError [%s %s %s]: %s", p.ID, p.Method, p.Route, p.Context)
	}
	return fmt.Sprintf("parserError [%s %s %s]: %s: %s", p.ID, p.Method, p.Route, p.Context, p.Err.Error())
}

// LogError will log to Debug() the context surrounding the error.
// the depth argument modifies the indentation depth of the pretty printed error
// If RawJSON is included, we will add the RawJSON to the logging
func (p *ParserError) LogError(depth int) {
	var (
		merr *multierror.Error
		perr *ParserError
	)
	base := log.Debug().
		Str("ID", p.ID).
		Str("Method", p.Method).
		Str("Route", p.Route).
		Str("Context", p.Context)

	// skip printing the raw json since its heaps noisy
	if p.RawJSON != nil {
		if len( p.RawJSON ) < 100 {
			base = base.RawJSON("JSON", p.RawJSON)
		}
	}

	if errors.As(p.Err, &merr) {
		base.Err(perr).Msg(prefixFromDepth(depth))
		PrintError(merr, depth+1)
	} else if errors.As(p.Err, &perr) {
		base.Err(perr.Err).Msg(prefixFromDepth(depth))
		perr.LogError(depth + 1)
	} else {
		base.Err(p.Err).Msg(prefixFromDepth(depth))
	}
}
