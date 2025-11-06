package cmd

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime/pprof"

	"github.com/assetnote/kiterunner/internal/scan"
	"github.com/assetnote/kiterunner/pkg/context"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/spf13/cobra"
)

var (
	textWordlist        = []string{}
	extensions = []string{}
	dirsearchCompatabilityMode =false
)

// bruteCmd represents the scan command
var bruteCmd = &cobra.Command{
	Use:   "brute INPUT [ -w wordlist.txt ]",
	Short: "brute one or multiple hosts with a provided wordlist",
	Long: `this will perform a concurrent scan of one or multiple hosts
using a default wordlist. This can be run in dirsearch compatability mode because shubs wanted it

We will attempt to find a file matching your provided <input>, and otherwise
attempt to parse it as a URI. 
If protocol is missing, then we will assume from the port.
If the port is missing, then we will try both http:80 and https:443


you can optionally enable depth based searching if your input wordlist has directories added, e.g.
/api/foo
/api/bar
/v1/foo
/v2/bar
we will perform baseline checks at each directory. This will create more traffic, but less noisy results

usage: 
kr brute <input> <flags>
kr brute domain.com -w wordlist.txt
cat hosts | kr brute - -w wordlist.txt -e aspx,asmx,ashx,asp --dirsearch-compat
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
			scan.LoadTextWordlist(textWordlist, extensions, dirsearchCompatabilityMode),
			scan.LoadAssetnoteWordlist(assetnoteWordlist, extensions, dirsearchCompatabilityMode),
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
			log.Info().Err(http.ListenAndServe("localhost:6060", nil)).Msg("Started http profiler server")
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
	rootCmd.AddCommand(bruteCmd)

	bruteCmd.Flags().StringSliceVarP(&textWordlist, "wordlist", "w", textWordlist, "normal wordlist to use for scanning")
	bruteCmd.Flags().StringSliceVarP(&extensions, "extensions", "e", extensions, "extensions to append while scanning")
	bruteCmd.Flags().BoolVarP(&dirsearchCompatabilityMode, "dirsearch-compat", "D", dirsearchCompatabilityMode, "this will replace %EXT% with the extensions provided. backwards compat with dirsearch because shubs loves him some dirsearch")

	// bruteCmd.Flags().StringSliceVarP(&headers, "header", "H", []string{"x-forwarded-for: 127.0.0.1"}, "headers to add to requests")
	// Fix for improper header definition
	bruteCmd.Flags().StringArrayVarP(&headers, "header", "H", []string{"x-forwarded-for: 127.0.0.1"}, "headers to add to requests")

	bruteCmd.Flags().BoolVar(&disablePrecheck, "disable-precheck", false, "whether to skip host discovery")

	bruteCmd.Flags().IntVarP(&maxConnPerHost, "max-connection-per-host", "x", maxConnPerHost, "max connections to a single host")
	bruteCmd.Flags().IntVarP(&maxParallelHosts, "max-parallel-hosts", "j", maxParallelHosts, "max number of concurrent hosts to scan at once")
	bruteCmd.Flags().DurationVar(&delay, "delay", delay, "delay to place inbetween requests to a single host")
	bruteCmd.Flags().StringVar(&userAgent, "user-agent", userAgent, "user agent to use for requests")
	bruteCmd.Flags().DurationVarP(&timeout, "timeout", "t", timeout, "timeout to use on all requests")
	bruteCmd.Flags().IntVar(&maxRedirects, "max-redirects", maxRedirects, "maximum number of redirects to follow")
	bruteCmd.Flags().StringVar(&forceMethod, "force-method", forceMethod, "whether to ignore the methods specified in the ogl file and force this method")

	bruteCmd.Flags().IntSliceVar(&successStatusCodes, "success-status-codes", successStatusCodes,
		"which status codes whitelist as success. this is the default mode")
	bruteCmd.Flags().IntSliceVar(&failStatusCodes, "fail-status-codes", failStatusCodes,
		"which status codes blacklist as fail. if this is set, this will override success-status-codes")

	bruteCmd.Flags().StringSliceVar(&blacklistDomains, "blacklist-domain", blacklistDomains, "domains that are blacklisted for redirects. We will not follow redirects to these domains")
	bruteCmd.Flags().BoolVar(&wildcardDetection, "wildcard-detection", wildcardDetection, "can be set to false to disable wildcard redirect detection")
	bruteCmd.Flags().StringSliceVar(&lengthIgnoreRange, "ignore-length", lengthIgnoreRange, "a range of content length bytes to ignore. you can have multiple. e.g. 100-105 or 1234 or 123,34-53. This is inclusive on both ends")

	bruteCmd.Flags().BoolVar(&progressBar, "progress", progressBar, "a progress bar while scanning. by default enabled only on Stderr")
	bruteCmd.Flags().Int64Var(&quarantineThreshold, "quarantine-threshold", quarantineThreshold, "if the host return N consecutive hits, we quarantine the host as wildcard. Set to 0 to disable")

	bruteCmd.Flags().Int64VarP(&preflightDepth, "preflight-depth", "d", 0, "when performing preflight checks, what directory depth do we attempt to check. 0 means that only the docroot is checked")
	bruteCmd.Flags().StringVar(&profileName, "profile-name", profileName, "name for profile output file")

	bruteCmd.Flags().StringSliceVar(&filterAPIs, "filter-api", filterAPIs, "only scan apis matching this ksuid")
	bruteCmd.Flags().StringSliceVarP(&assetnoteWordlist, "assetnote-wordlist", "A", assetnoteWordlist, "use the wordlists from wordlist.assetnote.io. specify the type/name to use, e.g. apiroutes-210228. You can specify an additional maxlength to use only the first N values in the wordlist, e.g. apiroutes-210228;20000 will only use the first 20000 lines in that wordlist")
}
