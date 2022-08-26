ARCH?=amd64
GOOS?=linux
OUT_DIR?=build
PACKAGE=github.com/circonus-labs/custom-metrics-circonus-adapter
PREFIX?=circonus
TAG = v0.1.9
PKG := $(shell find pkg/* -type f)

.PHONY: build docker push test clean

lint:
	golangci-lint run

release:
	goreleaser --rm-dist

build: build/adapter

build/adapter: adapter.go $(PKG)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -mod=vendor -a -o $(OUT_DIR)/$(ARCH)/adapter adapter.go

docker:
	docker build --pull -t ${PREFIX}/custom-metrics-circonus-adapter:$(TAG) .

push: docker
	docker push ${PREFIX}/custom-metrics-circonus-adapter:$(TAG)

test: $(PKG)
	CGO_ENABLED=0 go test -mod=vendor ./pkg/...

clean:
	rm -rf build

