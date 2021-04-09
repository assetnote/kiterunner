package cmd

import (
	"github.com/spf13/cobra"
)

// kidebuilderCmd represents the kitebuilder command
var kidebuilderCmd = &cobra.Command{
	Use:   "kb",
	Short: "manipulate the kitebuilder schema",
	Long: `manipuate the kitebuilder schema in various ways`,
}

func init() {
	rootCmd.AddCommand(kidebuilderCmd)
}
