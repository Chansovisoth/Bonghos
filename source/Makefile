# Bonghos — developer convenience commands.
#
# Normal users do not need this file. They use the root ./setup.sh instead.
#
#   make build     build ./bin/bonghos with the Web UI embedded
#   make test      run the test suite
#   make vet       run go vet
#   make fmt       gofmt the tree
#   make run       build and run against ./devhome
#   make clean     remove build artifacts

BINARY      := bonghos
BIN_DIR     := bin
OUT         := $(BIN_DIR)/$(BINARY)
VERSION     ?= 0.1.0
DEV_HOME    ?= $(CURDIR)/devhome

WEB_SRC     := web/src
WEB_DIST    := cmd/bonghos/webdist

LDFLAGS     := -s -w -X github.com/Chansovisoth/Bonghos/internal/app.Version=$(VERSION)

# Dependencies are vendored under third_party/, so builds work offline.
export GOPROXY ?= direct
export GOSUMDB ?= off

.PHONY: all build web test vet fmt fmt-check run clean tidy install-local help

all: build

## web: copy the dependency-free frontend into the Go embed directory.
## There is no JavaScript build step by design — see web/README.md.
web:
	@mkdir -p $(WEB_DIST)
	@rm -f $(WEB_DIST)/*.html $(WEB_DIST)/*.css $(WEB_DIST)/*.js
	@cp $(WEB_SRC)/* $(WEB_DIST)/
	@test -f $(WEB_DIST)/index.html || (echo "frontend build produced no index.html" && exit 1)
	@echo "Web UI prepared for embedding"

build: web
	@mkdir -p $(BIN_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(OUT) ./cmd/bonghos
	@echo "Built $(OUT)"

test:
	go test ./internal/...

test-race:
	go test -race ./internal/...

vet:
	go vet ./cmd/... ./internal/...

fmt:
	gofmt -w ./cmd ./internal

fmt-check:
	@unformatted="$$(gofmt -l ./cmd ./internal)"; \
	if [ -n "$$unformatted" ]; then \
		echo "These files need gofmt:"; echo "$$unformatted"; exit 1; \
	fi
	@echo "Formatting is clean"

check: fmt-check vet test

## run: start a development control plane against ./devhome.
## Run 'make setup-dev' once first to create the Owner account.
run: build
	$(OUT) --home $(DEV_HOME) serve

setup-dev: build
	$(OUT) --home $(DEV_HOME) setup

doctor: build
	$(OUT) --home $(DEV_HOME) doctor

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
	rm -f $(WEB_DIST)/*.html $(WEB_DIST)/*.css $(WEB_DIST)/*.js
	@echo "Cleaned build artifacts (devhome and runtime data untouched)"

## cross: build release binaries for both supported architectures.
cross: web
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-amd64 ./cmd/bonghos
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-arm64 ./cmd/bonghos
	@echo "Cross-compiled into $(BIN_DIR)/"

help:
	@echo "Bonghos developer commands:"
	@echo "  make build       Build $(OUT)"
	@echo "  make test        Run tests"
	@echo "  make test-race   Run tests with the race detector"
	@echo "  make vet         Run go vet"
	@echo "  make fmt         Format the tree"
	@echo "  make check       fmt-check + vet + test"
	@echo "  make setup-dev   Create a dev Owner account in $(DEV_HOME)"
	@echo "  make run         Run the control plane against $(DEV_HOME)"
	@echo "  make cross       Build linux-amd64 and linux-arm64 binaries"
	@echo "  make clean       Remove build artifacts"
	@echo ""
	@echo "Normal users should use the root ./setup.sh instead."
