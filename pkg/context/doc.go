/*
Package context provides utilities wrapping the native go/context package
for catching and handling multiple interrupts.

The main use-case is to to attach an interrupt signal handler to the context.
This can be used from your CLI applications to ensure a graceful shutdown of the scanning
and to clean up any resources mid-flight

	import "github.com/assetnote/kiterunner/pkg/context"

	...

	if err := scan.ScanDomainOrFile(context.Context(), domain, opts...); err != nil {
		log.Fatal().Err(err).Msg("failed to scan domain")
	}
 */
package context
