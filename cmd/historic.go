/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// historicCmd represents the historic command
var historicCmd = &cobra.Command{
	Use:   "historic",
	Short: "Capture the historic blocks and states.",
	Long:  `Capture the historic blocks and states.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("historic called")
	},
}

func init() {
	captureCmd.AddCommand(historicCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// historicCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// historicCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
