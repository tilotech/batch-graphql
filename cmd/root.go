package cmd

import (
	"context"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/tilotech/batch-graphql/batch"
)

var cfgFile string
var config batch.Config

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "batch-graphql",
	Short: "Run a batch of GraphQL queries or mutations",
	Long: `batch-graphql is a CLI tool for running high volumes of queries or
mutations on a GraphQL API with varying data.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := batch.Run(context.Background(), config)
		cobra.CheckErr(err)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.batch-graphql.json)")

	rootCmd.Flags().StringVarP(&config.URL, "url", "u", "", "URL of the GraphQL service")
	_ = viper.BindPFlag("url", rootCmd.Flags().Lookup("url"))
	_ = rootCmd.MarkFlagRequired("url")

	rootCmd.Flags().IntVarP(&config.Connections, "connections", "c", 10, "number of maximum of open connections and parallel requests")
	_ = viper.BindPFlag("connections", rootCmd.Flags().Lookup("connections"))

	rootCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "verbose output")
	_ = viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))

	rootCmd.Flags().StringSliceVar(&config.Headers, "header", []string{}, "additional headers to include in the request")
	_ = viper.BindPFlag("headers", rootCmd.Flags().Lookup("header"))

	rootCmd.Flags().StringVarP(&config.BearerToken, "token", "t", "", "bearer token (if authorization is needed, conflicts with all oauth options)")
	_ = viper.BindPFlag("token", rootCmd.Flags().Lookup("token"))

	rootCmd.Flags().StringVar(&config.OAuth.URL, "oauth.url", "", "URL of OAuth 2.0 service (if authorization is needed)")
	_ = viper.BindPFlag("oauth.url", rootCmd.Flags().Lookup("oauth.url"))

	rootCmd.Flags().StringVar(&config.OAuth.ClientID, "oauth.clientid", "", "client ID for OAuth 2.0 client credentials flow (if authorization is needed)")
	_ = viper.BindPFlag("oauth.clientid", rootCmd.Flags().Lookup("oauth.clientid"))

	rootCmd.Flags().StringVar(&config.OAuth.ClientSecret, "oauth.clientsecret", "", "client secret for OAuth 2.0 client credentials flow (if authorization is needed)")
	_ = viper.BindPFlag("oauth.clientsecret", rootCmd.Flags().Lookup("oauth.clientsecret"))

	rootCmd.Flags().StringVar(&config.OAuth.Scope, "oauth.scope", "", "requested scope for OAuth 2.0 client credentials flow (if authorization is needed)")
	_ = viper.BindPFlag("oauth.scope", rootCmd.Flags().Lookup("oauth.scope"))

	rootCmd.MarkFlagsMutuallyExclusive("token", "oauth.url")
	rootCmd.MarkFlagsRequiredTogether("oauth.url", "oauth.clientid", "oauth.clientsecret", "oauth.scope")

	rootCmd.Flags().StringVarP(&config.QueryFile, "query", "q", "", "file that contains the GraphQL query")
	_ = viper.BindPFlag("query", rootCmd.Flags().Lookup("query"))
	_ = rootCmd.MarkFlagRequired("query")

	rootCmd.Flags().StringVarP(&config.InputFile, "input", "i", "", "input file that contains the variables to send (default stdin)")
	_ = viper.BindPFlag("input", rootCmd.Flags().Lookup("input"))

	rootCmd.Flags().StringVarP(&config.OutputFile, "output", "o", "", "output file into which to write the responses (default stdout)")
	_ = viper.BindPFlag("output", rootCmd.Flags().Lookup("output"))

	rootCmd.Flags().StringVarP(&config.ErrorFile, "error", "e", "", "output file into which to write error responses (default stderr)")
	_ = viper.BindPFlag("error", rootCmd.Flags().Lookup("error"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetEnvPrefix("batch_graphql")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".batch-graphql" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("json")
		viper.SetConfigName(".batch-graphql")
	}

	// If a config file is found, read it in.
	err := viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			cobra.CheckErr(err)
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// index: name of cobra flag, value: name of viper flag
	flagMapping := map[string]string{
		"header": "headers",
	}

	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		name, ok := flagMapping[f.Name]
		if !ok {
			name = f.Name
		}
		if viper.IsSet(name) {
			if vs, ok := f.Value.(pflag.SliceValue); ok {
				err = vs.Replace(viper.GetStringSlice(name))
			} else {
				// don't set value directly as this would not mark required flags as provided
				err = rootCmd.Flags().Set(f.Name, viper.GetString(name))
			}
			cobra.CheckErr(err)
		}
	})
}
