BIN = $(GOPATH)/bin
BASE = $(GOPATH)/src/$(PACKAGE)
PKGS = go list ./... | grep -v "^vendor/"

# Tools
## Testing library
GINKGO = $(BIN)/ginkgo
$(BIN)/ginkgo:
	go install github.com/onsi/ginkgo/ginkgo


.PHONY: installtools
installtools: | $(GINKGO)
	echo "Installing tools"

.PHONY: test
test:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r

.PHONY: integration-test-ci
integration-test-ci:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r --label-filter integration \
	--procs=4 --compilers=4 \
	--flake-attempts=3 \
	--randomize-all --randomize-suites \
	--fail-on-pending --keep-going \
	--cover --coverprofile=cover.profile \
	--race --trace --json-report=report.json

.PHONY: integration-test-ci-no-race
integration-test-ci-no-race:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r --label-filter integration \
	--procs=4 --compilers=4 \
	--randomize-all --randomize-suites \
	--fail-on-pending --keep-going \
	--cover --coverprofile=cover.profile \
	--trace --json-report=report.json

.PHONY: integration-test-local
integration-test-local:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r --label-filter integration \
	--procs=4 --compilers=4 \
	--randomize-all --randomize-suites \
	--fail-on-pending --keep-going \
	--trace --race

.PHONY: integration-test-local-no-race
integration-test-local-no-race:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r --label-filter integration \
	--procs=4 --compilers=4 \
	--randomize-all --randomize-suites \
	--fail-on-pending --keep-going \
	--trace

.PHONY: unit-test-local
unit-test-local:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r --label-filter unit \
	--randomize-all --randomize-suites \
	--flake-attempts=3 \
	--fail-on-pending --keep-going \
	--trace

.PHONY: unit-test-ci
unit-test-ci:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r --label-filter unit \
	--randomize-all --randomize-suites \
	--flake-attempts=3 \
	--fail-on-pending --keep-going \
	--cover --coverprofile=cover.profile \
	--trace --json-report=report.json

.PHONY: system-test-ci
system-test-ci:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r --label-filter system \
	--randomize-all --randomize-suites \
	--fail-on-pending --keep-going \
	--cover --coverprofile=cover.profile \
	--trace --json-report=report.json

.PHONY: system-test-local
system-test-local:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r --label-filter system \
	--randomize-all --randomize-suites \
	--fail-on-pending --keep-going \
	--trace

.PHONY: build
build:
	go fmt ./...
	GO111MODULE=on go build

## Build docker image
.PHONY: docker-build
docker-build:
	docker build -t vulcanize/ipld-eth-beacon-indexer .