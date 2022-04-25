package beaconclient_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	beaconclient "github.com/vulcanize/ipld-ethcl-indexer/pkg/beaconclient"
)

var _ = Describe("Healthcheck", func() {
	var (
		BC = beaconclient.BeaconClient{
			Address: "localhost",
			Port:    5052,
		}
		errBc = beaconclient.BeaconClient{
			Address: "blah",
			Port:    10,
		}
	)
	Describe("Connecting to the lighthouse client", Label("integration"), func() {
		Context("When the client is running", func() {
			It("We should connect successfully", func() {
				err := BC.CheckBeaconClient()
				Expect(err).To(BeNil())
			})
		})
		Context("When the client is running", func() {
			It("We should connect successfully", func() {
				err := errBc.CheckBeaconClient()
				Expect(err).ToNot(BeNil())
			})
		})
	})
})
