GO ?= go
BIN_DIR ?= bin
CMD := ./cmd/sabx
SOURCES := $(shell find cmd internal -name '*.go')

VERSION ?= $(shell \
	if git describe --tags --exact-match >/dev/null 2>&1; then \
		git describe --tags --exact-match; \
	else \
		short=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
		if git diff-index --quiet HEAD 2>/dev/null; then \
			echo "dev-$$short"; \
		else \
			echo "dev-$$short-dirty"; \
		fi; \
	fi \
)
COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/sabx/sabx/internal/buildinfo.Version=$(VERSION) \
	-X github.com/sabx/sabx/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/sabx/sabx/internal/buildinfo.Date=$(BUILD_DATE)

.PHONY: build
build: $(BIN_DIR)/sabx

$(BIN_DIR)/sabx: $(SOURCES) go.mod go.sum
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/sabx $(CMD)

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: test
test:
	$(GO) test ./...

.PHONY: lint
lint:
	golangci-lint run ./...

.PHONY: e2e
e2e:
	$(GO) test ./test/e2e -count=1

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: clean
clean:
	rm -rf $(BIN_DIR) dist/

.PHONY: snapshot
snapshot:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. See https://goreleaser.com/install"; exit 1; }
	goreleaser release --config tools/goreleaser.yaml --snapshot --clean --skip=publish

# Convenience wrapper for the container smoke harness
.PHONY: smoke
smoke:
	$(GO) test ./test/e2e -run TestSmokeAgainstSABContainer -count=1

