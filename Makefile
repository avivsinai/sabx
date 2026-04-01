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
CODESIGN ?= codesign
CODESIGN_IDENTITY ?= -
CODESIGN_IDENTIFIER ?= io.github.avivsinai.sabx
LDFLAGS := -s -w \
	-X github.com/avivsinai/sabx/internal/buildinfo.Version=$(VERSION) \
	-X github.com/avivsinai/sabx/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/avivsinai/sabx/internal/buildinfo.Date=$(BUILD_DATE)

.PHONY: build check-skills release-skills
build: $(BIN_DIR)/sabx

# Skill integrity: skills/ is canonical, .claude/skills/ and .agents/skills/ are symlinks
check-skills:
	@echo "Checking skill symlinks..."
	@test -L .claude/skills/sabx || (echo "❌ .claude/skills/sabx is not a symlink" && exit 1)
	@test -L .agents/skills/sabx || (echo "❌ .agents/skills/sabx is not a symlink" && exit 1)
	@test "$$(readlink .claude/skills/sabx)" = "../../skills/sabx" || (echo "❌ .claude/skills/sabx target is not ../../skills/sabx" && exit 1)
	@test "$$(readlink .agents/skills/sabx)" = "../../skills/sabx" || (echo "❌ .agents/skills/sabx target is not ../../skills/sabx" && exit 1)
	@diff -rq skills/sabx .claude/skills/sabx || (echo "❌ .claude/skills/sabx content mismatch" && exit 1)
	@echo "✓ Skill symlinks valid"

$(BIN_DIR)/sabx: $(SOURCES) go.mod go.sum
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/sabx $(CMD)
	@if [ "$$(uname -s)" = "Darwin" ]; then \
		$(CODESIGN) --force --sign $(CODESIGN_IDENTITY) --identifier $(CODESIGN_IDENTIFIER) $@; \
	fi

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

release-skills:
	@test -n "$(RELEASE_VERSION)" || (echo "usage: make release-skills RELEASE_VERSION=0.1.4" && exit 1)
	./scripts/release-skills.sh "$(RELEASE_VERSION)"
