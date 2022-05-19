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
package boot

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql/postgres"
)

var (
	DB sql.Database               = &postgres.DB{}
	BC *beaconclient.BeaconClient = &beaconclient.BeaconClient{}
)

// This function will perform some boot operations. If any steps fail, the application will fail to start.
// Keep in mind that the DB connection can be lost later in the lifecycle of the application or
// it might not be able to connect to the beacon client.
//
// 1. Make sure the Beacon client is up.
//
// 2. Connect to the database.
//
// 3. Make sure the node is synced, unless disregardSync is true.
func BootApplication(ctx context.Context, dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string, bcAddress string, bcPort int, bcConnectionProtocol string, disregardSync bool) (*beaconclient.BeaconClient, sql.Database, error) {
	log.Info("Booting the Application")

	log.Debug("Creating the Beacon Client")
	BC = beaconclient.CreateBeaconClient(ctx, bcConnectionProtocol, bcAddress, bcPort)

	log.Debug("Checking Beacon Client")
	err := BC.CheckBeaconClient()
	if err != nil {
		return nil, nil, err
	}

	log.Debug("Setting up DB connection")
	DB, err = postgres.SetupPostgresDb(dbHostname, dbPort, dbName, dbUsername, dbPassword, driverName)
	if err != nil {
		return nil, nil, err
	}

	BC.Db = DB

	var status bool
	if !disregardSync {
		status, err = BC.CheckHeadSync()
		if err != nil {
			log.Error("Unable to get the nodes sync status")
			return BC, DB, err
		}
		if status {
			log.Error("The node is still syncing..")
			err = fmt.Errorf("The node is still syncing.")
			return BC, DB, err
		}
	} else {
		log.Warn("We are not checking to see if the node has synced to head.")
	}
	return BC, DB, nil
}

// Add retry logic to ensure that we are give the Beacon Client and the DB time to start.
func BootApplicationWithRetry(ctx context.Context, dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string,
	bcAddress string, bcPort int, bcConnectionProtocol string, bcType string, bcRetryInterval int, bcMaxRetry int, startUpMode string, disregardSync bool) (*beaconclient.BeaconClient, sql.Database, error) {
	var err error
	for i := 0; i < bcMaxRetry; i++ {
		BC, DB, err = BootApplication(ctx, dbHostname, dbPort, dbName, dbUsername, dbPassword, driverName, bcAddress, bcPort, bcConnectionProtocol, disregardSync)
		if err != nil {
			log.WithFields(log.Fields{
				"retryNumber": i,
				"err":         err,
			}).Warn("Unable to boot application. Going to try again")
			time.Sleep(time.Duration(bcRetryInterval) * time.Second)
			continue
		}
		break
	}

	switch strings.ToLower(startUpMode) {
	case "head":
		log.Debug("No further actions needed to boot the application at this phase.")
	case "historic":
		log.Debug("Performing additional boot steps for historical processing")
		headSlot, err := BC.GetLatestSlotInBeaconServer(bcType)
		if err != nil {
			return BC, DB, err
		}
		BC.UpdateLatestSlotInBeaconServer(int64(headSlot))
		// Add another switch case for bcType if its ever needed.
	}
	return BC, DB, err
}
