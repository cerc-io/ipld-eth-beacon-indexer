package boot

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql/postgres"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

var (
	maxRetry                                 = 5  // Max times to try to connect to the DB or BC at boot.
	retryInterval                            = 30 // The time to wait between each try.
	DB            sql.Database               = &postgres.DB{}
	BC            *beaconclient.BeaconClient = &beaconclient.BeaconClient{}
)

// A simple wrapper to create a DB object to use.
func SetupPostgresDb(dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string) (sql.Database, error) {
	log.Debug("Resolving Driver Type")
	DbDriver, err := postgres.ResolveDriverType(driverName)
	if err != nil {
		log.WithFields(log.Fields{
			"err":                  err,
			"driver_name_provided": driverName,
		}).Error("Can't resolve driver type")
	}
	log.Info("Using Driver:", DbDriver)

	postgresConfig := postgres.Config{
		Hostname:     dbHostname,
		Port:         dbPort,
		DatabaseName: dbName,
		Username:     dbUsername,
		Password:     dbPassword,
		Driver:       DbDriver,
	}
	DB, err = postgres.NewPostgresDB(postgresConfig)

	if err != nil {
		loghelper.LogError(err).Error("Unable to connect to the DB")
		return nil, err
	}
	return DB, err

}

// This function will perform some boot operations. If any steps fail, the application will fail to start.
// Keep in mind that the DB connection can be lost later in the lifecycle of the application or
// it might not be able to connect to the beacon client.
//
// 1. Make sure the Beacon client is up.
//
// 2. Connect to the database.
//
func BootApplication(dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string, bcAddress string, bcPort int) (sql.Database, error) {
	log.Info("Booting the Application")

	log.Debug("Creating the Beacon Client")
	BC.Address = bcAddress
	BC.Port = bcPort
	log.Debug("Checking Beacon Client")

	err := BC.CheckBeaconClient()
	if err != nil {
		return nil, err
	}

	log.Debug("Setting up DB connection")
	DB, err := SetupPostgresDb(dbHostname, dbPort, dbName, dbUsername, dbPassword, driverName)
	if err != nil {
		return nil, err
	}
	return DB, nil
}

// Add retry logic to ensure that we are give the Beacon Client and the DB time to start.
func BootApplicationWithRetry(dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string, bcAddress string, bcPort int) (sql.Database, error) {
	var err error
	for i := 0; i < maxRetry; i++ {
		DB, err = BootApplication(dbHostname, dbPort, dbName, dbUsername, dbPassword, driverName, bcAddress, bcPort)
		if err != nil {
			log.WithFields(log.Fields{
				"retryNumber": i,
			}).Warn("Unable to boot application. Going to try again")
			time.Sleep(time.Duration(retryInterval) * time.Second)
			continue
		}
	}
	return DB, err
}
