package postgres

import (
	"context"
	"fmt"

	"github.com/vulcanize/ipld-ethcl-indexer/pkg/dbtools/sql"
)

var _ sql.Database = &DB{}

// TODO: Make NewPostgresDB accept a string and Config. IT should
// Create a driver of its own.
// This will make sure that if you want a driver, it conforms to the interface.

// NewPostgresDB returns a postgres.DB using the provided Config and driver type.
func NewPostgresDB(c Config, driverName string) (*DB, error) {
	var driver *pgxDriver

	driverType, err := ResolveDriverType(driverName)
	if err != nil {
		return nil, err
	}

	driver, err = createDriver(c, driverType)

	if err != nil {
		return nil, err
	}

	return &DB{driver}, nil
}

func createDriver(c Config, driverType DriverType) (*pgxDriver, error) {
	switch driverType {
	case PGX:
		driver, err := newPGXDriver(context.Background(), c)
		if err != nil {
			return nil, fmt.Errorf("Error Creating Driver, err: %e", err)
		}
		return driver, nil
	default:
		return nil, fmt.Errorf("Can't find a driver to create")
	}

}

// DB implements sql.Database using a configured driver and Postgres statement syntax
type DB struct {
	sql.Driver
}
