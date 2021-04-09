package kiterunner

import (
	"context"
	"fmt"
	"testing"

	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/stretchr/testify/assert"
)

func TestKiterunnerEngineRun(b *testing.T) {
	tests := []struct {
		name  string
		input int
	}{
		{"singular-1", 1},
		{"tiny-5", 5},
		{"small-100", 100},
		{"large-500", 500},
		{"huge-1000", 1000},
	}

	count := 50
	log.SetLevelString("error")
	ctx := context.Background()
	for _, test := range tests {
		b.Run(test.name, func(t *testing.T) {
			e := NewEngine(MakeRoutes(count), MaxParallelHosts(test.input/2+1), TargetQuarantineThreshold(0))
			targets := MakeTargets(test.input)
			for _, v := range targets {
				v.ParseHostHeader()
				v.SetContext(ctx)
			}
			res, err := e.Run(ctx, targets)
			assert.Nil(t, err)
			assert.Len(t, res, test.input*count, "expected length didn't match. got: %v, want: %v", len(res), test.input*count)
		})
	}
}


// KiterunnerRunSync demonstrates how to perform a synchronous call against the kiterunner Engine.
// The results can be used as normal. Modifying the target or route that is returned may result in unexpected behaviour
// if the routes and targets are re-used in a later iteration of the run
func Example_kiterunnerRunSync() {
	ctx := context.Background()
	e := NewEngine(MakeRoutes(5), MaxParallelHosts(5), TargetQuarantineThreshold(0))
	targets := MakeTargets(5)
	for _, v := range targets {
		v.ParseHostHeader()
		v.SetContext(ctx)
	}
	res, err := e.Run(ctx, targets)
	if err != nil {
		// handle err
	}

	for _, v := range res {
		fmt.Println(v.String())
	}
}