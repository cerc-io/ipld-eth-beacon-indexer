/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
)

// captureCmd represents the capture command
var captureCmd = &cobra.Command{
	Use:   "capture",
	Short: "Capture the SignedBeaconBlocks and BeaconStates from the Beacon Chain",
	Long: `Capture SignedBeaconBlocks and BeaconStates from the Beacon Chain.
	These blocks and states will be captured in
	Postgres. They require a lighthouse client to be connected. You can run this to
	capture blocks and states at head or historic blocks.`,
}

func init() {
	rootCmd.AddCommand(captureCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// captureCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// captureCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
