package cmd

import (
	"os"

	"github.com/assetnote/kiterunner/internal/kitebuilder"
	"github.com/assetnote/kiterunner/pkg/context"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/spf13/cobra"
)

// compileCmd represents the compile command
var compileCmd = &cobra.Command{
	Use:   "compile <input> <output>",
	Short: "compile an kitebuilder schema and write the data to the specified file",
	Long: `compile an kitebuilder schema and write the data to the specified file

passing - as filename will read from stdin. This will read all of stdin
before processing and will not compile the input as streaming input

-d Debug mode will attempt to compile the schema with error handling
-v=debug Debug verbosity will print out the errors for the schema`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		filename := args[0]
		output := args[1]
		if filename == "-" {
			if err := kitebuilder.Compile(context.Context(), os.Stdin, output, kitebuilder.Debug(debug)); err != nil {
				log.Fatal().Err(err).Msg("failed to read from stdin")
			}
		} else {
			if err := kitebuilder.CompileFile(context.Context(), filename, output, kitebuilder.Debug(debug)); err != nil {
				log.Fatal().Err(err).Msg("failed to read from stdin")
			}
		}
	},
}

func init() {
	kidebuilderCmd.AddCommand(compileCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// compileCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	 compileCmd.Flags().BoolVarP(&debug, "debug", "d", false, "debug the parsing")
}
