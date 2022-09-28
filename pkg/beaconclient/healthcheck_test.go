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
package beaconclient_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	beaconclient "github.com/vulcanize/ipld-eth-beacon-indexer/pkg/beaconclient"
)

var _ = Describe("Healthcheck", func() {
	var (
		Bc    *beaconclient.BeaconClient
		errBc *beaconclient.BeaconClient
	)

	BeforeEach(func() {
		var err error
		Bc, err = beaconclient.CreateBeaconClient(context.Background(), "http", "localhost", 5052, 10, bcUniqueIdentifier, false, true, true)
		Expect(err).ToNot(HaveOccurred())
		errBc, err = beaconclient.CreateBeaconClient(context.Background(), "http", "blah-blah", 1010, 10, bcUniqueIdentifier, false, true, true)
		Expect(err).ToNot(HaveOccurred())
	})
	Describe("Connecting to the lighthouse client", Label("integration"), func() {
		Context("When the client is running", func() {
			It("We should connect successfully", func() {
				err := Bc.CheckBeaconClient()
				Expect(err).To(BeNil())
			})
		})
		Context("When the client is not running", func() {
			It("We not should connect successfully", func() {
				err := errBc.CheckBeaconClient()
				Expect(err).ToNot(BeNil())
			})
		})
	})
})
