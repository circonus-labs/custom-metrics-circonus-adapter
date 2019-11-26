ARCH?=amd64
GOOS?=linux
OUT_DIR?=build
PACKAGE=github.com/rileyberton/custom-metrics-circonus-adapter
PREFIX?=rileyberton
TAG = v0.0.10
PKG := $(shell find pkg/* -type f)

.PHONY: build docker push test clean

build: build/adapter

build/adapter: adapter.go $(PKG)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(ARCH) go build -a -o $(OUT_DIR)/$(ARCH)/adapter adapter.go

docker:
	docker build --pull -t ${PREFIX}/custom-metrics-circonus-adapter:$(TAG) .

push: docker
	docker push ${PREFIX}/custom-metrics-circonus-adapter:$(TAG)

test: $(PKG)
	CGO_ENABLED=0 go test ./pkg/...

clean:
	rm -rf build

