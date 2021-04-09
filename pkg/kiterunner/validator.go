package kiterunner

import (
	"fmt"

	"github.com/assetnote/kiterunner/pkg/convert"
	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/assetnote/kiterunner/pkg/log"
)

// RequestValidator is an interface that lets you add custom validators for what are good and bad responses
type RequestValidator interface {
	Validate(r http.Response, wildcardResponses []WildcardResponse, c *Config) error
}

type KnownBadSitesValidator struct{}

var (
	ErrGoogleBadRequest = fmt.Errorf("google bad request found")
	ErrAmazonGatewayBadRequest = fmt.Errorf("amazon gateway bad request found")
)

func (v *KnownBadSitesValidator) Validate(r http.Response, wildcardResponses []WildcardResponse, c *Config) error {
	if v == nil {
		return nil
	}
	// occurs with body + method mismatch (get with post body)
	if r.StatusCode == 400 &&
		r.BodyLength == 1555 &&
		r.Words == 82 &&
		r.Lines == 12 {
		return ErrGoogleBadRequest
	}

	// {"message":"Authorization header cannot be empty: ''"}
	if r.StatusCode == 403 &&
		r.Lines == 1 &&
		r.Words == 6 &&
		r.BodyLength == 54 {
		if len(r.Headers) > 0 {
			for _, v := range r.Headers {
				if v.Key == "X-Amzn-Requestid" {
					return ErrAmazonGatewayBadRequest
				}
			}
		}
		return ErrAmazonGatewayBadRequest
	}

	// {"message":"Authorization header requires 'Credential' parameter. Authorization header requires 'Signature' parameter. Authorization header requires 'SignedHeaders' parameter. Authorization header requires existence of either a 'X-Amz-Date' or a
	// 'Date' header. Authorization=29547877"}
	if r.StatusCode == 403&&
		r.Lines == 1 &&
		r.Words == 28 &&
		r.BodyLength >= 277 {
		if len(r.Headers) > 0 {
			for _, v := range r.Headers {
				if v.Key == "X-Amzn-Requestid" {
					return ErrAmazonGatewayBadRequest
				}
			}
		}
		return ErrAmazonGatewayBadRequest
	}

	// {"message":"'UEh6T0NPYkhOY3JSdGlmNDoxTUZ4WXExREg2bnBVR1Bi' not a valid key=value pair (missing equal-sign) in Authorization header: 'Basic UEh6T0NPYkhOY3JSdGlmNDoxTUZ4WXExREg2bnBVR1Bi'."}
	if r.StatusCode == 403&&
		r.Lines == 1 &&
		r.Words == 13 &&
		r.BodyLength >= 99 {
		if len(r.Headers) > 0 {
			for _, v := range r.Headers {
				if v.Key == "X-Amzn-Requestid" {
					return ErrAmazonGatewayBadRequest
				}
			}
		}
		return ErrAmazonGatewayBadRequest
	}


	return nil
}

type WildcardResponseValidator struct{}

func (v *WildcardResponseValidator) Validate(r http.Response, wildcardResponses []WildcardResponse, c *Config) error {
	if v == nil {
		return nil
	}

	// Not all paths provided to us will have a prefixing slash
	// TODO: precalculate this so it doesnt need to be done everytime
	basePathLen := len(r.OriginRequest.Route.Path)

	if basePathLen > 0 && r.OriginRequest.Route.Path[0] == '/' {
		basePathLen -= 1
	}

	// ignore / as a root path for wildcard detection. sometimes is helpful to see it once
	// disabled because its noisy
	if false && basePathLen == 0 {
		return nil
	}

	for _, wr := range wildcardResponses {
		// perform our wildcard detection check
		if r.StatusCode == wr.DefaultStatusCode ||
			(r.StatusCode-wr.DefaultStatusCode < 50) { // handle an edge case where we get load balanced. and
			// the load balanced servers respond on different statuscodes but with the same body

			expectedAdjustedLength := wr.AdjustedContentLength + (wr.AdjustmentScale * basePathLen)

			if r.BodyLength == wr.DefaultContentLength {
				log.Trace().Int("len", len(r.Body)).
					Msg("failed on length match")
				return ErrLengthMatch
			} //  if we have a perfect match on length

			if r.BodyLength == expectedAdjustedLength { // if we have a match on scaled length
				log.Trace().Int("adjustedLen", expectedAdjustedLength).
					Int("len", len(r.Body)).
					Msg("failed on scaled length match")
				return ErrScaledLengthMatch
			}
			// TODO: benchmark whether this is an effective mechanism
			if r.Words == wr.DefaultWordCount &&
				r.Lines == wr.DefaultLineCount {
				log.Trace().
					Int("words", r.Words).
					Int("lines", r.Lines).
					Msg("failed on line/word count match")
				return ErrWordCountMatch
			}

			log.Trace().Int("adjustedLen", expectedAdjustedLength).
				Bytes("basepath", r.OriginRequest.Route.Path).
				Int("basepathlen", basePathLen).
				Int("len", r.BodyLength).
				Bytes("body", r.Body). // TODO: disable this allocation
				Int("statusCode", r.StatusCode).
				Int("expectedSC", wr.DefaultStatusCode).
				Int("words", r.Words).
				Int("lines", r.Lines).
				Int("expectedWords", wr.DefaultWordCount).
				Int("expectedLines", wr.DefaultLineCount).
				Int("baselen", wr.AdjustedContentLength).
				Msg("passed wildcard test")
		}
	}
	return nil
}

type ContentLengthValidator struct {
	IgnoreRanges []http.Range
}

func NewContentLengthValidator(ranges []http.Range) *ContentLengthValidator {
	if len(ranges) == 0 {
		return nil
	}
	return &ContentLengthValidator{
		IgnoreRanges: ranges,
	}
}

func (v ContentLengthValidator) String() string {
	return fmt.Sprintf("ContentLengthValidator{%v}", v.IgnoreRanges)
}

func (v *ContentLengthValidator) Validate(r http.Response, _ []WildcardResponse, _ *Config) error {
	if v == nil {
		return nil
	}
	for _, v := range v.IgnoreRanges {
		if v.Min <= r.BodyLength && r.BodyLength <= v.Max {
			return ErrContentLengthRangeMatch
		}
	}
	return nil
}

type StatusCodeWhitelist struct {
	Codes map[int]interface{}
}

func NewStatusCodeWhitelist(valid []int) *StatusCodeWhitelist {
	if len(valid) == 0 {
		return nil
	}
	ret := &StatusCodeWhitelist{
		Codes: make(map[int]interface{}),
	}
	for _, v := range valid {
		ret.Codes[v] = struct{}{}
	}

	return ret
}

func (v StatusCodeWhitelist) String() string {
	return fmt.Sprintf("StatusCodeWhitelist{%v}", convert.IntMapToSlice(v.Codes))
}

func (v *StatusCodeWhitelist) Validate(r http.Response, _ []WildcardResponse, _ *Config) error {
	if v == nil {
		return nil
	}
	// only consider the whitelist if its populated
	if v.Codes != nil && len(v.Codes) != 0 {
		// we're not in the whitelist
		if _, ok := v.Codes[r.StatusCode]; !ok {
			return ErrWhitelistedStatusCode
		}
	}
	return nil
}

type StatusCodeBlacklist struct {
	Codes map[int]interface{}
}

func NewStatusCodeBlacklist(valid []int) *StatusCodeBlacklist {
	if len(valid) == 0 {
		return nil
	}
	ret := &StatusCodeBlacklist{
		Codes: make(map[int]interface{}),
	}
	for _, v := range valid {
		ret.Codes[v] = struct{}{}
	}

	return ret
}

func (v StatusCodeBlacklist) String() string {
	return fmt.Sprintf("StatusCodeBlacklist{%v}", convert.IntMapToSlice(v.Codes))
}

func (v *StatusCodeBlacklist) Validate(r http.Response, _ []WildcardResponse, _ *Config) error {
	if v == nil {
		return nil
	}
	// only consider the whitelist if its populated
	if v.Codes != nil && len(v.Codes) != 0 {
		// we're in the blacklist
		if _, ok := v.Codes[r.StatusCode]; ok {
			return ErrBlacklistedStatusCode
		}
	}
	return nil
}
