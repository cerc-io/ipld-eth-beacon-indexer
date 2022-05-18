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
package postgres

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/loghelper"
)

var _ sql.Database = &DB{}

// NewPostgresDB returns a postgres.DB using the provided Config and driver type.
func NewPostgresDB(c Config) (*DB, error) {
	var driver *pgxDriver

	driver, err := createDriver(c)

	if err != nil {
		return nil, err
	}

	return &DB{driver}, nil
}

// A simple wrapper to create a DB object to use.
func SetupPostgresDb(dbHostname string, dbPort int, dbName string, dbUsername string, dbPassword string, driverName string) (sql.Database, error) {
	log.Debug("Resolving Driver Type")
	DbDriver, err := ResolveDriverType(driverName)
	if err != nil {
		log.WithFields(log.Fields{
			"err":                  err,
			"driver_name_provided": driverName,
		}).Error("Can't resolve driver type")
	}
	log.Info("Using Driver:", DbDriver)

	postgresConfig := Config{
		Hostname:     dbHostname,
		Port:         dbPort,
		DatabaseName: dbName,
		Username:     dbUsername,
		Password:     dbPassword,
		Driver:       DbDriver,
	}
	DB, err := NewPostgresDB(postgresConfig)

	if err != nil {
		loghelper.LogError(err).Error("Unable to connect to the DB")
		return nil, err
	}
	return DB, err
}

// Create a driver based on the config
func createDriver(c Config) (*pgxDriver, error) {
	switch c.Driver {
	case PGX:
		log.Debug("Creating New Driver")
		driver, err := newPGXDriver(context.Background(), c)
		if err != nil {
			return nil, fmt.Errorf("Error Creating Driver, err: %e", err)
		}
		log.Info("Successfully created a driver for PGX")
		return driver, nil
	default:
		log.Error("Couldnt find a driver to create for: ", c.Driver)
		return nil, fmt.Errorf("Can't find a driver to create")
	}

}

// DB implements sql.Database using a configured driver and Postgres statement syntax
type DB struct {
	sql.Driver
}
