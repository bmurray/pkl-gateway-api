VERSION ?= v1.4.1
K8S_VERSION ?= 1.3.0
OUTPUT_DIR ?= generated-package
EXPERIMENTAL ?= false

.PHONY: generate clean build

generate: build
ifeq ($(EXPERIMENTAL),true)
	./bin/generate --version=$(VERSION) --k8s-version=$(K8S_VERSION) --output=$(OUTPUT_DIR) --experimental
else
	./bin/generate --version=$(VERSION) --k8s-version=$(K8S_VERSION) --output=$(OUTPUT_DIR)
endif

build:
	go build -o bin/generate ./cmd/generate

clean:
	rm -rf $(OUTPUT_DIR) bin/

.PHONY: test
test:
	go test ./...
