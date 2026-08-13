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
VERSION     ?= 0.2.0-rc.1
DEV_HOME    ?= $(CURDIR)/devhome

WEB_SRC     := web/src
WEB_DIST    := cmd/bonghos/webdist

LDFLAGS     := -s -w -X github.com/Chansovisoth/Bonghos/internal/app.Version=$(VERSION)

# Selected dependencies are maintained as local replacements under third_party/.
# Remaining modules use the normal Go proxy and checksum database.

.PHONY: all build web test vet fmt fmt-check run clean tidy install-local help

all: build

## web: copy the dependency-free frontend into the Go embed directory.
## There is no JavaScript build step by design — see web/README.md.
web:
	@mkdir -p $(WEB_DIST)
	@find $(WEB_DIST) -mindepth 1 -maxdepth 1 ! -name .gitkeep -delete
	@cp $(WEB_SRC)/* $(WEB_DIST)/
	@for asset in index.html app.js style.css theme-init.js Telegram.png Discord.png server-placeholder.png; do \
		test -f $(WEB_DIST)/$$asset || { echo "frontend build produced no $$asset"; exit 1; }; \
	done
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
##
## IMPORTANT: the SQLite driver (mattn/go-sqlite3) is a cgo package. Building
## with CGO_ENABLED=0 produces a binary that links and runs until the first
## database access, then fails with "go-sqlite3 requires cgo to work". So cgo
## must stay enabled, and arm64 needs a cross-compiler:
##
##     sudo apt install gcc-aarch64-linux-gnu
cross: web
	@mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-amd64 ./cmd/bonghos
	@if command -v aarch64-linux-gnu-gcc >/dev/null 2>&1; then \
		GOOS=linux GOARCH=arm64 CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
			go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-arm64 ./cmd/bonghos; \
		echo "Built $(BIN_DIR)/$(BINARY)-linux-arm64"; \
	else \
		echo "SKIPPED arm64: aarch64-linux-gnu-gcc not found."; \
		echo "  Install it with: sudo apt install gcc-aarch64-linux-gnu"; \
	fi
	@echo "Cross-compile finished; artifacts in $(BIN_DIR)/"

## verify-binaries: prove each built binary can actually open a database.
## Guards against the CGO_ENABLED=0 trap above, which only fails at runtime.
verify-binaries:
	@for b in $(BIN_DIR)/$(BINARY)-linux-*; do \
		case "$$b" in *arm64) echo "skip $$b (cannot execute on this host)"; continue;; esac; \
		tmp=$$(mktemp -d); mkdir -p "$$tmp/home"; \
		if "$$b" --home "$$tmp/home" user list >/dev/null 2>&1; then \
			echo "ok   $$b opens its database"; \
		else \
			echo "FAIL $$b cannot open a database (cgo missing?)"; rm -rf "$$tmp"; exit 1; \
		fi; \
		rm -rf "$$tmp"; \
	done

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
