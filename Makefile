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
	--randomize-all --randomize-suites \
	--fail-on-pending --keep-going \
	--cover --coverprofile=cover.profile \
	--race --trace --json-report=report.json --timeout=TIMEOUT

.PHONY: unit-test-ci
test:
	go vet ./...
	go fmt ./...
	$(GINKGO) -r --label-filter unit \
	--procs=4 --compilers=4 \
	--randomize-all --randomize-suites \
	--fail-on-pending --keep-going \
	--cover --coverprofile=cover.profile \
	--race --trace --json-report=report.json --timeout=TIMEOUT


.PHONY: build
build:
	go fmt ./...
	GO111MODULE=on go build

## Build docker image
.PHONY: docker-build
docker-build:
	docker build -t vulcanize/ipld-ethcl-indexer .