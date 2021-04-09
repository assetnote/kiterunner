package cmd

import (
	"github.com/spf13/cobra"
)

// kidebuilderCmd represents the kitebuilder command
var wordlistCmd = &cobra.Command{
	Use:   "wordlist",
	Short: "look at your cached wordlists and remote wordlists",
	Long: `used to help manage your wordlists in your .cache/kiterunner`,
}

func init() {
	rootCmd.AddCommand(wordlistCmd)
}
