.DEFAULT_GOAL := help

## Node
NPM ?= npm
NPX ?= npx
NODE_MODULES := node_modules/.package-lock.json

## Python helpers
PYTHON ?= python3
SCRIPT_PYTHON := $(abspath bin/python-env/bin/python3)
SCRIPT_REQUIREMENTS := bin/python-env/.requirements

## Tool Versions
# renovate: datasource=github-releases depName=golangci/golangci-lint
GOLANGCI_LINT_VERSION ?= v2.14.0

# renovate: datasource=github-releases depName=gi8lino/dev-tools
DEV_TOOLS_VERSION ?= v0.9.0

# renovate: datasource=github-releases depName=kumbuka-me/cli
KUMBUKA_CLI_VERSION ?= v0.8.2

## Shared development tools
include bin/dev-tools.mk
include $(call dev-tools-module,tag)
include $(call dev-tools-module,help)

## Project-local tools
GOLANGCI_LINT := bin/golangci-lint
KUMBUKA_PLUGIN := bin/kumbuka-plugin
KUMBUKA_CLI ?= bin/kumbuka-cli-$(KUMBUKA_CLI_VERSION)
KUMBUKA_CLI_ASSET ?= kumbuka-cli_{version}_{os}_{arch}.tar.gz

## GitHub
GH ?= gh
GITHUB_REPOSITORY ?= kumbuka-me/plugins
RELEASE_WORKFLOW ?= release.yml
RELEASE_REF ?= main
RELEASE_REMOTE ?= origin

## Build Configuration
DIST ?= dist
PLUGIN ?=
BUMP ?=
PLUGIN_DIRS := $(sort $(patsubst %/plugin.yaml,%,$(wildcard */plugin.yaml)))

DOCS_DIR ?= ../docs

## Previews
PREVIEW_BROWSER_CHANNEL ?=
PREVIEW_SKIP_BROWSER_INSTALL ?= 0

## Formatting
PRETTIER_SOURCES := \
	README.md \
	catalog.json \
	"*/DEVELOPMENT.md" \
	"*/README.md" \
	"*/preview.md" \
	"*/preview.static.md" \
	"*/plugin.yaml" \
	"*/browser.ts" \
	".github/workflows/*.yml" \
	.github/renovate.json \
	"scripts/previews/*.mjs" \
	scripts/previews/README.md \
	package.json


##@ Development

.PHONY: download
download: $(NODE_MODULES) $(SCRIPT_REQUIREMENTS) dev-tools $(KUMBUKA_PLUGIN) ## Download Go, Node, Python, and development dependencies.
	go mod download

.PHONY: check-plugins
check-plugins: ## Verify first-party plugin identity and version metadata.
	./scripts/validation/plugins.sh

.PHONY: browser
browser: $(NODE_MODULES) ## Build browser TypeScript assets.
	@set -eu; \
	for plugin in $(PLUGIN_DIRS); do \
		./scripts/build/browser.sh "$$plugin"; \
	done

.PHONY: build
build: $(NODE_MODULES) $(KUMBUKA_PLUGIN) check-plugins ## Build all versioned plugin packages.
	rm -rf "$(DIST)"
	@mkdir -p "$(DIST)"
	@set -eu; \
	for plugin in $(PLUGIN_DIRS); do \
		./scripts/build/plugin.sh "$$plugin" "$(DIST)"; \
	done

.PHONY: build-plugin
build-plugin: $(NODE_MODULES) $(KUMBUKA_PLUGIN) check-plugins ## Build PLUGIN=<name> as a versioned package.
	@test -n "$(PLUGIN)" || { echo "PLUGIN is required" >&2; exit 1; }
	./scripts/build/plugin.sh "$(PLUGIN)" "$(DIST)"

.PHONY: docs
docs: $(NODE_MODULES) $(SCRIPT_REQUIREMENTS) ## Generate plugin pages and previews into DOCS_DIR.
	$(SCRIPT_PYTHON) scripts/docs/generate.py --docs "$(DOCS_DIR)"
	$(NPX) prettier --write "$(DOCS_DIR)/content/extensions/*.md"

.PHONY: previews
previews: $(NODE_MODULES) $(KUMBUKA_CLI) $(KUMBUKA_PLUGIN) $(SCRIPT_REQUIREMENTS) ## Rebuild every plugin preview from its preview.md.
	@SCRIPT_PYTHON="$(SCRIPT_PYTHON)" \
		PREVIEW_BROWSER_CHANNEL="$(PREVIEW_BROWSER_CHANNEL)" \
		PREVIEW_SKIP_BROWSER_INSTALL="$(PREVIEW_SKIP_BROWSER_INSTALL)" \
		KUMBUKA_CLI="$(abspath $(KUMBUKA_CLI))" \
		./scripts/previews/run.sh

.PHONY: previews-check
previews-check: previews ## Rebuild previews and fail when generated preview images changed.
	git diff --exit-code -- '*/assets/preview.png'

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

.PHONY: test-browser
test-browser: $(NODE_MODULES) $(KUMBUKA_CLI) $(KUMBUKA_PLUGIN) $(SCRIPT_REQUIREMENTS) ## Run focused Playwright plugin behavior tests.
	@SCRIPT_PYTHON="$(SCRIPT_PYTHON)" \
		PREVIEW_BROWSER_CHANNEL="$(PREVIEW_BROWSER_CHANNEL)" \
		PREVIEW_SKIP_BROWSER_INSTALL="$(PREVIEW_SKIP_BROWSER_INSTALL)" \
		PREVIEW_PLUGINS="includes" \
		PREVIEW_ASSERT_ONLY=1 \
		KUMBUKA_CLI="$(abspath $(KUMBUKA_CLI))" \
		./scripts/previews/run.sh

.PHONY: cover
cover: ## Display Go test coverage.
	go test -coverprofile=coverage.out -covermode=set -count=1 -timeout=3m ./...
	go tool cover -html=coverage.out

.PHONY: clean
clean: ## Remove generated plugin packages and local binaries.
	rm -rf "$(DIST)"
	rm -rf */dist
	rm -f */assets/plugin.js
	rm -f simple-icons/assets/icons.json
	rm -f coverage.out coverage.html


##@ Versioning

.PHONY: release
release: check-plugins ## Interactively version, validate, commit, tag, and push plugin releases.
	+RELEASE_MAKE="$(MAKE)" RELEASE_REMOTE="$(RELEASE_REMOTE)" RELEASE_REF="$(RELEASE_REF)" ./scripts/release/run.sh

.PHONY: version
version: check-plugins ## Interactively bump one plugin version.
	./scripts/version/bump.sh

.PHONY: version-plugin
version-plugin: check-plugins ## Bump one plugin. Usage: make version-plugin PLUGIN=tables BUMP=patch
	@test -n "$(PLUGIN)" || { echo "PLUGIN is required" >&2; exit 1; }
	@test -n "$(BUMP)" || { echo "BUMP is required: patch, minor, or major" >&2; exit 1; }
	./scripts/version/bump.sh "$(PLUGIN)" "$(BUMP)"

.PHONY: version-all
version-all: check-plugins ## Bump every plugin. Usage: make version-all BUMP=patch
	@test -n "$(BUMP)" || { echo "BUMP is required: patch, minor, or major" >&2; exit 1; }
	./scripts/version/bump.sh --all "$(BUMP)"

.PHONY: tag-plugin
tag-plugin: check-plugins ## Tag one plugin's current version. Usage: make tag-plugin PLUGIN=tables
	@test -n "$(PLUGIN)" || { echo "PLUGIN is required" >&2; exit 1; }
	./scripts/release/tag.sh "$(PLUGIN)"

.PHONY: tag-all
tag-all: check-plugins ## Tag all current plugin versions that are not already tagged.
	./scripts/release/tag.sh --all

.PHONY: catalog
catalog: ## Regenerate catalog.json from published plugin releases.
	GITHUB_REPOSITORY="$(GITHUB_REPOSITORY)" ./scripts/catalog/generate.sh catalog.json

.PHONY: check-releases
check-releases: ## Check whether every plugin's latest tag has a GitHub release.
	GH="$(GH)" GITHUB_REPOSITORY="$(GITHUB_REPOSITORY)" ./scripts/release/check.sh

.PHONY: release-missing
release-missing: ## Create releases for latest plugin tags that do not have one.
	GH="$(GH)" GITHUB_REPOSITORY="$(GITHUB_REPOSITORY)" RELEASE_WORKFLOW="$(RELEASE_WORKFLOW)" RELEASE_REF="$(RELEASE_REF)" ./scripts/release/missing.sh


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
lint-fix: fmt golangci-lint ## Format supported files, then run Go linters with fixes.
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

$(KUMBUKA_PLUGIN): $(GO_INSTALL_TOOL) go.mod
	@set -eu; \
	sdk_version="$$(go list -m -f '{{.Version}}' github.com/kumbuka-me/sdk)"; \
	if [ -z "$$sdk_version" ]; then \
		echo "unable to determine Kumbuka SDK version" >&2; \
		exit 1; \
	fi; \
	$(GO_INSTALL_TOOL) \
		--target "$@" \
		--package github.com/kumbuka-me/sdk/cmd/kumbuka-plugin \
		--tool-version "$$sdk_version"

$(KUMBUKA_CLI): $(GITHUB_RELEASE_INSTALL)
	@$(GITHUB_RELEASE_INSTALL) \
		--repo kumbuka-me/cli \
		--tag "v$(patsubst v%,%,$(KUMBUKA_CLI_VERSION))" \
		--asset "$(KUMBUKA_CLI_ASSET)" \
		--binary kumbuka-cli \
		--target "$@"


$(SCRIPT_REQUIREMENTS): scripts/requirements.txt
	$(PYTHON) -m venv bin/python-env
	$(SCRIPT_PYTHON) -m pip install -r scripts/requirements.txt
	@touch "$@"
