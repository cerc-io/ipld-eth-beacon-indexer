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
test: | $(GINKGO)
	go vet ./...
	go fmt ./...
	$(GINKGO) -r

#.PHONY: integrationtest
#integrationtest: | $(GINKGO) $(GOOSE)
#	go vet ./...
#	go fmt ./...
#	$(GINKGO) -r test/ -v

.PHONY: build
build:
	go fmt ./...
	GO111MODULE=on go build

## Build docker image
.PHONY: docker-build
docker-build:
	docker build -t vulcanize/ipld-ethcl-indexer .