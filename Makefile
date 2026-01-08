VERSION ?= v1.4.1
K8S_VERSION ?= 1.3.0
RELEASE_TAG ?=
OUTPUT_DIR ?= generated-package
EXPERIMENTAL ?= false

.PHONY: generate clean build

generate: build
ifndef RELEASE_TAG
	$(error RELEASE_TAG is required. Usage: make generate RELEASE_TAG=v0.2.0)
endif
ifeq ($(EXPERIMENTAL),true)
	./bin/generate --version=$(VERSION) --k8s-version=$(K8S_VERSION) --release-tag=$(RELEASE_TAG) --output=$(OUTPUT_DIR) --experimental
else
	./bin/generate --version=$(VERSION) --k8s-version=$(K8S_VERSION) --release-tag=$(RELEASE_TAG) --output=$(OUTPUT_DIR)
endif

build:
	go build -o bin/generate ./cmd/generate

clean:
	rm -rf $(OUTPUT_DIR) bin/

.PHONY: test
test:
	go test ./...

.PHONY: examples
examples:
	cd examples/simple && pkl project resolve && pkl eval gateway.pkl -f yaml > gateway.yaml
	@echo "Generated examples/simple/gateway.yaml"
