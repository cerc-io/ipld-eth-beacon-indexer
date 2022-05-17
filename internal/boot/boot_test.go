package boot_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/boot"
)

var _ = Describe("Boot", func() {
	var (
		dbAddress            string = "localhost"
		dbPort               int    = 8076
		dbName               string = "vulcanize_testing"
		dbUsername           string = "vdbm"
		dbPassword           string = "password"
		dbDriver             string = "PGX"
		bcAddress            string = "localhost"
		bcPort               int    = 5052
		bcConnectionProtocol string = "http"
	)
	Describe("Booting the application", Label("integration"), func() {
		Context("When the DB and BC are both up and running, and we skip checking for a synced head", func() {
			It("Should connect successfully", func() {
				_, db, err := boot.BootApplicationWithRetry(context.Background(), dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol, true)
				defer db.Close()
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When the DB and BC are both up and running, and we check for a synced head", func() {
			It("Should not connect successfully", func() {
				_, db, err := boot.BootApplicationWithRetry(context.Background(), dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol, false)
				defer db.Close()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When the DB is running but not the BC", func() {
			It("Should not connect successfully", func() {
				_, _, err := boot.BootApplication(context.Background(), dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, "hi", 100, bcConnectionProtocol, true)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When the BC is running but not the DB", func() {
			It("Should not connect successfully", func() {
				_, _, err := boot.BootApplication(context.Background(), "hi", 10, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol, true)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When neither the BC or DB are running", func() {
			It("Should not connect successfully", func() {
				_, _, err := boot.BootApplication(context.Background(), "hi", 10, dbName, dbUsername, dbPassword, dbDriver, "hi", 100, bcConnectionProtocol, true)
				Expect(err).To(HaveOccurred())
			})
		})
	})

})
