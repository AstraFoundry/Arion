.DEFAULT_GOAL := help
CONFIG ?= benchmarks/configs/desktop-factorial.json
PLAN ?= .workspace/benchmarks/plan.json
SUITE ?= tests/e2e/desktop-scenarios.json
REWRITE_CONFIG ?= benchmarks/configs/desktop-rewrite.json
REWRITE_PLAN ?= .workspace/benchmarks/rewrite-plan.json
export CORPUS DRIVER OUT RESULTS MANIFEST BUNDLE SEGMENT LLM_MODEL LLM_RUNTIME VARIANT RUNS CONFIG PLAN SUITE REWRITE_CONFIG REWRITE_PLAN REVIEW RATINGS RATERS RATING_SEED EVAL_PYTHON MODELS MODEL_IDS QWEN_CONFIG QWEN_PROMPTS QWEN_ASR AUDIO_CORPUS PREFIX_CONFIG

.PHONY: install check test lint format docs core-check tooling-test desktop-check desktop-libs desktop-build desktop-dev e2e e2e-macos e2e-scenarios bench-plan ablation-plan bench-validate bench-run bench-report datasets-check i18n-check stream-plan stream-bench rewrite-plan rewrite-validate rewrite-run rewrite-report rewrite-human-export rewrite-human-report rewrite-e2e qwen-env qwen-provision qwen-diagnostic qwen-report qwen-asr-diagnostic qwen-prefix-diagnostic live-fixtures live-text-bench live-audio-bench live-report help

install: ## Install the pinned pnpm workspace with its frozen lockfile
	@pnpm install --frozen-lockfile

check: docs test ## Run portable repository and automation gates

docs: ## Validate documentation, JSON and layer boundaries
	@pnpm docs:check

i18n-check: ## Validate translation keys, templates and hardcoded UI copy
	@pnpm i18n:check

core-check: ## Run portable core tests, vet and formatting checks
	@pnpm core:check

tooling-test: ## Test benchmark metrics, leakage guards and driver conformance
	@pnpm test:tooling

desktop-check: ## Compile/check native shell for current host
	@pnpm desktop:check

# Speech runtimes are cgo libraries packaged in Contents/Resources/lib; the
# compiler wrapper drops the module cache rpath the speech bindings link with.
DESKTOP_ENV = CGO_ENABLED=1 CC="$(CURDIR)/scripts/cc-relocatable.sh" CGO_LDFLAGS="-Wl,-rpath,@executable_path/../Resources/lib"

desktop-libs: ## Stage speech runtimes, the LLM worker, localized permission copy and the macOS input method
	@node scripts/stage-desktop-localizations.mjs
	@node scripts/stage-native-libs.mjs darwin
	@node scripts/stage-llm-runtime.mjs darwin-arm64
	@node scripts/build-input-method.mjs

desktop-build: desktop-libs ## Package the native app, its disk image and (with MYGO_UPDATER_PRIVATE_KEY) signed updates
	@cd apps/desktop && $(DESKTOP_ENV) go tool mygo build
	@node scripts/verify-macos-app.mjs

desktop-dev: desktop-libs ## Launch the native app in development mode
	@cd apps/desktop && $(DESKTOP_ENV) go tool mygo dev

test: ## Run core and evaluation tooling tests
	@pnpm test

lint: docs ## Validate architecture and documentation

format: ## Format Go modules
	@pnpm format

e2e: ## Run production native window lifecycle on macOS/Windows
	@pnpm e2e:native

e2e-macos: ## Run owned macOS product scenarios against the packaged Arion.app
	@node scripts/run-macos-scenarios.mjs

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

rewrite-plan: ## Plan contextual rewrite ablations; REWRITE_CONFIG and REWRITE_PLAN optional
	@node scripts/evaluation-command.mjs rewrite-plan

rewrite-validate: ## Validate reviewed rewrite corpus and content hashes; requires CORPUS
	@node scripts/evaluation-command.mjs rewrite-validate

rewrite-run: ## Run actual native local rewriting; requires CORPUS, DRIVER and OUT
	@node scripts/evaluation-command.mjs rewrite-run

rewrite-report: ## Report rewrite proxies, latency and paired CIs; requires RESULTS
	@node scripts/evaluation-command.mjs rewrite-report

rewrite-human-export: ## Export private blind rating tasks; requires RESULTS and OUT
	@node scripts/evaluation-command.mjs rewrite-human-export

rewrite-human-report: ## Validate independent votes; requires RESULTS, REVIEW and RATINGS
	@node scripts/evaluation-command.mjs rewrite-human-report

rewrite-e2e: SUITE = tests/e2e/contextual-rewrite-scenarios.json
rewrite-e2e: e2e-scenarios ## Run rewrite ownership/privacy cases on actual macOS/Windows

qwen-env: ## Provision a private, pinned Python CPU diagnostic environment using uv
	@node scripts/qwen-diagnostic-command.mjs env

qwen-provision: ## Explicitly download pinned weights; MODEL_IDS required; QWEN_ASR=1 optional
	@node scripts/qwen-diagnostic-command.mjs provision

qwen-diagnostic: ## Run unreviewed development inference; CORPUS and OUT required
	@node scripts/qwen-diagnostic-command.mjs run

qwen-report: ## Recompute development proxies; CORPUS and RESULTS required
	@node scripts/qwen-diagnostic-command.mjs report

qwen-asr-diagnostic: ## Run offline Qwen ASR; CORPUS, DRIVER and OUT required
	@node scripts/qwen-diagnostic-command.mjs asr

qwen-prefix-diagnostic: ## Compare real ASR replay and ahead-of-stop prefix prefill; CORPUS, AUDIO_CORPUS, DRIVER, OUT required
	@node scripts/qwen-diagnostic-command.mjs prefix

live-fixtures: ## Synthesize pause-segmented speech fixtures from a text corpus; CORPUS and OUT required
	@node scripts/live-bench-command.mjs fixtures

live-text-bench: ## Rewrite a text corpus through the production LLM worker with live edits; CORPUS, LLM_MODEL, OUT required
	@node scripts/live-bench-command.mjs text

live-audio-bench: ## Replay fixtures through the live dictation pipeline; MANIFEST, CORPUS, MODELS, BUNDLE, OUT required; SEGMENT=1, LLM_MODEL optional
	@node scripts/live-bench-command.mjs audio

live-report: ## Aggregate scored live benchmark rows into a content-free report; RUNS and OUT required
	@node scripts/live-bench-command.mjs report

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
