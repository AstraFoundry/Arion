.DEFAULT_GOAL := help
.PHONY: workflows-check
.PHONY: memory-asr-bench memory-asr-report
export MEMORY_SELECTION
export MEMORY_SPLIT MEMORY_DRIVER CONTEXT_DRIVER WORKER_PYTHON MODEL_METADATA MEMORY_BIN_DIR
.PHONY: memory-loop-build memory-loop-verify memory-loop-bench memory-loop-report memory-context-bench memory-context-report
.PHONY: phonetic-extract phonetic-bench phonetic-report benchmark-archive-check benchmark-archive-report
export PHONETIC_CONFIG PHONE_MODELS RUNTIME_ARTIFACTS LLM_ARTIFACTS EVIDENCE PHONETIC_PREFLIGHT
CONFIG ?= benchmarks/configs/desktop-factorial.json
PLAN ?= .workspace/benchmarks/plan.json
SUITE ?= tests/e2e/desktop-scenarios.json
REWRITE_CONFIG ?= benchmarks/configs/desktop-rewrite.json
REWRITE_PLAN ?= .workspace/benchmarks/rewrite-plan.json
export CORPUS DRIVER OUT RESULTS MANIFEST BUNDLE SEGMENT LLM_MODEL LLM_RUNTIME VARIANT RUNS CONFIG PLAN SUITE REWRITE_CONFIG REWRITE_PLAN REVIEW RATINGS RATERS RATING_SEED EVAL_PYTHON MODELS MODEL_IDS QWEN_CONFIG QWEN_PROMPTS QWEN_ASR AUDIO_CORPUS PREFIX_CONFIG CASES DEV TEST REPS

.PHONY: install check test lint format docs core-check sync-check sync-integration tooling-test desktop-check desktop-libs desktop-build desktop-dev site-check site-build site-assets site-deploy e2e e2e-macos e2e-scenarios bench-plan ablation-plan bench-validate bench-run bench-report datasets-check i18n-check stream-plan stream-bench rewrite-plan rewrite-validate rewrite-run rewrite-report rewrite-human-export rewrite-human-report rewrite-e2e qwen-env qwen-provision qwen-diagnostic qwen-report qwen-asr-diagnostic qwen-prefix-diagnostic live-fixtures live-text-bench live-audio-bench live-report scene-bench context-ocr memory-bench help

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

sync-integration: ## Run the PostgreSQL relay test; requires ARION_SYNC_TEST_DATABASE_URL
	@cd services/sync && GOWORK=off go test -tags integration ./infrastructure/postgres/

tooling-test: ## Test benchmark metrics, leakage guards and driver conformance
	@pnpm test:tooling

desktop-check: ## Compile/check native shell for current host
	@pnpm desktop:check

desktop-libs: ## Stage speech runtimes, the LLM worker, localized permission copy and the macOS input method
	@node scripts/stage-desktop-resources.mjs

desktop-build: ## Package and verify the app, its installer and (with MYGO_UPDATER_PRIVATE_KEY) signed updates
	@node scripts/desktop-command.mjs build

desktop-dev: ## Launch the native app in development mode
	@node scripts/desktop-command.mjs dev

site-check: ## Check the marketing site: dictionaries, types, lint, tests and the static export
	@pnpm site:check

site-build: ## Build the marketing site's static export into apps/site/out
	@pnpm site:build

site-assets: ## Regenerate the site's backdrop, icon and Open Graph images from repository sources
	@pnpm site:assets

site-deploy: site-build ## Deploy apps/site/out to Cloudflare Workers static assets (needs wrangler auth)
	@pnpm site:deploy

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

live-text-bench: ## Rewrite a text corpus through the production LLM worker with live edits; CORPUS, LLM_MODEL, OUT required; SCENE=corpus|auto|<scene>, CORRECT=1 applies rule correction first, CONTEXT=cursor,history,screen with SCREEN=oracle|accurate|fast, OCR and CONTEXT_AS=prompt|terms|both
	@node scripts/live-bench-command.mjs text

live-audio-bench: ## Replay fixtures through the live dictation pipeline; MANIFEST, CORPUS, MODELS, BUNDLE, OUT required; SEGMENT=1, LLM_MODEL optional
	@node scripts/live-bench-command.mjs audio

live-report: ## Aggregate scored live benchmark rows into a content-free report; RUNS and OUT required
	@node scripts/live-bench-command.mjs report

context-ocr: ## Recognize the window images of a context corpus with the production Vision OCR at both levels; CORPUS and OUT required
	@node scripts/live-bench-command.mjs ocr

scene-bench: ## Score scene selection against labelled app/site contexts; CONTEXTS defaults to benchmarks/configs/scene-contexts.json
	@node scripts/live-bench-command.mjs scene

memory-bench: ## Edit learning and vocabulary recall quality, latency and faults; requires OUT, optional CORPUS with DEV and TEST results
	@node scripts/memory-bench-command.mjs

memory-asr-bench: ## Run recognition-stage memory; CORPUS fixture directory, MODELS model directory and OUT required; MEMORY_SELECTION optional
	@node scripts/asr-memory-bench-command.mjs run

memory-asr-report: ## Replay recognition-stage entity and speech metrics without a model; CORPUS fixture directory, RESULTS and OUT required
	@node scripts/asr-memory-bench-command.mjs report

memory-loop-build: ## Build production extraction/vault and prompt/validation research drivers; MEMORY_BIN_DIR optional
	@node scripts/memory-loop-command.mjs build

memory-loop-verify: ## Verify exact bytes of the closed-loop audio and text-context research archives
	@node scripts/memory-loop-command.mjs verify

memory-loop-bench: ## Run empty-vault learning and later audio recognition; CORPUS, MEMORY_SPLIT, MODELS, MEMORY_DRIVER, OUT required
	@node scripts/memory-loop-command.mjs run

memory-loop-report: ## Replay closed-loop memory observations; CORPUS fixture root, RESULTS and new OUT JSON required
	@node scripts/memory-loop-command.mjs report

memory-context-bench: ## Run text-context research; requires CORPUS, MEMORY_SPLIT, MODELS, MODEL_METADATA, WORKER_PYTHON, MEMORY_DRIVER, CONTEXT_DRIVER, OUT
	@node scripts/memory-loop-command.mjs context

memory-context-report: ## Replay text-context research, separate from audio/OCR acceptance; CORPUS, RESULTS and OUT required
	@node scripts/memory-loop-command.mjs context-report

phonetic-extract: ## Extract real ASR scores and audio IPA; CORPUS, EVAL_PYTHON, MODELS, PHONE_MODELS, RUNTIME_ARTIFACTS and OUT required
	@node scripts/phonetic-bench-command.mjs extract

phonetic-bench: ## Run paired phonetic correction ablations; CORPUS, EVAL_PYTHON, MODELS, RUNTIME_ARTIFACTS, LLM_ARTIFACTS, EVIDENCE, PHONETIC_PREFLIGHT and OUT required
	@node scripts/phonetic-bench-command.mjs run

phonetic-report: ## Score a completed phonetic ablation; RESULTS and OUT required
	@node scripts/phonetic-bench-command.mjs report

benchmark-archive-check: ## Verify archived benchmark bytes, linked evidence, audio and completed checkpoint selection
	@pnpm bench:archive:verify

benchmark-archive-report: ## Regenerate and compare eight archived report families without inference; OUT required
	@pnpm bench:archive:report --out "$(OUT)"

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)
