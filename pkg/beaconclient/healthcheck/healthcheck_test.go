package healthcheck_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient/healthcheck"
)

var _ = Describe("Healthcheck", func() {
	var (
		bcAddress string = "localhost"
		bcPort    int    = 5052
	)
	Describe("Connecting to the lighthouse client", Label("integration"), func() {
		Context("When the client is running", func() {
			It("We should connect successfully", func() {
				err := healthcheck.CheckBeaconClient(bcAddress, bcPort)
				Expect(err).To(BeNil())
			})
		})
		Context("When the client is running", func() {
			It("We should connect successfully", func() {
				err := healthcheck.CheckBeaconClient("blah-blah", 10)
				Expect(err).ToNot(BeNil())
			})
		})
	})
})
