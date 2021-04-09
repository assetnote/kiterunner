package cmd

import (
	"github.com/assetnote/kiterunner/internal/wordlist"
	"github.com/assetnote/kiterunner/pkg/context"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/spf13/cobra"
)

var (
	noCache bool = false
)

// wordlistSaveCmd represents the wordlistCache command
var wordlistSaveCmd = &cobra.Command{
	Use:   "save [wordlists ...]",
	Short: "save the wordlists specified (full filename or alias)",
	Long: `save will download the wordlists specified to ~/.cache/kiterunner/wordlists

you can use the alias or the full filename listed in [kr wordlist list]
`,

	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := wordlist.Save(context.Context(), noCache, args...); err != nil {
			log.Fatal().Err(err).Msg("failed to list wordlists")
		}
	},
}

func init() {
	wordlistCmd.AddCommand(wordlistSaveCmd)
	wordlistSaveCmd.Flags().BoolVar(&noCache, "no-cache", noCache, "delete the local files matching the names and pull fresh files")
}
