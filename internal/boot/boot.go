package boot

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql/postgres"
)

var (
	maxRetry                                 = 5  // Max times to try to connect to the DB or BC at boot.
	retryInterval                            = 30 // The time to wait between each try.
	DB            sql.Database               = &postgres.DB{}
	BC            *beaconclient.BeaconClient = &beaconclient.BeaconClient{}
)

// This function will perform some boot operations. If any steps fail, the application will fail to start.
// Keep in mind that the DB connection can be lost later in the lifecycle of the application or
// it might not be able to connect to the beacon client.
//
// 1. Make sure the Beacon client is up.
//
// 2. Connect to the database.
//
func BootApplication(ctx context.Context, dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string, bcAddress string, bcPort int, bcConnectionProtocol string) (*beaconclient.BeaconClient, sql.Database, error) {
	log.Info("Booting the Application")

	log.Debug("Creating the Beacon Client")
	BC = beaconclient.CreateBeaconClient(ctx, bcConnectionProtocol, bcAddress, bcPort)

	log.Debug("Checking Beacon Client")
	err := BC.CheckBeaconClient()
	if err != nil {
		return nil, nil, err
	}

	log.Debug("Setting up DB connection")
	DB, err := postgres.SetupPostgresDb(dbHostname, dbPort, dbName, dbUsername, dbPassword, driverName)
	if err != nil {
		return nil, nil, err
	}

	BC.Db = DB
	return BC, DB, nil
}

// Add retry logic to ensure that we are give the Beacon Client and the DB time to start.
func BootApplicationWithRetry(ctx context.Context, dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string, bcAddress string, bcPort int, bcConnectionProtocol string) (*beaconclient.BeaconClient, sql.Database, error) {
	var err error
	for i := 0; i < maxRetry; i++ {
		BC, DB, err = BootApplication(ctx, dbHostname, dbPort, dbName, dbUsername, dbPassword, driverName, bcAddress, bcPort, bcConnectionProtocol)
		if err != nil {
			log.WithFields(log.Fields{
				"retryNumber": i,
			}).Warn("Unable to boot application. Going to try again")
			time.Sleep(time.Duration(retryInterval) * time.Second)
			continue
		}
		break
	}
	return BC, DB, err
}
