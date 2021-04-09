package scan

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/assetnote/kiterunner/pkg/convert"
	"github.com/assetnote/kiterunner/pkg/http"
	"github.com/assetnote/kiterunner/pkg/kiterunner"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
)

func runWithProgress(ctx context.Context, pbEnabled bool, routes http.RouteMap, targets []*http.Target, wcopts []kiterunner.ConfigOption) ([]*kiterunner.Result, error) {
	select {
	case <-ctx.Done():
		return nil, nil
	default:
	}

	var b *ProgressBar
	max := int64(len(routes.Flatten()))*int64(len(targets)) + int64(len(kiterunner.PreflightCheckRoutes))*int64(len(routes))*int64(len(targets))
	if pbEnabled {
		b = NewProgress(max)
		wcopts = append(wcopts, []kiterunner.ConfigOption{
			kiterunner.AddProgressBar(b),
		}...)
		defer b.Requests.Finish()
	}
	e := kiterunner.NewEngine(routes, wcopts...)

	res, err := e.RunCallback(ctx, targets, kiterunner.LogResult)
	if err != nil {
		return res, fmt.Errorf("failed to scan: %w", err)
	}
	return res, nil
}

// ScanDomainOrFile will perform a scan using the domain or file provided.
// This will first attempt to read the file specified, and if not found, attempt to parse
// the input as a target.
// If you wish to read from stdin, use ScanStdin
func ScanDomainOrFile(ctx context.Context, domainOrFile string, opts ...ScanOption) error {
	targets, err := ParseInput(domainOrFile)
	if err != nil {
		return err
	}

	for _, v := range targets {
		v.SetContext(ctx)
	}

	start := time.Now()
	s := NewDefaultScanOptions()
	for _, o := range opts {
		if err := o(s); err != nil {
			return fmt.Errorf("failed to apply option: %w", err)
		}
	}
	log.Debug().Msgf("Options loaded: \n%s", s)

	// only scan what we want to scan
	routeSlice := s.FilteredRoutes()
	log.Debug().Int("routes", len(routeSlice)).Msg("loaded routes")
	wcopts := s.KiterunnerOptions()
	routes := http.GroupRouteDepth(routeSlice, s.PreflightDepth)

	krconf := kiterunner.NewDefaultConfig()
	for _, v := range wcopts {
		v(krconf)
	}

	quickRoutes := http.GroupRouteDepth(http.UniqueSource(routeSlice), s.PreflightDepth)

	if true {
		fields := map[string]interface{}{
			"total-routes":         len(routes.Flatten()),
			"max-conn-per-host":    krconf.MaxConnPerHost,
			"max-parallel-host":    krconf.MaxParallelHosts,
			"delay":                fmt.Sprintf("%s", krconf.Delay),
			"skip-preflight":       len(krconf.PreflightCheckRoutes) == 0,
			"preflight-routes":     len(krconf.PreflightCheckRoutes),
			"max-redirects":        krconf.HTTP.MaxRedirects,
			"max-timeout":          krconf.HTTP.Timeout,
			"read-headers":         krconf.HTTP.ReadHeaders,
			"read-body":            krconf.HTTP.ReadBody,
			"quarantine-threshold": krconf.QuarantineThreshold,
			"full-scan":            s.KitebuilderFullScan,
			"scan-depth":           s.PreflightDepth,
			"user-agent":           s.UserAgent,
			"full-scan-requests":   len(targets)*len(routes.Flatten()) + len(targets)*len(routes)*len(krconf.PreflightCheckRoutes),
		}
		if len(targets) == 1 {
			fields["target"] = targets[0].String()
		} else {
			fields["targets"] = len(targets)
		}

		if len(s.prouteAPIs) != 0 && !s.KitebuilderFullScan {
			fields["quick-scan-requests"] = len(quickRoutes.Flatten())*len(targets) + len(targets)*len(quickRoutes)*len(kiterunner.PreflightCheckRoutes)
		}

		if len(s.Headers) > 0 {
			ret := make([]string, 0)
			for _, v := range s.Headers {
				ret = append(ret, fmt.Sprintf("%s:%s", v.Key, v.Value))
			}
			fields["headers"] = ret
		}
		if len(s.extensions) > 0 {
			fields["dirsearch-compatability"] = s.dirsearchCompatabilityMode
			fields["extensions"] = convert.UniqueStrings(s.extensions)
		}
		if len(s.assetnoteAPINames) > 0 {
			fields["assetnote-apis"] = s.assetnoteAPINames
		}
		if len(s.kitebuilderAPINames) > 0 {
			fields["kitebuilder-apis"] = s.kitebuilderAPINames
		}
		if len(s.wordlistNames) > 0 {
			fields["wordlists"] = s.wordlistNames
		}
		if len(s.SuccessStatusCodes) > 0 {
			fields["status-code-whitelist"] = convert.IntMapToSlice(s.SuccessStatusCodes)
		}
		if len(s.FailStatusCodes) > 0 {
			fields["status-code-blacklist"] = convert.IntMapToSlice(s.FailStatusCodes)
		}
		if len(s.ContentLengthIgnoreRange) > 0 {
			ret := make([]string, 0)
			for _, v := range s.ContentLengthIgnoreRange {
				ret = append(ret, v.String())
			}
			fields["content-length-ignore"] = ret
		}

		switch log.GetLogFormat() {
		case "json":
			log.Info().Fields(fields).Msg("scan options")
		case "text":
			fallthrough
		case "pretty":
			fallthrough
		default:
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"setting", "value"})
			table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetAutoWrapText(false)
			table.SetAutoFormatHeaders(true)
			// table.SetCenterSeparator("|")
			// table.SetColumnSeparator("|")
			// table.SetRowSeparator("")
			// table.SetBorder(false)
			// table.SetTablePadding("\t") // pad with tabs
			// table.SetHeaderLine(false)
			// table.SetNoWhiteSpace(true)

			keys := make([]string, 0, len(fields))
			for k := range fields {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, v := range keys {
				table.Append([]string{v, fmt.Sprintf("%v", fields[v])})
			}
			fmt.Fprintf(os.Stderr, "\n")
			table.Render()
			fmt.Fprintf(os.Stderr, "\n")
		}
	}

	if len(s.prouteAPIs) != 0 && !s.KitebuilderFullScan {
		// perform a quick scan with minimal routes (one per source)
		log.Debug().
			Int("routes", len(quickRoutes.Flatten())).
			Int("targets", len(targets)).
			Msg("beginning quick scan")

		res, err := runWithProgress(ctx, s.ProgressBar, quickRoutes, targets, wcopts)
		if err != nil {
			return err
		}

		// common case, where the user exits the first scan
		select {
		case <-ctx.Done():
			log.Info().
				Int("results", len(res)).
				Dur("duration", time.Since(start)).
				Msg("scan complete")
			return nil
		default:
		}

		// if we got no results, then we should prompt the user
		if len(res) == 0 {
			log.Info().Msg("no results found")
			prompt := promptui.Prompt{
				Label:     "Continue Scanning with full wordlist? [y/n]",
				IsConfirm: true,
				Stdout:    os.Stderr,
			}

			v, err := prompt.Run()
			if err != nil || strings.ToLower(v) != "y" {
				log.Info().
					Dur("duration", time.Since(start)).
					Msg("scan complete")
				return nil
			}
		} else {
			// modify the final scan list to restrict the apis to use
			want := make(map[string]interface{})
			for _, v := range res {
				want[v.Route.Source] = struct{}{}
			}

			refinedRoutes := http.FilterSource(routeSlice, want)
			routes = http.GroupRouteDepth(refinedRoutes, s.PreflightDepth)

			// restrict the targets to only those that responded
			wantTargets := make(map[string]interface{})
			newt := make([]*http.Target, 0)
			for _, v := range res {
				if _, ok := wantTargets[string(v.Target.Bytes())]; ok {
					continue
				}
				wantTargets[string(v.Target.Bytes())] = struct{}{}
				newt = append(newt, v.Target)
			}
			targets = newt

			log.Info().
				Int("routes", len(refinedRoutes)).
				Int("targets", len(targets)).
				Msg("finished quick scan")
		}
	}

	log.Debug().
		Int("routes", len(routes.Flatten())).
		Int("targets", len(targets)).
		Msg("beginning scan")

	res, err := runWithProgress(ctx, s.ProgressBar, routes, targets, wcopts)
	if err != nil {
		return err
	}

	log.Info().
		Int("results", len(res)).
		Dur("duration", time.Since(start)).
		Msg("scan complete")

	return nil
}

// ScanStdin will perform a scan using the options provided, reading targets from stdin
// TODO: figure out how to do phase scanning with stdin scan
func ScanStdin(ctx context.Context, opts ...ScanOption) error {
	input, err := ParseStdin(ctx)
	if err != nil {
		return err
	}

	s := NewDefaultScanOptions()
	for _, o := range opts {
		if err := o(s); err != nil {
			return fmt.Errorf("failed to apply option: %w", err)
		}
	}
	log.Debug().Msgf("Options loaded: %+v", s)
	routeSlice := s.FilteredRoutes()
	routes := http.GroupRouteDepth(routeSlice, s.PreflightDepth)
	wcopts := s.KiterunnerOptions()

	var pb *ProgressBar
	if s.ProgressBar {
		pb = NewProgress(int64(len(routeSlice)))
		wcopts = append(wcopts, kiterunner.AddProgressBar(pb))
	}

	e := kiterunner.NewEngine(routes, wcopts...)

	// we always do the check, as we won't connect to host that are offline.
	// The disable precheck is a lie
	tx, rx, err := e.RunAsync(ctx)
	if err != nil {
		return fmt.Errorf("failed to start scan: %w", err)
	}

	go func() {
		total := 0
		for {
			select {
			case targets, ok := <-input:
				{
					if !ok && targets == nil {
						close(tx)
						return
					}

					for _, v := range targets {
						total += 1
						if pb != nil {
							pb.AddTotal(int64(len(routeSlice)))
						}
						tx <- v
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	kiterunner.LogResultsChan(ctx, rx, e.Config())
	pb.Requests.Finish()
	return nil
}
