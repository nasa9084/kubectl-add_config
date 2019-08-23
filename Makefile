GO ?= GO111MODULE=on go
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.GoVersion=$(GO_VERSION) -X main.Revision=$(GIT_HASH)"
GO_VERSION = $(shell go version | cut -d" " -f3)
BIN_DIR = ./bin
DIST_DIR = ./dist
TAGGED_VERSION = $(shell git describe --abbrev=0 --tags 2> /dev/null)
VERSION ?= $(if $(TAGGED_VERSION),$(TAGGED_VERSION),v0.0.0)
GIT_HASH = $(shell git rev-parse --short HEAD)
TARBALL_NAME = kubectl-add_config-$(VERSION).tar.gz


.PHONY: clean build

clean:
	@-rm -fr $(BIN_DIR)
	@-rm -fr $(DIST_DIR)

build: $(BIN_DIR)/kubectl-add_config-linux $(BIN_DIR)/kubectl-add_config-darwin $(BIN_DIR)/kubectl-add_config-windows.exe

$(BIN_DIR)/kubectl-add_config-linux: *.go
	@echo ">> Build [LINUX]: $@"
	@GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $@ $^
	@chmod +x $@

$(BIN_DIR)/kubectl-add_config-darwin: *.go
	@echo ">> Build [DARWIN]: $@"
	@GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $@ $^
	@chmod +x $@

$(BIN_DIR)/kubectl-add_config-windows.exe: *.go
	@echo ">> Build [WINDOWS]: $@"
	@GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $@ $^
	@chmod +x $@
