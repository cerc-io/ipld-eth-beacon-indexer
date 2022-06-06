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
		bcType               string = "lighthouse"
		bcBootRetryInterval  int    = 1
		bcBootMaxRetry       int    = 5
		bcKgTableIncrement   int    = 10
		bcUniqueIdentifier   int    = 100
		bcCheckDb            bool   = false
	)
	Describe("Booting the application", Label("integration"), func() {
		Context("When the DB and BC are both up and running, we skip checking for a synced head, and we are processing head", func() {
			It("Should connect successfully", func() {
				_, db, err := boot.BootApplicationWithRetry(context.Background(), dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol, bcType, bcBootRetryInterval, bcBootMaxRetry, bcKgTableIncrement, "head", true, bcUniqueIdentifier, bcCheckDb)
				defer db.Close()
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When the DB and BC are both up and running, we skip checking for a synced head, and we are processing historic ", func() {
			It("Should connect successfully", func() {
				_, db, err := boot.BootApplicationWithRetry(context.Background(), dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol, bcType, bcBootRetryInterval, bcBootMaxRetry, bcKgTableIncrement, "historic", true, bcUniqueIdentifier, bcCheckDb)
				defer db.Close()
				Expect(err).ToNot(HaveOccurred())
			})
		})
		Context("When the DB and BC are both up and running, and we check for a synced head", func() {
			It("Should not connect successfully", func() {
				_, db, err := boot.BootApplicationWithRetry(context.Background(), dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol, bcType, bcBootRetryInterval, bcBootMaxRetry, bcKgTableIncrement, "head", false, bcUniqueIdentifier, bcCheckDb)
				defer db.Close()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When the DB and BC are both up and running, we skip checking for a synced head, but the unique identifier is 0", func() {
			It("Should not connect successfully", func() {
				_, db, err := boot.BootApplicationWithRetry(context.Background(), dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol, bcType, bcBootRetryInterval, bcBootMaxRetry, bcKgTableIncrement, "head", false, 0, bcCheckDb)
				defer db.Close()
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When the DB is running but not the BC", func() {
			It("Should not connect successfully", func() {
				_, _, err := boot.BootApplication(context.Background(), dbAddress, dbPort, dbName, dbUsername, dbPassword, dbDriver, "hi", 100, bcConnectionProtocol, bcKgTableIncrement, true, bcUniqueIdentifier, bcCheckDb)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When the BC is running but not the DB", func() {
			It("Should not connect successfully", func() {
				_, _, err := boot.BootApplication(context.Background(), "hi", 10, dbName, dbUsername, dbPassword, dbDriver, bcAddress, bcPort, bcConnectionProtocol, bcKgTableIncrement, true, bcUniqueIdentifier, bcCheckDb)
				Expect(err).To(HaveOccurred())
			})
		})
		Context("When neither the BC or DB are running", func() {
			It("Should not connect successfully", func() {
				_, _, err := boot.BootApplication(context.Background(), "hi", 10, dbName, dbUsername, dbPassword, dbDriver, "hi", 100, bcConnectionProtocol, bcKgTableIncrement, true, bcUniqueIdentifier, bcCheckDb)
				Expect(err).To(HaveOccurred())
			})
		})
	})

})
