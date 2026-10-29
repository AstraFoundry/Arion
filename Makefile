.DEFAULT_GOAL := help
SUITE ?= tests/e2e/desktop-scenarios.json
export SUITE DRIVER OUT
.PHONY: build check contributors-update core-check desktop-check desktop-dev desktop-libs docs e2e e2e-macos e2e-paste e2e-scenarios format help i18n-check install keyboard-check keyboard-publisher-check keyboard-stage keyboard-verify keyboard-vocabulary-check lint notifications-check notifications-prepare reset rewrite-e2e site-assets site-check site-deploy sync-check sync-integration test tooling-test updater-check updater-e2e updates-check workflows-check

# Make treats the component as a second goal; validate before any recipes run.
ifneq ($(filter build,$(MAKECMDGOALS)),)
ifneq ($(words $(MAKECMDGOALS)),2)
$(error Usage: make build <desktop|site>)
endif
ifneq ($(firstword $(MAKECMDGOALS)),build)
$(error Usage: make build <desktop|site>)
endif
ifeq ($(filter $(word 2,$(MAKECMDGOALS)),desktop site),)
$(error Unsupported build component: $(word 2,$(MAKECMDGOALS)). Choose desktop or site)
endif
.PHONY: $(word 2,$(MAKECMDGOALS))
$(word 2,$(MAKECMDGOALS)): build
	@:
endif

# Validate every reset goal before any recipe can mutate the profile.
ifneq ($(filter reset,$(MAKECMDGOALS)),)
ifneq ($(MAKECMDGOALS),reset desktop onboarding)
ifneq ($(firstword $(MAKECMDGOALS)),reset)
$(error Usage: make reset [keep-models] [keep-preferences] [keep-vocabulary] [keep-keyboard] [keep-usage])
endif
ifneq ($(filter-out reset keep-models keep-preferences keep-vocabulary keep-keyboard keep-usage keep-credentials keep-permissions keep-input-method keep-system,$(MAKECMDGOALS)),)
$(error Unsupported reset suffix; see docs/operations/local-dev.md)
endif
endif
ifneq ($(filter-out 0 1,$(DRY_RUN)),)
$(error DRY_RUN must be 0 or 1)
endif
.PHONY: $(filter-out reset,$(MAKECMDGOALS))
ifneq ($(filter-out reset,$(MAKECMDGOALS)),)
$(filter-out reset,$(MAKECMDGOALS)): reset
	@:
endif
endif

install: ## Install the pinned pnpm workspace with its frozen lockfile
	@pnpm install --frozen-lockfile

check: docs workflows-check test ## Run portable repository and automation gates

workflows-check: ## Validate GitHub Actions syntax, runners and reusable workflow calls
	@pnpm workflows:check

docs: ## Validate documentation, JSON and layer boundaries
	@pnpm docs:check

i18n-check: ## Validate translation keys, templates and hardcoded UI copy
	@pnpm i18n:check

core-check: ## Run portable core tests, vet and formatting checks
	@pnpm core:check

sync-check: ## Test and vet the sync relay module
	@cd services/sync && GOWORK=off go test ./... && GOWORK=off go vet ./...

updates-check: ## Test and vet the update channel service
	@node scripts/check-go.mjs updates

sync-integration: ## Run the PostgreSQL relay test; requires ARION_SYNC_TEST_DATABASE_URL
	@cd services/sync && GOWORK=off go test -tags integration ./infrastructure/postgres/

tooling-test: ## Test public driver and native scenario contracts
	@pnpm test:tooling

desktop-check: ## Compile/check native shell for current host
	@pnpm desktop:check

desktop-libs: ## Stage speech runtimes, the LLM worker, localized permission copy and the macOS input method
	@node scripts/stage-desktop-resources.mjs

build: ## Build a component: make build desktop | make build site (desktop signing follows ENV)
	@node scripts/build.mjs $(word 2,$(MAKECMDGOALS))

desktop-dev: ## Launch the native app in development mode
	@node scripts/desktop-command.mjs dev

reset: ## Reset local data and system state; append keep-models or other keep-* suffixes; DRY_RUN=1 previews
ifeq ($(MAKECMDGOALS),reset desktop onboarding)
	@node scripts/reset-desktop-onboarding.mjs
else
	@node scripts/reset-desktop.mjs $(filter-out reset,$(MAKECMDGOALS)) $(if $(filter 1,$(DRY_RUN)),--dry-run)
endif

site-check: ## Check the marketing site: dictionaries, types, lint, tests and the static export
	@pnpm site:check

site-assets: ## Regenerate the site's backdrop, icon and Open Graph images from repository sources
	@pnpm site:assets

site-deploy: ## Build and deploy the marketing site (needs wrangler auth)
	@node scripts/build.mjs site
	@pnpm site:deploy

test: ## Run portable product and public tooling tests
	@pnpm test

lint: docs ## Validate architecture and documentation

format: ## Format Go modules
	@pnpm format

e2e: ## Run production native window lifecycle on macOS/Windows
	@pnpm e2e:native

e2e-paste: ## Run guarded system paste against owned macOS editors
	@pnpm e2e:paste

e2e-macos: ## Run owned macOS product scenarios against the packaged Arion.app
	@node scripts/run-macos-scenarios.mjs

e2e-scenarios: ## Run OS product scenarios; requires DRIVER and OUT
	@node scripts/evaluation-command.mjs e2e

rewrite-e2e: SUITE = tests/e2e/contextual-rewrite-scenarios.json
rewrite-e2e: e2e-scenarios ## Run rewrite ownership/privacy cases on actual macOS/Windows

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Research-only ASR -> correction -> intent rewrite reproduction.
keyboard-stage: ## Stage pinned offline Pinyin runtime, schemas, data and notices
	@pnpm keyboard:stage

contributors-update: ## Refresh the contributors and avatars shown in Settings › About from GitHub
	@pnpm contributors:update

keyboard-check: ## Run actual macOS typing and encrypted learning checks in isolated fixtures
	@pnpm keyboard:check

keyboard-publisher-check: ## Validate actual Developer ID socket peers without installing an input source
	@pnpm keyboard:publisher:check

keyboard-vocabulary-check: ## Validate signed native vocabulary publication, restart and deletion
	@pnpm keyboard:vocabulary:check

keyboard-verify: ## Verify staged universal IMK bundle and its offline data closure
	@pnpm keyboard:verify

updater-check: ## Verify the maintained MyGo update-channel patch
	@pnpm updater:check

updater-e2e: ## Run the isolated native packaged updater lifecycle
	pnpm updater:e2e

notifications-check: ## Check notification APIs, registration and platform builds
	pnpm notifications:check

notifications-prepare: ## Build an isolated native notification consent probe
	pnpm notifications:prepare


.PHONY: build check contributors-update core-check desktop-check desktop-dev desktop-libs docs e2e e2e-macos e2e-paste e2e-scenarios format help i18n-check install keyboard-check keyboard-publisher-check keyboard-stage keyboard-verify keyboard-vocabulary-check lint notifications-check notifications-prepare reset rewrite-e2e site-assets site-check site-deploy sync-check sync-integration test tooling-test updater-check updater-e2e updates-check workflows-check
