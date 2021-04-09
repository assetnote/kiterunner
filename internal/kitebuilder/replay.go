package kitebuilder

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	nethttp "net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/assetnote/kiterunner/pkg/proute"
)

func Replay(ctx context.Context, kitefile string, kuid string, method string, path string, host string, proxy string) error {
	log.Debug().Fields(map[string]interface{}{
		"kitefile": kitefile,
		"kuid":     kuid,
		"method":   method,
		"path":     path,
		"host":     host,
		"proxy":    proxy,
	}).Msg("replaying")

	apis, err := proute.DecodeAPIProtoFile(kitefile)
	if err != nil {
		return fmt.Errorf("failed to decode kite file: %w", err)
	}

	foundKsuid := false
	foundPath := false
	foundMethod := false

	longestPath := ""
	var data []byte

	var route *http.Route
apisearch:
	for _, v := range apis {
		if v.ID != kuid {
			continue
		}

		foundKsuid = true
		for _, vv := range v.Routes {
			m := strings.TrimSpace(strings.ToUpper(vv.Method))
			if m != strings.TrimSpace(strings.ToUpper(method)) {
				continue
			}
			foundMethod = true

			p := strings.SplitN(vv.TemplatePath, "{", 2)[0]
			if !strings.Contains(path, p) {
				continue
			}
			foundPath = true

			if len(p) <= len(longestPath) {
				continue
			}

			longestPath = p

			r, err := vv.ToKiterunner(v.Headers(true)...)
			if err != nil {
				log.Error().Err(err).Msg("failed to generate route")
			}
			route = r
			break apisearch
		}
	}

	if route == nil {
		if !foundKsuid {
			return fmt.Errorf("unable to find ksuid")
		}
		if !foundPath {
			return fmt.Errorf("unable to find path")
		}
		if !foundMethod {
			return fmt.Errorf("unable to find method")
		}
		return fmt.Errorf("unexpected no result found")
	}

	log.Info().Msg("Raw reconstructed request")
	data = route.AppendBytes(data[:0])
	fmt.Println(string(data))

	if host != "" {
		t := &nethttp.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}

		if proxy != "" {
			parsedProxy, err := url.Parse(proxy)
			if err != nil {
				return fmt.Errorf("failed to parse proxy url: %w", err)
			}
			t.Proxy = nethttp.ProxyURL(parsedProxy)
		}

		c := &nethttp.Client{
			Timeout:   10 * time.Second,
			Transport: t,
		}

		reader := bytes.NewReader(route.Body)
		fullurl := host + path
		if len(route.Query) > 0 {
			fullurl = fullurl + "?" + string(route.Query)
		}

		req, err := nethttp.NewRequest(string(route.Method), fullurl, reader)
		if err != nil {
			return fmt.Errorf("failed to build request: %w", err)
		}
		for _, v := range route.Headers {
			req.Header.Add(v.Key, v.Value)
		}

		if len(route.Body) > 0 {
			req.Body = ioutil.NopCloser(bytes.NewReader(route.Body))
		}

		raw, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			return fmt.Errorf("failed to dump request: %w", err)
		}

		log.Info().Msg("Outbound request")
		fmt.Println(string(raw))

		resp, err := c.Do(req)
		if err != nil {
			return fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		data, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return fmt.Errorf("failed to dump response: %w", err)
		}

		log.Info().Msg("Response After Redirects")
		fmt.Println(string(data))
	}

	return nil
}
