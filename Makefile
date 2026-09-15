.DEFAULT_GOAL := help

## Node
NPM ?= npm
NPX ?= npx
NODE_MODULES := node_modules/.package-lock.json

## Tool Versions
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.13.2

# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.7.0

## Shared development tools
include bin/dev-tools.mk
include $(call dev-tools-module,tag)
include $(call dev-tools-module,help)

## Project-local tools
GOLANGCI_LINT := bin/golangci-lint

## Build Configuration
DIST ?= dist
PLUGIN ?=
BUMP ?=
PLUGIN_DIRS := $(sort $(patsubst %/plugin.yaml,%,$(wildcard */plugin.yaml)))

## Formatting
PRETTIER_SOURCES := \
	README.md \
	"*/README.md" \
	"*/plugin.yaml" \
	"*/browser.ts" \
	".github/workflows/*.yml" \
	package.json


##@ Development

.PHONY: download
download: $(NODE_MODULES) dev-tools ## Download Go, Node, and development dependencies.
	go mod download

.PHONY: check-plugins
check-plugins: ## Verify first-party plugin identity and version metadata.
	./scripts/check-plugins.sh

.PHONY: browser
browser: $(NODE_MODULES) ## Build browser TypeScript assets.
	@set -eu; \
	for plugin in $(PLUGIN_DIRS); do \
		./scripts/build-browser.sh "$$plugin"; \
	done

.PHONY: build
build: $(NODE_MODULES) check-plugins ## Build all versioned plugin packages.
	rm -rf "$(DIST)"
	@mkdir -p "$(DIST)"
	@set -eu; \
	for plugin in $(PLUGIN_DIRS); do \
		./scripts/build-plugin.sh "$$plugin" "$(DIST)"; \
	done

.PHONY: build-plugin
build-plugin: $(NODE_MODULES) check-plugins ## Build PLUGIN=<name> as a versioned package.
	@test -n "$(PLUGIN)" || { echo "PLUGIN is required" >&2; exit 1; }
	./scripts/build-plugin.sh "$(PLUGIN)" "$(DIST)"

.PHONY: vet
vet: ## Run Go static analysis.
	go vet ./...

.PHONY: test
test: check-plugins vet ## Run executable plugin tests.
	go test -covermode=set -timeout=3m ./...

.PHONY: test-fresh
test-fresh: check-plugins vet ## Run executable plugin tests without the Go test cache.
	go test -covermode=set -count=1 -timeout=3m ./...

.PHONY: test-race
test-race: check-plugins vet ## Run executable plugin tests with the race detector.
	go test -race -count=1 -timeout=3m ./...

.PHONY: cover
cover: ## Display Go test coverage.
	go test -coverprofile=coverage.out -covermode=set -count=1 -timeout=3m ./...
	go tool cover -html=coverage.out

.PHONY: clean
clean: ## Remove generated plugin packages and local binaries.
	rm -rf "$(DIST)"
	rm -rf */dist
	rm -f tables/assets/plugin.js
	rm -f coverage.out coverage.html


##@ Versioning

.PHONY: version
version: check-plugins ## Interactively bump one plugin version.
	./scripts/version-plugins.sh

.PHONY: version-plugin
version-plugin: check-plugins ## Bump one plugin. Usage: make version-plugin PLUGIN=tables BUMP=patch
	@test -n "$(PLUGIN)" || { echo "PLUGIN is required" >&2; exit 1; }
	@test -n "$(BUMP)" || { echo "BUMP is required: patch, minor, or major" >&2; exit 1; }
	./scripts/version-plugins.sh "$(PLUGIN)" "$(BUMP)"

.PHONY: version-all
version-all: check-plugins ## Bump every plugin. Usage: make version-all BUMP=patch
	@test -n "$(BUMP)" || { echo "BUMP is required: patch, minor, or major" >&2; exit 1; }
	./scripts/version-plugins.sh --all "$(BUMP)"

.PHONY: tag-plugin
tag-plugin: check-plugins ## Tag one plugin's current version. Usage: make tag-plugin PLUGIN=tables
	@test -n "$(PLUGIN)" || { echo "PLUGIN is required" >&2; exit 1; }
	./scripts/tag-plugins.sh "$(PLUGIN)"

.PHONY: tag-all
tag-all: check-plugins ## Tag all current plugin versions that are not already tagged.
	./scripts/tag-plugins.sh --all


##@ Formatting

.PHONY: fmt
fmt: fmt-go fmt-prettier ## Format all supported files.

.PHONY: fmt-go
fmt-go: ## Format Go source.
	go fmt ./...

.PHONY: fmt-prettier
fmt-prettier: $(NODE_MODULES) ## Format Markdown, YAML, JSON, and TypeScript.
	$(NPX) prettier --write $(PRETTIER_SOURCES)

.PHONY: check-prettier
check-prettier: $(NODE_MODULES) ## Check Prettier formatting.
	$(NPX) prettier --check $(PRETTIER_SOURCES)


##@ Linting

.PHONY: lint
lint: check-plugins check-prettier lint-go ## Run all linters and formatting checks.

.PHONY: lint-go
lint-go: golangci-lint ## Run golangci-lint.
	$(call run-tool,$(GOLANGCI_LINT),run)

.PHONY: lint-fix
lint-fix: fmt-prettier golangci-lint ## Run linters and apply fixes.
	$(call run-tool,$(GOLANGCI_LINT),run --fix)


##@ Dependencies

$(NODE_MODULES): package.json package-lock.json
	$(NPM) ci

.PHONY: dev-tools
dev-tools: $(DEV_TAG) $(MAKE_HELP) $(GO_INSTALL_TOOL) ## Download pinned development tools.

.PHONY: golangci-lint
golangci-lint: $(GO_INSTALL_TOOL) ## Download golangci-lint locally if necessary.
	@$(GO_INSTALL_TOOL) \
		--target "$(GOLANGCI_LINT)" \
		--package github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
		--tool-version "$(GOLANGCI_LINT_VERSION)"
