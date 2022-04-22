package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v "github.com/vulcanize/ipld-ethcl-indexer/pkg/version"
)

var (
	Major = 0  // Major version component of the current release
	Minor = 0  // Minor version component of the current release
	Patch = 0  // Patch version component of the current release
	Meta  = "" // Version metadata to append to the version string
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of ipld-eth-server",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		version := v.Version{
			Major: Major,
			Minor: Minor,
			Patch: Patch,
			Meta:  Meta,
		}
		log.Infof("ipld-ethcl-indexer version: %s", version.GetVersionWithMeta())
		fmt.Println(version.GetVersionWithMeta())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
