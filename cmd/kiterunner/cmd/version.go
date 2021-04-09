package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// These global variables are injected at build time to provide the
// version command
var (
	Version = "v0.0.0"
	Commit  = "commit"
	Date    = "today"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "version of the binary you're running",
	Long: `this shows you the version of the binary that is running`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s - %s\n", Version, Commit)
		fmt.Printf("Built on %s\n", Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
