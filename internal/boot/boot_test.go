package boot_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vulcanize/ipld-ethcl-indexer/internal/boot"
)

var _ = Describe("Boot", func() {
	var (
		dbAddress  string = "localhost"
		dbPort     int    = 8077
		dbName     string = "vulcanize_testing"
		dbUsername string = "vdbm"
		dbPassword string = "password"
		dbDriver   string = "PGX"
		bcAddress  string = "localhost"
		bcPort     int    = 5052
	)
	Describe("Booting the application", Label("integration"), func() {
		Context("When the DB and BC are both up and running", func() {
			It("Should connect successfully", func() {
				db, err := boot.BootApplicationWithRetry(dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort)
				defer db.Close()
				Expect(err).To(BeNil())
			})
		})
		Context("When the DB is running but not the BC", func() {
			It("Should not connect successfully", func() {
				_, err := boot.BootApplication(dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, "hi", 100)
				Expect(err).ToNot(BeNil())
			})
		})
		Context("When the BC is running but not the DB", func() {
			It("Should not connect successfully", func() {
				_, err := boot.BootApplication("hi", 10, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort)
				Expect(err).ToNot(BeNil())
			})
		})
		Context("When neither the BC or DB are running", func() {
			It("Should not connect successfully", func() {
				_, err := boot.BootApplication("hi", 10, dbName, dbUsername, dbPassword, dbDriver, "hi", 100)
				Expect(err).ToNot(BeNil())
			})
		})
	})

})
