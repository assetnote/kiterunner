package cmd

import (
	"github.com/assetnote/kiterunner/internal/kitebuilder"
	"github.com/assetnote/kiterunner/pkg/context"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/spf13/cobra"
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert <input> <output>",
	Short: "convert an input file format into the specified output file format",
	Long: `convert an input file format into the specified output file format

this will determine the conversion based on the extensions of the input and the output
we support the following filetypes: txt, json, kite
You can convert any of the following into the corresponding types 

-d Debug mode will attempt to convert the schema with error handling
-v=debug Debug verbosity will print out the errors for the schema`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		output := args[1]

		if err := kitebuilder.ConvertFiles(context.Context(), input, output); err != nil {
			log.Fatal().Err(err).Msg("failed to convert files")
		}
		log.Info().Msg("conversion complete")
	},
}

func init() {
	kidebuilderCmd.AddCommand(convertCmd)
	convertCmd.Flags().BoolVarP(&debug, "debug", "d", false, "debug the parsing")
}
