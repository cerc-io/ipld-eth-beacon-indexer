package postgres

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
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
