BINARY := mac-onboarding
DIST   := dist
MODULE := github.com/oleg-koval/mac-onboarding
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"

.PHONY: build test lint clean release

build:
	mkdir -p $(DIST)
	go build $(LDFLAGS) -o $(DIST)/$(BINARY) .

# Cross-compile both architectures (for release)
release:
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-amd64 .

test:
	go test ./... -count=1

lint:
	go vet ./...

clean:
	rm -rf $(DIST)

# Install binary to /usr/local/bin
install: build
	install -m 755 $(DIST)/$(BINARY) /usr/local/bin/$(BINARY)
