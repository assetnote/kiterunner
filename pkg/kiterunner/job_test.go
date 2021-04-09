package kiterunner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWildcardResponses_AddWildcard(t *testing.T) {
	type args struct {
		wr WildcardResponse
	}
	tests := []struct {
		name     string
		w        WildcardResponses
		args     args
		expected WildcardResponses
	}{
		{"nil", nil, args{WildcardResponse{DefaultWordCount: 1}}, WildcardResponses{{DefaultWordCount: 1}}},
		{"simple", WildcardResponses{}, args{WildcardResponse{DefaultWordCount: 1}}, WildcardResponses{{DefaultWordCount: 1}}},
		{"simple + 1", WildcardResponses{{DefaultWordCount: 2}}, args{WildcardResponse{DefaultWordCount: 1}}, WildcardResponses{{DefaultWordCount: 2}, {DefaultWordCount: 1}}},
		{"dedupe", WildcardResponses{{DefaultWordCount: 1}}, args{WildcardResponse{DefaultWordCount: 1}}, WildcardResponses{{DefaultWordCount: 1}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.w, _ = tt.w.UniqueAdd(tt.args.wr)
			assert.ElementsMatch(t, tt.w, tt.expected)
		})
	}
}
