// This file will allow users to setup a new DB based on the user provided inputs.

package boot

import (
	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql/postgres"
)

func SetupDb(dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string) (*postgres.DB, error) {
	log.Debug("Resolving Driver Type")
	DbDriver, err := postgres.ResolveDriverType(driverName)
	if err != nil {
		log.Fatal("Can't Connect to DB")
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
	DB, err := postgres.NewPostgresDB(postgresConfig)
	return DB, err

}
