/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// headCmd represents the head command
var headCmd = &cobra.Command{
	Use:   "head",
	Short: "Capture only the blocks and state at head.",
	Long:  `Capture only the blocks and state at head.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("head called")
	},
}

func init() {
	captureCmd.AddCommand(headCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// headCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// headCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
