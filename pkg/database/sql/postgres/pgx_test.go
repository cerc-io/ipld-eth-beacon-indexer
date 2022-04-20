package postgres

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/testhelpers"
)

var (
	pgConfig, _ = makeConfig(DefaultConfig)
	ctx         = context.Background()
)

func expectContainsSubstring(t *testing.T, full string, sub string) {
	if !strings.Contains(full, sub) {
		t.Fatalf("Expected \"%v\" to contain substring \"%v\"\n", full, sub)
	}
}

func TestPostgresPGX(t *testing.T) {
	t.Run("connects to the sql", func(t *testing.T) {
		dbPool, err := pgxpool.ConnectConfig(context.Background(), pgConfig)
		if err != nil {
			t.Fatalf("failed to connect to db with connection string: %s err: %v", pgConfig.ConnString(), err)
		}
		if dbPool == nil {
			t.Fatal("DB pool is nil")
		}
		dbPool.Close()
	})

	t.Run("serializes big.Int to db", func(t *testing.T) {
		// postgres driver doesn't support go big.Int type
		// various casts in golang uint64, int64, overflow for
		// transaction value (in wei) even though
		// postgres numeric can handle an arbitrary
		// sized int, so use string representation of big.Int
		// and cast on insert

		dbPool, err := pgxpool.ConnectConfig(context.Background(), pgConfig)
		if err != nil {
			t.Fatalf("failed to connect to db with connection string: %s err: %v", pgConfig.ConnString(), err)
		}
		defer dbPool.Close()

		bi := new(big.Int)
		bi.SetString("34940183920000000000", 10)
		testhelpers.ExpectEqual(t, bi.String(), "34940183920000000000")

		defer dbPool.Exec(ctx, `DROP TABLE IF EXISTS example`)
		_, err = dbPool.Exec(ctx, "CREATE TABLE example ( id INTEGER, data NUMERIC )")
		if err != nil {
			t.Fatal(err)
		}

		sqlStatement := `
			INSERT INTO example (id, data)
			VALUES (1, cast($1 AS NUMERIC))`
		_, err = dbPool.Exec(ctx, sqlStatement, bi.String())
		if err != nil {
			t.Fatal(err)
		}

		var data string
		err = dbPool.QueryRow(ctx, `SELECT cast(data AS TEXT) FROM example WHERE id = 1`).Scan(&data)
		if err != nil {
			t.Fatal(err)
		}

		testhelpers.ExpectEqual(t, data, bi.String())
		actual := new(big.Int)
		actual.SetString(data, 10)
		testhelpers.ExpectEqual(t, actual, bi)
	})

	t.Run("throws error when can't connect to the database", func(t *testing.T) {
		_, err := NewPostgresDB(Config{
			Driver: "PGX",
		})
		if err == nil {
			t.Fatal("Expected an error")
		}

		expectContainsSubstring(t, err.Error(), sql.DbConnectionFailedMsg)
	})
	t.Run("Connect to the database", func(t *testing.T) {
		driver, err := NewPostgresDB(DefaultConfig)
		defer driver.Close()

		if err != nil {
			t.Fatal("Error creating the postgres driver")
		}

	})
}
