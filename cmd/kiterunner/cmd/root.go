package cmd

import (
	"fmt"
	"os"

	"github.com/assetnote/kiterunner/internal/art"
	"github.com/assetnote/kiterunner/pkg/log"
	"github.com/spf13/cobra"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)


// These global variables can be configured with the corresponding lowercase flag
var (
	Verbose string // Verbose defines the logging level, either trace, debug, info, error, fatal
	Output  string // Output defines the output format, either pretty, text, json
	Quiet   bool // Quiet will hide the beautiful ascii art upon startup

	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kite",
	Short: "kite scan one or mulitple hosts",
	Long: `kite is a context based webscanner that uses common
api paths for content discovery of an applications api paths`,
	// Run: func(cmd *cobra.Command, args []string) {},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initLogging)
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.kiterunner.yaml)")

	rootCmd.PersistentFlags().StringVarP(&Verbose, "verbose", "v", "info", "level of logging verbosity. can be error,info,debug,trace")
	rootCmd.PersistentFlags().StringVarP(&Output, "output", "o", "pretty", "output format. can be json,text,pretty")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "quiet mode. will mute unecessarry pretty text")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("quiet", rootCmd.PersistentFlags().Lookup("quiet"))
}

func initLogging() {
	log.SetFormat(viper.GetString("output"))

	level := viper.GetString("verbose")
	if level != "" {
		if err := log.SetLevelString(level); err != nil {
			log.Fatal().Err(err).Msg("failed to initialize logging")
		}
	}
	log.Debug().Str("level", level).Str("format", viper.GetString("output")).Msg("custom log settings")

	if Output == "pretty" && !viper.GetBool("quiet") {
		art.WriteArtBytes(os.Stderr)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".kiterunner" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".kiterunner")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
