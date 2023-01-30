package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var shortVersion bool

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Long:  `Print the version.`,
	Run: func(cmd *cobra.Command, args []string) {
		err := printVersion()
		cobra.CheckErr(err)
	},
}

func init() {
	versionCmd.Flags().BoolVarP(&shortVersion, "short", "s", false, "Show only the version string")

	rootCmd.AddCommand(versionCmd)
}

// Version will hold the actual build version from main for precompiled binaries.
var Version = ""

func printVersion() error {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return fmt.Errorf("could not read BuildInfo")
	}
	version := Version
	if version == "" {
		version = buildInfo.Main.Version
	}
	if version == "" {
		version = "development"
	}
	if shortVersion {
		fmt.Println(version)
	} else {
		fmt.Printf("Version: %v\n", version)
		fmt.Printf("Configuration: %v\n", viper.ConfigFileUsed())
	}
	return nil
}
