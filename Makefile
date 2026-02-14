# aria2bango Makefile

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=aria2bango
BINARY_UNIX=$(BINARY_NAME)_unix

# Directories
BIN_DIR=./bin
CMD_DIR=./cmd/aria2bango

.PHONY: all build clean test install uninstall run deps

all: deps build

deps:
	$(GOMOD) download
	$(GOMOD) tidy

build:
	mkdir -p $(BIN_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) $(CMD_DIR)

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BIN_DIR)/$(BINARY_UNIX) $(CMD_DIR)

clean:
	$(GOCLEAN)
	rm -rf $(BIN_DIR)

test:
	$(GOTEST) -v ./...

install: build
	install -m 755 $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/
	install -d -m 755 /etc/aria2bango
	install -m 644 configs/config.yaml /etc/aria2bango/config.yaml
	install -d -m 755 /var/log/aria2bango
	install -m 644 systemd/aria2bango.service /etc/systemd/system/
	systemctl daemon-reload

uninstall:
	rm -f /usr/local/bin/$(BINARY_NAME)
	rm -rf /etc/aria2bango
	rm -f /etc/systemd/system/aria2bango.service
	systemctl daemon-reload

run: build
	$(BIN_DIR)/$(BINARY_NAME) -config configs/config.yaml

# Development targets
fmt:
	gofmt -w .

lint:
	golangci-lint run

# Docker support
docker-build:
	docker build -t aria2bango:$(VERSION) .

docker-run:
	docker run --rm --cap-add=NET_ADMIN --network=host aria2bango:$(VERSION)