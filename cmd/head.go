/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/boot"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/shutdown"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

var (
	kgTableIncrement int
)

// headCmd represents the head command
var headCmd = &cobra.Command{
	Use:   "head",
	Short: "Capture only the blocks and state at head.",
	Long:  `Capture only the blocks and state at head.`,
	Run: func(cmd *cobra.Command, args []string) {
		startHeadTracking()
	},
}

// Start the application to track at head.
func startHeadTracking() {
	// Boot the application
	log.Info("Starting the application in head tracking mode.")
	ctx := context.Background()

	BC, DB, err := boot.BootApplicationWithRetry(ctx, dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol)
	if err != nil {
		loghelper.LogError(err).Error("Unable to Start application")
	}

	// Capture head blocks
	go BC.CaptureHead(kgTableIncrement)

	// Shutdown when the time is right.
	err = shutdown.ShutdownServices(ctx, notifierCh, maxWaitSecondsShutdown, DB, BC)
	if err != nil {
		loghelper.LogError(err).Error("Ungracefully Shutdown ipld-ethcl-indexer!")
	} else {
		log.Info("Gracefully shutdown ipld-ethcl-indexer")
	}

}

func init() {
	captureCmd.AddCommand(headCmd)

	// Known Gaps specific
	captureCmd.PersistentFlags().IntVarP(&kgTableIncrement, "kg.increment", "", 10000, "The max slots within a single entry to the known_gaps table.")
	err := viper.BindPFlag("kg.increment", captureCmd.PersistentFlags().Lookup("kg.increment"))
	exitErr(err)
}
