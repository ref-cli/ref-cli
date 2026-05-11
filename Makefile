VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS    := -ldflags "-X main.version=$(VERSION)"
PLATFORMS  := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
BIN        := bin/ref
DIST       := dist

.PHONY: build install build-all release test lint bundle-examples clean

build:
	go build $(LDFLAGS) -o $(BIN) ./cmd/ref

install:
	go install $(LDFLAGS) ./cmd/ref

build-all:
	$(foreach p,$(PLATFORMS), \
	  GOOS=$(word 1,$(subst /, ,$(p))) \
	  GOARCH=$(word 2,$(subst /, ,$(p))) \
	  go build $(LDFLAGS) -o $(DIST)/ref_$(subst /,_,$(p))/ref ./cmd/ref;)

release: build-all
	$(foreach p,$(PLATFORMS), \
	  cp LICENSE README.md $(DIST)/ref_$(subst /,_,$(p))/; \
	  tar -czf $(DIST)/ref_$(subst /,_,$(p)).tar.gz \
	    -C $(DIST)/ref_$(subst /,_,$(p)) ref LICENSE README.md;)

test:
	go test ./...

lint:
	golangci-lint run

bundle-examples:
	scripts/bundle-examples.sh

clean:
	rm -rf $(BIN) $(DIST)
