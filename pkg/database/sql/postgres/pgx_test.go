package postgres_test

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/database/sql/postgres"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/testhelpers"
)

var _ = Describe("Pgx", func() {

	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Connecting to the DB", Label("integration"), func() {
		Context("But connection is unsucessful", func() {
			It("throws error when can't connect to the database", func() {
				_, err := postgres.NewPostgresDB(postgres.Config{
					Driver: "PGX",
				})
				Expect(err).NotTo(HaveOccurred())

				present, err := doesContainsSubstring(err.Error(), sql.DbConnectionFailedMsg)
				Expect(present).To(BeTrue())
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("The connection is successful", func() {
			It("Should create a DB object", func() {
				db, err := postgres.NewPostgresDB(postgres.DefaultConfig)
				Expect(err).To(BeNil())
				defer db.Close()
			})
		})
	})
	Describe("Write to the DB", Label("integration"), func() {
		Context("Serialize big.Int to DB", func() {
			It("Should serialize successfully", func() {
				dbPool, err := postgres.NewPostgresDB(postgres.DefaultConfig)
				Expect(err).To(BeNil())
				defer dbPool.Close()

				bi := new(big.Int)
				bi.SetString("34940183920000000000", 10)
				isEqual, err := testhelpers.IsEqual(bi.String(), "34940183920000000000")
				Expect(err).To(BeNil())
				Expect(isEqual).To(BeTrue())

				defer func() {
					_, err := dbPool.Exec(ctx, `DROP TABLE IF EXISTS example`)
					Expect(err).To(BeNil())
				}()
				_, err = dbPool.Exec(ctx, "CREATE TABLE example ( id INTEGER, data NUMERIC )")
				Expect(err).To(BeNil())

				sqlStatement := `
			INSERT INTO example (id, data)
			VALUES (1, cast($1 AS NUMERIC))`
				_, err = dbPool.Exec(ctx, sqlStatement, bi.String())
				Expect(err).To(BeNil())

				var data string
				err = dbPool.QueryRow(ctx, `SELECT cast(data AS TEXT) FROM example WHERE id = 1`).Scan(&data)
				Expect(err).To(BeNil())

				isEqual, err = testhelpers.IsEqual(data, bi.String())
				Expect(isEqual).To(BeTrue())
				Expect(err).To(BeNil())
				actual := new(big.Int)
				actual.SetString(data, 10)

				isEqual, err = testhelpers.IsEqual(actual, bi)
				Expect(isEqual).To(BeTrue())
				Expect(err).To(BeNil())
			})
		})
	})
})

func doesContainsSubstring(full string, sub string) (bool, error) {
	if !strings.Contains(full, sub) {
		return false, fmt.Errorf("Expected \"%v\" to contain substring \"%v\"\n", full, sub)
	}
	return true, nil
}
