package cmd

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"
	"time"

	"github.com/assetnote/kiterunner/internal/scan"
	"github.com/assetnote/kiterunner/pkg/context"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/spf13/cobra"
)

var (
	kitebuilderFiles    = []string{}
	kitebuilderFullScan = false
	headers             = []string{}

	failStatusCodes    = []int{}
	successStatusCodes = []int{}
	lengthIgnoreRange  = []string{}

	progressBar               = true
	disablePrecheck           = false
	wildcardDetection         = true
	maxConnPerHost            = 3
	maxParallelHosts          = 50
	delay                     = 0 * time.Second
	userAgent                 = "Chrome. Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/88.0.4324.96 Safari/537.36"
	quarantineThreshold int64 = 10

	timeout      = 3 * time.Second
	maxRedirects = 3

	preflightDepth   int64 = 1
	blacklistDomains       = []string{}
	filterAPIs             = []string{}

	forceMethod = ""

	profileName = ""

	assetnoteWordlist = []string{}
)

// scanCmd represents the scan command
var scanCmd = &cobra.Command{
	Use:   "scan INPUT [ -w wordlist.kite ]",
	Short: "scan one or multiple hosts with a provided wordlist",
	Long: `this will perform a concurrent scan of one or multiple hosts
using a generate kiterunner wordlist.
We will attempt to find a file matching your provided <input>, and otherwise
attempt to parse it as a URI. 
If protocol is missing, then we will assume from the port.
If the port is missing, then we will try both http:80 and https:443

The kitebuilder file format is a modified openAPI schema that allows you to specify
arguments, parameters, headers, methods and body structure for structured api calls.
We can load an kitebuilder file in as the wordlist. 
By default, we perform a 2 phase kitebuilder scan. The first phase uses a single route for api schema.
If any of the routes respond, we perform a second phase scan on the host where all the routes for an api
are scanned

usage: 
kr scan <input> <flags>
kr scan hosts.txt -A=apiroutes-210228:5000 
kr scan domain.com -w wordlist.kite
kr scan domains.txt -W rafter.txt -D=0 # this just uses the words as a normal wordlist, disables depth scanning

`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		domain := args[0]

		opts := []scan.ScanOption{
			scan.MaxParallelHosts(maxParallelHosts),
			scan.MaxConnPerHost(maxConnPerHost),
			scan.MaxRedirects(maxRedirects),
			scan.ContentLengthIgnoreRanges(lengthIgnoreRange),
			scan.Timeout(timeout),
			scan.Delay(delay),
			scan.AddHeaders(headers),
			scan.LoadKitebuilderFile(kitebuilderFiles),
			scan.KitebuilderFullScan(kitebuilderFullScan),
			scan.LoadAssetnoteWordlistKitebuilder(assetnoteWordlist),
			scan.ForceMethod(forceMethod),
			scan.UserAgent(userAgent),
			scan.SuccessStatusCodes(successStatusCodes),
			scan.FailStatusCodes(failStatusCodes),
			scan.BlacklistDomains(blacklistDomains),
			scan.FilterAPIs(filterAPIs),
			scan.WildcardDetection(wildcardDetection),
			scan.ProgressBarEnabled(progressBar),
			scan.QuarantineThreshold(quarantineThreshold),
			scan.PreflightDepth(preflightDepth),
			scan.Precheck(!disablePrecheck),
		}

		go func() {
			log.Debug().Err(http.ListenAndServe("localhost:6060", nil)).Msg("Started http profiler server")
		}()

		if profileName != "" {
			f, err := os.Create(profileName)
			if err != nil {
				log.Fatal().Err(err).Msg("failed to create profile")
			}

			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
		}

		if domain == "-" {
			if err := scan.ScanStdin(context.Context(), opts...); err != nil {
				log.Fatal().Err(err).Msg("failed to read from stdin")
			}
		} else {
			if err := scan.ScanDomainOrFile(context.Context(), domain, opts...); err != nil {
				log.Fatal().Err(err).Msg("failed to scan domain")
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)

	scanCmd.Flags().StringSliceVarP(&kitebuilderFiles, "kitebuilder-list", "w", kitebuilderFiles, "ogl wordlist to use for scanning")
	scanCmd.Flags().BoolVar(&kitebuilderFullScan, "kitebuilder-full-scan", kitebuilderFullScan, "perform a full scan without first performing a phase scan.")
	//scanCmd.Flags().StringSliceVarP(&headers, "header", "H", []string{"x-forwarded-for: 127.0.0.1"}, "headers to add to requests")
	// Fix for improper header definition
	scanCmd.Flags().StringArrayVarP(&headers, "header", "H", []string{"x-forwarded-for: 127.0.0.1"}, "headers to add to requests")

	scanCmd.Flags().BoolVar(&disablePrecheck, "disable-precheck", false, "whether to skip host discovery")

	scanCmd.Flags().IntVarP(&maxConnPerHost, "max-connection-per-host", "x", maxConnPerHost, "max connections to a single host")
	scanCmd.Flags().IntVarP(&maxParallelHosts, "max-parallel-hosts", "j", maxParallelHosts, "max number of concurrent hosts to scan at once")
	scanCmd.Flags().DurationVar(&delay, "delay", delay, "delay to place inbetween requests to a single host")
	scanCmd.Flags().StringVar(&userAgent, "user-agent", userAgent, "user agent to use for requests")
	scanCmd.Flags().DurationVarP(&timeout, "timeout", "t", timeout, "timeout to use on all requests")
	scanCmd.Flags().IntVar(&maxRedirects, "max-redirects", maxRedirects, "maximum number of redirects to follow")
	scanCmd.Flags().StringVar(&forceMethod, "force-method", forceMethod, "whether to ignore the methods specified in the ogl file and force this method")

	scanCmd.Flags().IntSliceVar(&successStatusCodes, "success-status-codes", successStatusCodes,
		"which status codes whitelist as success. this is the default mode")
	scanCmd.Flags().IntSliceVar(&failStatusCodes, "fail-status-codes", failStatusCodes,
		"which status codes blacklist as fail. if this is set, this will override success-status-codes")

	scanCmd.Flags().StringSliceVar(&blacklistDomains, "blacklist-domain", blacklistDomains, "domains that are blacklisted for redirects. We will not follow redirects to these domains")
	scanCmd.Flags().BoolVar(&wildcardDetection, "wildcard-detection", wildcardDetection, "can be set to false to disable wildcard redirect detection")
	scanCmd.Flags().StringSliceVar(&lengthIgnoreRange, "ignore-length", lengthIgnoreRange, "a range of content length bytes to ignore. you can have multiple. e.g. 100-105 or 1234 or 123,34-53. This is inclusive on both ends")

	scanCmd.Flags().BoolVar(&progressBar, "progress", progressBar, "a progress bar while scanning. by default enabled only on Stderr")
	scanCmd.Flags().Int64Var(&quarantineThreshold, "quarantine-threshold", quarantineThreshold, "if the host return N consecutive hits, we quarantine the host as wildcard. Set to 0 to disable")

	scanCmd.Flags().Int64VarP(&preflightDepth, "preflight-depth", "d", 1, "when performing preflight checks, what directory depth do we attempt to check. 0 means that only the docroot is checked")
	scanCmd.Flags().StringVar(&profileName, "profile-name", profileName, "name for profile output file")

	scanCmd.Flags().StringSliceVar(&filterAPIs, "filter-api", filterAPIs, "only scan apis matching this ksuid")

	scanCmd.Flags().StringSliceVarP(&assetnoteWordlist, "assetnote-wordlist", "A", assetnoteWordlist, "use the wordlists from wordlist.assetnote.io. specify the type/name to use, e.g. apiroutes-210228. You can specify an additional maxlength to use only the first N values in the wordlist, e.g. apiroutes-210228;20000 will only use the first 20000 lines in that wordlist")
}
