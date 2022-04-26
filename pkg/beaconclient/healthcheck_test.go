package beaconclient_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	beaconclient "github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
)

var _ = Describe("Healthcheck", func() {
	var (
		BC    = beaconclient.CreateBeaconClient(context.Background(), "localhost", 5052)
		errBc = beaconclient.CreateBeaconClient(context.Background(), "blah-blah", 1010)
	)
	Describe("Connecting to the lighthouse client", Label("integration"), func() {
		Context("When the client is running", func() {
			It("We should connect successfully", func() {
				err := BC.CheckBeaconClient()
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
