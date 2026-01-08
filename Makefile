VERSION ?= v1.4.1
K8S_VERSION ?= 1.3.0
OUTPUT_DIR ?= generated-package

.PHONY: generate clean build

generate: build
	./bin/generate --version=$(VERSION) --k8s-version=$(K8S_VERSION) --output=$(OUTPUT_DIR)

build:
	go build -o bin/generate ./cmd/generate

clean:
	rm -rf $(OUTPUT_DIR) bin/

.PHONY: test
test:
	go test ./...
