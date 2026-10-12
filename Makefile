.DEFAULT_GOAL := help
CONFIG ?= benchmarks/configs/desktop-factorial.json
PLAN ?= .workspace/benchmarks/plan.json
SUITE ?= tests/e2e/desktop-scenarios.json
export CORPUS DRIVER OUT RESULTS CONFIG PLAN SUITE

.PHONY: install check test lint format docs core-check tooling-test desktop-check desktop-build desktop-dev e2e e2e-scenarios bench-plan ablation-plan bench-validate bench-run bench-report datasets-check stream-plan stream-bench help

install: ## Install the pinned pnpm workspace with its frozen lockfile
	@pnpm install --frozen-lockfile

check: docs test ## Run portable repository and automation gates

docs: ## Validate documentation, JSON and layer boundaries
	@pnpm docs:check

core-check: ## Run portable core tests, vet and formatting checks
	@pnpm core:check

tooling-test: ## Test benchmark metrics, leakage guards and driver conformance
	@pnpm test:tooling

desktop-check: ## Compile/check native shell for current host
	@pnpm desktop:check

desktop-build: ## Package native shell for current host
	@cd apps/desktop && go tool mygo build

desktop-dev: ## Launch native development shell
	@cd apps/desktop && go tool mygo dev

test: ## Run core and evaluation tooling tests
	@pnpm test

lint: docs ## Validate architecture and documentation

format: ## Format Go modules
	@pnpm format

e2e: ## Run production native window lifecycle on macOS/Windows
	@pnpm e2e:native

e2e-scenarios: ## Run OS product scenarios; requires DRIVER and OUT
	@node scripts/evaluation-command.mjs e2e

bench-plan: ## Generate a reproducible ablation plan; CONFIG and PLAN optional
	@node scripts/evaluation-command.mjs plan

ablation-plan: bench-plan ## Alias for the full factorial experiment plan

datasets-check: ## Verify pinned Hugging Face metadata; downloads no audio
	@pnpm datasets:check

stream-plan: CONFIG = benchmarks/configs/desktop-streaming.json
stream-plan: PLAN = .workspace/benchmarks/stream-plan.json
stream-plan: bench-plan ## Generate the chunk-size and endpoint streaming matrix

stream-bench: CONFIG = benchmarks/configs/desktop-streaming.json
stream-bench: bench-run ## Run streaming sweep; requires CORPUS, DRIVER and OUT

bench-validate: ## Validate licensed corpus, hashes and split isolation; requires CORPUS
	@node scripts/evaluation-command.mjs validate

bench-run: ## Run real inference matrix; requires CORPUS, DRIVER and OUT
	@node scripts/evaluation-command.mjs run

bench-report: ## Produce aggregate metrics and paired CIs; requires RESULTS
	@node scripts/evaluation-command.mjs report

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
