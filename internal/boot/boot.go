package boot

import (
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql/postgres"
)

func setUpLightHouse() {

}

// This function will perform some boot operations.
// 1. Setup a logger
// 2. Connect to the database.
// 3. Connect to to the lighthouse client.
func BootApplication(dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string) (*postgres.DB, error) {
	log.Debug("Setting up DB connection")
	DB, err := SetupDb(dbHostname, dbPort, dbName, dbUsername, dbPassword, driverName)
	if err != nil {
		return nil, err
	}
	return DB, nil
}
