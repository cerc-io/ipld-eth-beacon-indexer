package beaconclient_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestBeaconClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BeaconClient Suite", Label("beacon-client"))
}
