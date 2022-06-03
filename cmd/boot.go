// VulcanizeDB
// Copyright © 2022 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"context"
	"os"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/boot"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/shutdown"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

// bootCmd represents the boot command
var bootCmd = &cobra.Command{
	Use:   "boot",
	Short: "Run the boot command then exit",
	Long:  `Run the application to boot and exit. Primarily used for testing.`,
	Run: func(cmd *cobra.Command, args []string) {
		bootApp()
	},
}

func bootApp() {

	// Boot the application
	log.Info("Starting the application in boot mode.")
	ctx := context.Background()

	Bc, Db, err := boot.BootApplicationWithRetry(ctx, viper.GetString("db.address"), viper.GetInt("db.port"), viper.GetString("db.name"), viper.GetString("db.username"), viper.GetString("db.password"), viper.GetString("db.driver"),
		viper.GetString("bc.address"), viper.GetInt("bc.port"), viper.GetString("bc.connectionProtocol"), viper.GetString("bc.type"), viper.GetInt("bc.bootRetryInterval"), viper.GetInt("bc.bootMaxRetry"),
		viper.GetInt("kg.increment"), "boot", viper.GetBool("t.skipSync"), viper.GetInt("bc.uniqueNodeIdentifier"))
	if err != nil {
		StopApplicationPreBoot(err, Db)
	}

	log.Info("Boot complete, we are going to shutdown.")

	notifierCh := make(chan os.Signal, 1)

	go func() {
		notifierCh <- syscall.SIGTERM
	}()

	err = shutdown.ShutdownBoot(ctx, notifierCh, maxWaitSecondsShutdown, Db, Bc)
	if err != nil {
		loghelper.LogError(err).Error("Ungracefully Shutdown ipld-ethcl-indexer!")
	} else {
		log.Info("Gracefully shutdown ipld-ethcl-indexer")
	}
}

func init() {
	captureCmd.AddCommand(bootCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// bootCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// bootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
