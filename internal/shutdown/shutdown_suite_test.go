package shutdown_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestShutdown(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shutdown Suite")
}
