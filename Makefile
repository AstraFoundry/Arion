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

.PHONY: install check test lint format docs core-check sync-check updates-check sync-integration tooling-test build desktop-check desktop-libs desktop-dev site-check site-assets site-deploy e2e e2e-paste e2e-macos e2e-scenarios bench-plan ablation-plan bench-validate bench-run bench-report datasets-check i18n-check stream-plan stream-bench rewrite-plan rewrite-validate rewrite-run rewrite-report rewrite-human-export rewrite-human-report rewrite-e2e qwen-env qwen-provision qwen-diagnostic qwen-report qwen-asr-diagnostic qwen-prefix-diagnostic live-fixtures live-text-bench live-audio-bench live-report scene-bench context-ocr memory-bench help

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

tooling-test: ## Test benchmark metrics, leakage guards and driver conformance
	@pnpm test:tooling

desktop-check: ## Compile/check native shell for current host
	@pnpm desktop:check

desktop-libs: ## Stage speech runtimes, the LLM worker, localized permission copy and the macOS input method
	@node scripts/stage-desktop-resources.mjs

build: ## Build a component: make build desktop | make build site (desktop signing follows ENV)
	@node scripts/build.mjs $(word 2,$(MAKECMDGOALS))

desktop-dev: ## Launch the native app in development mode
	@node scripts/desktop-command.mjs dev

.PHONY: reset
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

test: ## Run core and evaluation tooling tests
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

# Research-only ASR -> correction -> intent rewrite reproduction.
.PHONY: chain-env chain-build chain-provision chain-fixtures chain-bench chain-report
chain-env:
	pnpm chain:env
chain-build:
	pnpm chain:build
chain-provision:
	pnpm chain:provision
chain-fixtures:
	pnpm chain:fixtures
chain-bench:
	pnpm chain:run --split $(SPLIT) --out $(OUT)
chain-report:
	pnpm chain:report $(RESULTS) $(OUT)
.PHONY: chain-test
chain-test:
	pnpm chain:test
.PHONY: chain-review
chain-review:
	pnpm chain:review $(RESULTS) $(OUT)
.PHONY: chain-semantic-report chain-semantic-test
chain-semantic-report:
	pnpm chain:semantic-report $(OUT)
chain-semantic-test:
	pnpm chain:semantic-test

.PHONY: chain-semantic-verify
chain-semantic-verify:
	pnpm chain:semantic-verify

.PHONY: chain-expanded-fixtures chain-expanded-bench
chain-expanded-fixtures:
	pnpm chain:expanded-fixtures
chain-expanded-bench:
	pnpm chain:expanded-run --split test --out $(OUT)
.PHONY: chain-expanded-report chain-expanded-test
chain-expanded-report:
	pnpm chain:expanded-report $(RESULTS) $(RATINGS) $(OUT)
chain-expanded-test:
	pnpm chain:expanded-test
.PHONY: chain-expanded-verify
chain-expanded-verify:
	pnpm chain:expanded-verify

.PHONY: rewrite-alignment-build rewrite-alignment-bench rewrite-alignment-report rewrite-alignment-verify rewrite-alignment-test
rewrite-alignment-build: ## Build isolated scene rewrite research workers
	@pnpm rewrite:alignment:build
rewrite-alignment-bench: ## Run frozen dev/test text study; SPLIT, OUT and optional SELECTION
	@pnpm rewrite:alignment:run --split $(SPLIT) --out $(OUT) $(if $(SELECTION),--selection $(SELECTION))
rewrite-alignment-report: ## Replay paired study; DEV, TEST and OUT required
	@pnpm rewrite:alignment:report --dev $(DEV) --test $(TEST) --out $(OUT)
rewrite-alignment-verify: ## Verify archived scene rewrite study without models
	@pnpm rewrite:alignment:verify
rewrite-alignment-test: ## Test report evidence integrity guards without inference
	@pnpm rewrite:alignment:test

.PHONY: keyboard-stage keyboard-check keyboard-verify
keyboard-stage: ## Stage pinned offline Pinyin runtime, schemas, data and notices
	@pnpm keyboard:stage

.PHONY: contributors-update
contributors-update: ## Refresh the contributors and avatars shown in Settings › About from GitHub
	@pnpm contributors:update

keyboard-check: ## Run actual macOS typing and encrypted learning checks in isolated fixtures
	@pnpm keyboard:check

keyboard-verify: ## Verify staged universal IMK bundle and its offline data closure
	@pnpm keyboard:verify

.PHONY: rewrite-prompt-bench rewrite-prompt-report rewrite-prompt-verify rewrite-prompt-test
rewrite-prompt-bench: ## Run frozen prompt study; PHASE, OUT and optional SELECTION
	@pnpm rewrite:prompt:run --phase $(PHASE) --out $(OUT) $(if $(SELECTION),--selection $(SELECTION))
rewrite-prompt-report: ## Replay complete prompt review; DEV, EVALUATION and OUT required
	@pnpm rewrite:prompt:report --dev $(DEV) --evaluation $(EVALUATION) --out $(OUT)
rewrite-prompt-verify: ## Verify prompt study archive without inference
	@pnpm rewrite:prompt:verify
rewrite-prompt-test: ## Check complete review and raw evidence integrity
	@pnpm rewrite:prompt:test

.PHONY: rewrite-context-bench rewrite-context-report rewrite-context-verify rewrite-context-test
rewrite-context-bench: ## Run frozen contextual ASR-text correction diagnostic; OUT required
	@pnpm rewrite:context:run --out $(OUT)
rewrite-context-report: ## Replay complete context and structure review; CONTEXT, STRUCTURE and OUT required
	@pnpm rewrite:context:report --context $(CONTEXT) --structure $(STRUCTURE) --out $(OUT)
rewrite-context-verify: ## Verify structure and contextual repair evidence
	@pnpm rewrite:context:verify
rewrite-context-test: ## Test contextual review integrity
	@pnpm rewrite:context:test

.PHONY: rewrite-ungated-bench rewrite-ungated-report rewrite-ungated-verify rewrite-ungated-test
rewrite-ungated-bench: ## Run frozen text-validator ablation; PHASE, OUT and optional SELECTION
	@pnpm rewrite:ungated:run --phase $(PHASE) --out $(OUT) $(if $(SELECTION),--selection $(SELECTION))
rewrite-ungated-report: ## Replay all raw/cleanup and strict/envelope/none deliveries; DEV, EVALUATION, OUT
	@pnpm rewrite:ungated:report --dev $(DEV) --evaluation $(EVALUATION) --out $(OUT)
rewrite-ungated-verify: ## Verify validator-ablation evidence without models
	@pnpm rewrite:ungated:verify
rewrite-ungated-test: ## Test complete review and validator-free delivery semantics
	@pnpm rewrite:ungated:test

.PHONY: rewrite-ocr-bench rewrite-ocr-report rewrite-ocr-verify rewrite-ocr-test
rewrite-ocr-bench: ## Run frozen OCR-alignment study; PHASE, OUT and optional SELECTION
	@pnpm rewrite:ocr:run --phase $(PHASE) --out $(OUT) $(if $(SELECTION),--selection $(SELECTION))
rewrite-ocr-report: ## Replay complete OCR/raw/cleanup policy review; DEV, EVALUATION, PARITY and OUT
	@pnpm rewrite:ocr:report --dev $(DEV) --evaluation $(EVALUATION) --parity $(PARITY) --out $(OUT)
rewrite-ocr-verify: ## Verify OCR-alignment evidence without inference
	@pnpm rewrite:ocr:verify
rewrite-ocr-test: ## Test OCR repair review and exact delivery exports
	@pnpm rewrite:ocr:test

.PHONY: rewrite-packaging-build rewrite-packaging-bench rewrite-packaging-report rewrite-packaging-verify rewrite-packaging-test rewrite-release-report rewrite-release-test
rewrite-packaging-build: ## Build the isolated packaging worker; OUT required
	@pnpm rewrite:packaging:build --out $(OUT)
rewrite-packaging-bench: ## Run paired packaging inference; PHASE, OUT, optional SELECTION
	@pnpm rewrite:packaging:run --phase $(PHASE) --out $(OUT) $(if $(SELECTION),--selection $(SELECTION))
rewrite-packaging-report: ## Replay packaging review; DEV, EVALUATION, OUT required
	@pnpm rewrite:packaging:report --dev $(DEV) --evaluation $(EVALUATION) --out $(OUT)
rewrite-packaging-verify: ## Verify packaging evidence hashes
	@pnpm rewrite:packaging:verify
rewrite-packaging-test: ## Test packaging evidence guards
	@pnpm rewrite:packaging:test
rewrite-release-report: ## Rebuild the offline iteration HTML from reviewed outputs
	@pnpm rewrite:release:report
rewrite-release-test: ## Test HTML evidence binding and safe embedding
	@pnpm rewrite:release:test

.PHONY: rewrite-entity-ocr rewrite-entity-bench rewrite-entity-report rewrite-entity-test rewrite-entity-verify rewrite-retrospective-verify
rewrite-entity-ocr: ## Run actual Vision on authored images; OUT required
	@pnpm rewrite:entity:ocr --out $(OUT)
rewrite-entity-bench: ## Run frozen entity policy study; PHASE, OUT, optional OCR
	@pnpm rewrite:entity:run --phase $(PHASE) --out $(OUT) $(if $(OCR),--ocr $(OCR))
rewrite-entity-report: ## Replay complete entity and OCR evidence; OUT required
	@pnpm rewrite:entity:report --out $(OUT)
rewrite-entity-test: ## Test exact OCR and review evidence bindings
	@pnpm rewrite:entity:test
rewrite-entity-verify: ## Verify entity evidence archive
	@pnpm rewrite:entity:verify
rewrite-retrospective-verify: ## Verify historical review against original inference
	@pnpm rewrite:retrospective:verify

.PHONY: rewrite-model-screen-report
rewrite-model-screen-report: ## Replay historical evidence; OUT required
	@pnpm rewrite:model-screen:report --out $(OUT)

.PHONY: rewrite-model-screen-verify
rewrite-model-screen-verify: ## Check historical evidence without inference
	@pnpm rewrite:model-screen:verify

.PHONY: rewrite-model-screen-test
rewrite-model-screen-test: ## Check historical evidence without inference
	@pnpm rewrite:model-screen:test

.PHONY: rewrite-4b-packaging-report
rewrite-4b-packaging-report: ## Replay historical evidence; OUT required
	@pnpm rewrite:4b-packaging:report --out $(OUT)

.PHONY: rewrite-4b-packaging-verify
rewrite-4b-packaging-verify: ## Check historical evidence without inference
	@pnpm rewrite:4b-packaging:verify

.PHONY: rewrite-4b-packaging-test
rewrite-4b-packaging-test: ## Check historical evidence without inference
	@pnpm rewrite:4b-packaging:test

.PHONY: rewrite-4b-entity-report
rewrite-4b-entity-report: ## Replay historical evidence; OUT required
	@pnpm rewrite:4b-entity:report --out $(OUT)

.PHONY: rewrite-4b-entity-verify
rewrite-4b-entity-verify: ## Check historical evidence without inference
	@pnpm rewrite:4b-entity:verify

.PHONY: rewrite-4b-entity-test
rewrite-4b-entity-test: ## Check historical evidence without inference
	@pnpm rewrite:4b-entity:test

.PHONY: rewrite-small-models-report
rewrite-small-models-report: ## Replay latest small-model evidence; OUT required
	@pnpm rewrite:small-models:report --out $(OUT)

.PHONY: rewrite-small-models-verify
rewrite-small-models-verify: ## Check latest small-model evidence
	@pnpm rewrite:small-models:verify

.PHONY: rewrite-small-models-test
rewrite-small-models-test: ## Check latest small-model evidence
	@pnpm rewrite:small-models:test

.PHONY: rewrite-small-models-run
rewrite-small-models-run: ## Run latest pinned CPU model; MODEL, RUNTIME, OUT required
	@pnpm rewrite:small-models:run --model $(MODEL) --runtime $(RUNTIME) --out $(OUT)

.PHONY: rewrite-small-prompt-report
rewrite-small-prompt-report: ## Replay frozen evidence; OUT required
	@pnpm rewrite:small-prompt:report --out $(OUT)

.PHONY: rewrite-small-prompt-test
rewrite-small-prompt-test: ## Check frozen rewrite experiment evidence
	@pnpm rewrite:small-prompt:test

.PHONY: rewrite-small-prompt-verify
rewrite-small-prompt-verify: ## Check frozen rewrite experiment evidence
	@pnpm rewrite:small-prompt:verify

.PHONY: rewrite-small-cache-report
rewrite-small-cache-report: ## Replay frozen evidence; OUT required
	@pnpm rewrite:small-cache:report --out $(OUT)

.PHONY: rewrite-small-cache-test
rewrite-small-cache-test: ## Check frozen rewrite experiment evidence
	@pnpm rewrite:small-cache:test

.PHONY: rewrite-small-cache-verify
rewrite-small-cache-verify: ## Check frozen rewrite experiment evidence
	@pnpm rewrite:small-cache:verify

.PHONY: rewrite-cache-failed-report
rewrite-cache-failed-report: ## Replay frozen evidence; OUT required
	@pnpm rewrite:cache-failed:report --out $(OUT)

.PHONY: rewrite-cache-failed-test
rewrite-cache-failed-test: ## Check frozen rewrite experiment evidence
	@pnpm rewrite:cache-failed:test

.PHONY: rewrite-cache-failed-verify
rewrite-cache-failed-verify: ## Check frozen rewrite experiment evidence
	@pnpm rewrite:cache-failed:verify

.PHONY: rewrite-cache-parity-report
rewrite-cache-parity-report: ## Replay frozen evidence; OUT required
	@pnpm rewrite:cache-parity:report --out $(OUT)

.PHONY: rewrite-cache-parity-test
rewrite-cache-parity-test: ## Check frozen rewrite experiment evidence
	@pnpm rewrite:cache-parity:test

.PHONY: rewrite-cache-parity-verify
rewrite-cache-parity-verify: ## Check frozen rewrite experiment evidence
	@pnpm rewrite:cache-parity:verify

.PHONY: rewrite-2b-contrast-report
rewrite-2b-contrast-report: ## Check or replay frozen rewrite evidence
	pnpm rewrite:2b-contrast:report --out "$(OUT)"

.PHONY: rewrite-2b-contrast-test
rewrite-2b-contrast-test: ## Check or replay frozen rewrite evidence
	pnpm rewrite:2b-contrast:test

.PHONY: rewrite-2b-contrast-verify
rewrite-2b-contrast-verify: ## Check or replay frozen rewrite evidence
	pnpm rewrite:2b-contrast:verify

.PHONY: rewrite-pronunciation-report
rewrite-pronunciation-report: ## Check or replay frozen rewrite evidence
	pnpm rewrite:pronunciation:report --out "$(OUT)"

.PHONY: rewrite-pronunciation-test
rewrite-pronunciation-test: ## Check or replay frozen rewrite evidence
	pnpm rewrite:pronunciation:test

.PHONY: rewrite-pronunciation-verify
rewrite-pronunciation-verify: ## Check or replay frozen rewrite evidence
	pnpm rewrite:pronunciation:verify

.PHONY: rewrite-source-plan-report
rewrite-source-plan-report: ## Check or replay frozen source-plan evidence
	pnpm rewrite:source-plan:report --out "$(OUT)"

.PHONY: rewrite-source-plan-test
rewrite-source-plan-test: ## Check or replay frozen source-plan evidence
	pnpm rewrite:source-plan:test

.PHONY: rewrite-source-plan-verify
rewrite-source-plan-verify: ## Check or replay frozen source-plan evidence
	pnpm rewrite:source-plan:verify

.PHONY: rewrite-local-protection-report
rewrite-local-protection-report: ## Check or replay frozen local-protection evidence
	pnpm rewrite:local-protection:report --out "$(OUT)"

.PHONY: rewrite-local-protection-test
rewrite-local-protection-test: ## Check or replay frozen local-protection evidence
	pnpm rewrite:local-protection:test

.PHONY: rewrite-local-protection-verify
rewrite-local-protection-verify: ## Check or replay frozen local-protection evidence
	pnpm rewrite:local-protection:verify

.PHONY: rewrite-protection-stress-report
rewrite-protection-stress-report: ## Check or replay frozen local-protection evidence
	pnpm rewrite:protection-stress:report --out "$(OUT)"

.PHONY: rewrite-protection-stress-test
rewrite-protection-stress-test: ## Check or replay frozen local-protection evidence
	pnpm rewrite:protection-stress:test

.PHONY: rewrite-protection-stress-verify
rewrite-protection-stress-verify: ## Check or replay frozen local-protection evidence
	pnpm rewrite:protection-stress:verify

.PHONY: rewrite-natural-sources-verify
rewrite-natural-sources-verify: ## Check existing natural audio and frozen ASR source lineage
	pnpm rewrite:natural-sources:verify

.PHONY: rewrite-natural-sources-test
rewrite-natural-sources-test: ## Check existing natural audio and frozen ASR source lineage
	pnpm rewrite:natural-sources:test

.PHONY: rewrite-timing-audit-verify
rewrite-timing-audit-verify: ## Recompute fixed native/external timing consistency snapshot
	pnpm rewrite:timing-audit:verify

.PHONY: rewrite-current-task-report
rewrite-current-task-report:
	pnpm rewrite:current-task:report $(ARGS)

.PHONY: rewrite-current-task-test
rewrite-current-task-test:
	pnpm rewrite:current-task:test $(ARGS)

.PHONY: rewrite-current-task-verify
rewrite-current-task-verify:
	pnpm rewrite:current-task:verify $(ARGS)

.PHONY: rewrite-system-probe-report
rewrite-system-probe-report:
	pnpm rewrite:system-probe:report $(ARGS)

.PHONY: rewrite-system-probe-test
rewrite-system-probe-test:
	pnpm rewrite:system-probe:test $(ARGS)

.PHONY: rewrite-system-probe-verify
rewrite-system-probe-verify:
	pnpm rewrite:system-probe:verify $(ARGS)

.PHONY: rewrite-cleanup-preflight-report
rewrite-cleanup-preflight-report:
	pnpm rewrite:cleanup-preflight:report $(ARGS)

.PHONY: rewrite-cleanup-preflight-verify
rewrite-cleanup-preflight-verify:
	pnpm rewrite:cleanup-preflight:verify $(ARGS)

.PHONY: rewrite-natural-text-verify
rewrite-natural-text-verify:
	pnpm rewrite:natural-text:verify $(ARGS)

.PHONY: rewrite-private-spans-report
rewrite-private-spans-report:
	pnpm rewrite:private-spans:report $(ARGS)

.PHONY: rewrite-private-spans-test
rewrite-private-spans-test:
	pnpm rewrite:private-spans:test $(ARGS)

.PHONY: rewrite-private-spans-verify
rewrite-private-spans-verify:
	pnpm rewrite:private-spans:verify $(ARGS)

.PHONY: rewrite-data-object-report
rewrite-data-object-report:
	pnpm rewrite:data-object:report $(ARGS)

.PHONY: rewrite-data-object-test
rewrite-data-object-test:
	pnpm rewrite:data-object:test $(ARGS)

.PHONY: rewrite-data-object-verify
rewrite-data-object-verify:
	pnpm rewrite:data-object:verify $(ARGS)

.PHONY: rewrite-2b-precision-report
rewrite-2b-precision-report:
	pnpm rewrite:2b-precision:report $(ARGS)

.PHONY: rewrite-2b-precision-test
rewrite-2b-precision-test:
	pnpm rewrite:2b-precision:test $(ARGS)

.PHONY: rewrite-2b-precision-verify
rewrite-2b-precision-verify:
	pnpm rewrite:2b-precision:verify $(ARGS)

.PHONY: rewrite-bounded-run
rewrite-bounded-run:
	pnpm rewrite:bounded:run $(ARGS)

.PHONY: rewrite-bounded-report
rewrite-bounded-report:
	pnpm rewrite:bounded:report $(ARGS)

.PHONY: rewrite-bounded-test
rewrite-bounded-test:
	pnpm rewrite:bounded:test $(ARGS)

.PHONY: rewrite-bounded-verify
rewrite-bounded-verify:
	pnpm rewrite:bounded:verify $(ARGS)

.PHONY: rewrite-bounded-v2-run
rewrite-bounded-v2-run:
	pnpm rewrite:bounded-v2:run $(ARGS)

.PHONY: rewrite-bounded-v2-report
rewrite-bounded-v2-report:
	pnpm rewrite:bounded-v2:report $(ARGS)

.PHONY: rewrite-bounded-v2-test
rewrite-bounded-v2-test:
	pnpm rewrite:bounded-v2:test $(ARGS)

.PHONY: rewrite-bounded-v2-verify
rewrite-bounded-v2-verify:
	pnpm rewrite:bounded-v2:verify $(ARGS)

.PHONY: rewrite-formatter-v3-run
rewrite-formatter-v3-run:
	pnpm rewrite:formatter-v3:run $(ARGS)

.PHONY: rewrite-formatter-v3-report
rewrite-formatter-v3-report:
	pnpm rewrite:formatter-v3:report $(ARGS)

.PHONY: rewrite-formatter-v3-test
rewrite-formatter-v3-test:
	pnpm rewrite:formatter-v3:test $(ARGS)

.PHONY: rewrite-formatter-v3-verify
rewrite-formatter-v3-verify:
	pnpm rewrite:formatter-v3:verify $(ARGS)

.PHONY: rewrite-metal-probe-report
rewrite-metal-probe-report:
	pnpm rewrite:metal-probe:report $(ARGS)

.PHONY: rewrite-metal-probe-verify
rewrite-metal-probe-verify:
	pnpm rewrite:metal-probe:verify $(ARGS)

.PHONY: rewrite-metal-regression-v1-run
rewrite-metal-regression-v1-run:
	pnpm rewrite:metal-regression-v1:run $(ARGS)

.PHONY: rewrite-metal-regression-v1-report
rewrite-metal-regression-v1-report:
	pnpm rewrite:metal-regression-v1:report $(ARGS)

.PHONY: rewrite-metal-regression-v1-test
rewrite-metal-regression-v1-test:
	pnpm rewrite:metal-regression-v1:test $(ARGS)

.PHONY: rewrite-metal-regression-v1-verify
rewrite-metal-regression-v1-verify:
	pnpm rewrite:metal-regression-v1:verify $(ARGS)

.PHONY: rewrite-natural-format-v1-run
rewrite-natural-format-v1-run:
	pnpm rewrite:natural-format-v1:run $(ARGS)

.PHONY: rewrite-natural-format-v1-report
rewrite-natural-format-v1-report:
	pnpm rewrite:natural-format-v1:report $(ARGS)

.PHONY: rewrite-natural-format-v1-test
rewrite-natural-format-v1-test:
	pnpm rewrite:natural-format-v1:test $(ARGS)

.PHONY: rewrite-natural-format-v1-verify
rewrite-natural-format-v1-verify:
	pnpm rewrite:natural-format-v1:verify $(ARGS)

.PHONY: rewrite-language-profile-v1-run
rewrite-language-profile-v1-run:
	pnpm rewrite:language-profile-v1:run $(ARGS)

.PHONY: rewrite-language-profile-v1-report
rewrite-language-profile-v1-report:
	pnpm rewrite:language-profile-v1:report $(ARGS)

.PHONY: rewrite-language-profile-v1-test
rewrite-language-profile-v1-test:
	pnpm rewrite:language-profile-v1:test $(ARGS)

.PHONY: rewrite-language-profile-v1-verify
rewrite-language-profile-v1-verify:
	pnpm rewrite:language-profile-v1:verify $(ARGS)

.PHONY: rewrite-language-profile-v2-run
rewrite-language-profile-v2-run:
	pnpm rewrite:language-profile-v2:run $(ARGS)

.PHONY: rewrite-language-profile-v2-report
rewrite-language-profile-v2-report:
	pnpm rewrite:language-profile-v2:report $(ARGS)

.PHONY: rewrite-language-profile-v2-test
rewrite-language-profile-v2-test:
	pnpm rewrite:language-profile-v2:test $(ARGS)

.PHONY: rewrite-language-profile-v2-verify
rewrite-language-profile-v2-verify:
	pnpm rewrite:language-profile-v2:verify $(ARGS)

.PHONY: rewrite-metal-chain-v1-run
rewrite-metal-chain-v1-run:
	pnpm rewrite:metal-chain-v1:run $(ARGS)

.PHONY: rewrite-metal-chain-v1-report
rewrite-metal-chain-v1-report:
	pnpm rewrite:metal-chain-v1:report $(ARGS)

.PHONY: rewrite-metal-chain-v1-test
rewrite-metal-chain-v1-test:
	pnpm rewrite:metal-chain-v1:test $(ARGS)

.PHONY: rewrite-metal-chain-v1-verify
rewrite-metal-chain-v1-verify:
	pnpm rewrite:metal-chain-v1:verify $(ARGS)

.PHONY: rewrite-metal-prefix-v1-run
rewrite-metal-prefix-v1-run:
	pnpm rewrite:metal-prefix-v1:run $(ARGS)

.PHONY: rewrite-metal-prefix-v1-report
rewrite-metal-prefix-v1-report:
	pnpm rewrite:metal-prefix-v1:report $(ARGS)

.PHONY: rewrite-metal-prefix-v1-test
rewrite-metal-prefix-v1-test:
	pnpm rewrite:metal-prefix-v1:test $(ARGS)

.PHONY: rewrite-metal-prefix-v1-verify
rewrite-metal-prefix-v1-verify:
	pnpm rewrite:metal-prefix-v1:verify $(ARGS)

.PHONY: rewrite-metal-primed-v1-run
rewrite-metal-primed-v1-run:
	pnpm rewrite:metal-primed-v1:run $(ARGS)

.PHONY: rewrite-metal-primed-v1-report
rewrite-metal-primed-v1-report:
	pnpm rewrite:metal-primed-v1:report $(ARGS)

.PHONY: rewrite-metal-primed-v1-test
rewrite-metal-primed-v1-test:
	pnpm rewrite:metal-primed-v1:test $(ARGS)

.PHONY: rewrite-metal-primed-v1-verify
rewrite-metal-primed-v1-verify:
	pnpm rewrite:metal-primed-v1:verify $(ARGS)

.PHONY: rewrite-history-chain-v1-run
rewrite-history-chain-v1-run:
	pnpm rewrite:history-chain-v1:run $(ARGS)

.PHONY: rewrite-history-chain-v1-report
rewrite-history-chain-v1-report:
	pnpm rewrite:history-chain-v1:report $(ARGS)

.PHONY: rewrite-history-chain-v1-test
rewrite-history-chain-v1-test:
	pnpm rewrite:history-chain-v1:test $(ARGS)

.PHONY: rewrite-history-chain-v1-verify
rewrite-history-chain-v1-verify:
	pnpm rewrite:history-chain-v1:verify $(ARGS)

.PHONY: rewrite-history-format-v2-run
rewrite-history-format-v2-run:
	pnpm rewrite:history-format-v2:run $(ARGS)

.PHONY: rewrite-history-format-v2-report
rewrite-history-format-v2-report:
	pnpm rewrite:history-format-v2:report $(ARGS)

.PHONY: rewrite-history-format-v2-test
rewrite-history-format-v2-test:
	pnpm rewrite:history-format-v2:test $(ARGS)

.PHONY: rewrite-history-format-v2-verify
rewrite-history-format-v2-verify:
	pnpm rewrite:history-format-v2:verify $(ARGS)

.PHONY: rewrite-work-scope-v3-run
rewrite-work-scope-v3-run:
	pnpm rewrite:work-scope-v3:run $(ARGS)

.PHONY: rewrite-work-scope-v3-report
rewrite-work-scope-v3-report:
	pnpm rewrite:work-scope-v3:report $(ARGS)

.PHONY: rewrite-work-scope-v3-test
rewrite-work-scope-v3-test:
	pnpm rewrite:work-scope-v3:test $(ARGS)

.PHONY: rewrite-work-scope-v3-verify
rewrite-work-scope-v3-verify:
	pnpm rewrite:work-scope-v3:verify $(ARGS)

.PHONY: rewrite-work-scope-v4-run
rewrite-work-scope-v4-run:
	pnpm rewrite:work-scope-v4:run $(ARGS)

.PHONY: rewrite-work-scope-v4-report
rewrite-work-scope-v4-report:
	pnpm rewrite:work-scope-v4:report $(ARGS)

.PHONY: rewrite-work-scope-v4-test
rewrite-work-scope-v4-test:
	pnpm rewrite:work-scope-v4:test $(ARGS)

.PHONY: rewrite-work-scope-v4-verify
rewrite-work-scope-v4-verify:
	pnpm rewrite:work-scope-v4:verify $(ARGS)

.PHONY: rewrite-tone-chain-v1-run
rewrite-tone-chain-v1-run:
	pnpm rewrite:tone-chain-v1:run $(ARGS)

.PHONY: rewrite-tone-chain-v1-report
rewrite-tone-chain-v1-report:
	pnpm rewrite:tone-chain-v1:report $(ARGS)

.PHONY: rewrite-tone-chain-v1-test
rewrite-tone-chain-v1-test:
	pnpm rewrite:tone-chain-v1:test $(ARGS)

.PHONY: rewrite-tone-chain-v1-verify
rewrite-tone-chain-v1-verify:
	pnpm rewrite:tone-chain-v1:verify $(ARGS)

.PHONY: rewrite-selector-semantic-v1-run
rewrite-selector-semantic-v1-run:
	@pnpm rewrite:selector-semantic-v1:run $(ARGS)

.PHONY: rewrite-selector-semantic-v1-report
rewrite-selector-semantic-v1-report:
	@pnpm rewrite:selector-semantic-v1:report $(ARGS)

.PHONY: rewrite-selector-semantic-v1-test
rewrite-selector-semantic-v1-test:
	@pnpm rewrite:selector-semantic-v1:test $(ARGS)

.PHONY: rewrite-selector-semantic-v1-verify
rewrite-selector-semantic-v1-verify:
	@pnpm rewrite:selector-semantic-v1:verify $(ARGS)

.PHONY: rewrite-selector-semantic-v2-run
rewrite-selector-semantic-v2-run:
	@pnpm rewrite:selector-semantic-v2:run $(ARGS)

.PHONY: rewrite-selector-semantic-v2-report
rewrite-selector-semantic-v2-report:
	@pnpm rewrite:selector-semantic-v2:report $(ARGS)

.PHONY: rewrite-selector-semantic-v2-test
rewrite-selector-semantic-v2-test:
	@pnpm rewrite:selector-semantic-v2:test $(ARGS)

.PHONY: rewrite-selector-semantic-v2-verify
rewrite-selector-semantic-v2-verify:
	@pnpm rewrite:selector-semantic-v2:verify $(ARGS)

.PHONY: rewrite-selector-preview-v1-run
rewrite-selector-preview-v1-run:
	@pnpm rewrite:selector-preview-v1:run $(ARGS)

.PHONY: rewrite-selector-preview-v1-report
rewrite-selector-preview-v1-report:
	@pnpm rewrite:selector-preview-v1:report $(ARGS)

.PHONY: rewrite-selector-preview-v1-test
rewrite-selector-preview-v1-test:
	@pnpm rewrite:selector-preview-v1:test $(ARGS)

.PHONY: rewrite-selector-preview-v1-verify
rewrite-selector-preview-v1-verify:
	@pnpm rewrite:selector-preview-v1:verify $(ARGS)

.PHONY: rewrite-selector-preview-v1-payload-test
rewrite-selector-preview-v1-payload-test:
	@pnpm rewrite:selector-preview-v1:payload-test $(ARGS)

.PHONY: rewrite-selector-screen-v1-run
rewrite-selector-screen-v1-run:
	@pnpm rewrite:selector-screen-v1:run $(ARGS)

.PHONY: rewrite-selector-screen-v1-report
rewrite-selector-screen-v1-report:
	@pnpm rewrite:selector-screen-v1:report $(ARGS)

.PHONY: rewrite-selector-screen-v1-test
rewrite-selector-screen-v1-test:
	@pnpm rewrite:selector-screen-v1:test $(ARGS)

.PHONY: rewrite-selector-screen-v1-verify
rewrite-selector-screen-v1-verify:
	@pnpm rewrite:selector-screen-v1:verify $(ARGS)

.PHONY: rewrite-selector-model-v1-run
rewrite-selector-model-v1-run:
	pnpm rewrite:selector-model-v1:run $(ARGS)

.PHONY: rewrite-selector-model-v1-report
rewrite-selector-model-v1-report:
	pnpm rewrite:selector-model-v1:report $(ARGS)

.PHONY: rewrite-selector-model-v1-test
rewrite-selector-model-v1-test:
	pnpm rewrite:selector-model-v1:test $(ARGS)

.PHONY: rewrite-selector-model-v1-verify
rewrite-selector-model-v1-verify:
	pnpm rewrite:selector-model-v1:verify $(ARGS)

.PHONY: rewrite-selector-model-cpu-v1-run
rewrite-selector-model-cpu-v1-run:
	pnpm rewrite:selector-model-cpu-v1:run $(ARGS)

.PHONY: rewrite-selector-model-cpu-v1-report
rewrite-selector-model-cpu-v1-report:
	pnpm rewrite:selector-model-cpu-v1:report $(ARGS)

.PHONY: rewrite-selector-model-cpu-v1-test
rewrite-selector-model-cpu-v1-test:
	pnpm rewrite:selector-model-cpu-v1:test $(ARGS)

.PHONY: rewrite-selector-model-cpu-v1-verify
rewrite-selector-model-cpu-v1-verify:
	pnpm rewrite:selector-model-cpu-v1:verify $(ARGS)

.PHONY: rewrite-selector-referent-v1-run
rewrite-selector-referent-v1-run:
	@pnpm rewrite:selector-referent-v1:run $(ARGS)

.PHONY: rewrite-selector-referent-v1-report
rewrite-selector-referent-v1-report:
	@pnpm rewrite:selector-referent-v1:report $(ARGS)

.PHONY: rewrite-selector-referent-v1-test
rewrite-selector-referent-v1-test:
	@pnpm rewrite:selector-referent-v1:test $(ARGS)

.PHONY: rewrite-selector-referent-v1-payload-test
rewrite-selector-referent-v1-payload-test:
	@pnpm rewrite:selector-referent-v1:payload-test $(ARGS)

.PHONY: rewrite-selector-referent-v1-verify
rewrite-selector-referent-v1-verify:
	@pnpm rewrite:selector-referent-v1:verify $(ARGS)

.PHONY: rewrite-selector-source-last-v1-run
rewrite-selector-source-last-v1-run:
	@pnpm rewrite:selector-source-last-v1:run $(ARGS)

.PHONY: rewrite-selector-source-last-v1-report
rewrite-selector-source-last-v1-report:
	@pnpm rewrite:selector-source-last-v1:report $(ARGS)

.PHONY: rewrite-selector-source-last-v1-test
rewrite-selector-source-last-v1-test:
	@pnpm rewrite:selector-source-last-v1:test $(ARGS)

.PHONY: rewrite-selector-source-last-v1-payload-test
rewrite-selector-source-last-v1-payload-test:
	@pnpm rewrite:selector-source-last-v1:payload-test $(ARGS)

.PHONY: rewrite-selector-source-last-v1-verify
rewrite-selector-source-last-v1-verify:
	@pnpm rewrite:selector-source-last-v1:verify $(ARGS)

.PHONY: rewrite-selector-sentence-choice-v1-run
rewrite-selector-sentence-choice-v1-run:
	@pnpm rewrite:selector-sentence-choice-v1:run $(ARGS)

.PHONY: rewrite-selector-sentence-choice-v1-report
rewrite-selector-sentence-choice-v1-report:
	@pnpm rewrite:selector-sentence-choice-v1:report $(ARGS)

.PHONY: rewrite-selector-sentence-choice-v1-test
rewrite-selector-sentence-choice-v1-test:
	@pnpm rewrite:selector-sentence-choice-v1:test $(ARGS)

.PHONY: rewrite-selector-sentence-choice-v1-payload-test
rewrite-selector-sentence-choice-v1-payload-test:
	@pnpm rewrite:selector-sentence-choice-v1:payload-test $(ARGS)

.PHONY: rewrite-selector-sentence-choice-v1-verify
rewrite-selector-sentence-choice-v1-verify:
	@pnpm rewrite:selector-sentence-choice-v1:verify $(ARGS)

.PHONY: rewrite-selector-hybrid-v1-preflight
rewrite-selector-hybrid-v1-preflight:
	@pnpm rewrite:selector-hybrid-v1:preflight $(ARGS)

.PHONY: rewrite-selector-hybrid-v1-run
rewrite-selector-hybrid-v1-run:
	@pnpm rewrite:selector-hybrid-v1:run $(ARGS)

.PHONY: rewrite-selector-hybrid-v1-report
rewrite-selector-hybrid-v1-report:
	@pnpm rewrite:selector-hybrid-v1:report $(ARGS)

.PHONY: rewrite-selector-hybrid-v1-test
rewrite-selector-hybrid-v1-test:
	@pnpm rewrite:selector-hybrid-v1:test $(ARGS)

.PHONY: rewrite-selector-hybrid-v1-verify
rewrite-selector-hybrid-v1-verify:
	@pnpm rewrite:selector-hybrid-v1:verify $(ARGS)

.PHONY: rewrite-selector-instruction-language-v1-run
rewrite-selector-instruction-language-v1-run:
	@pnpm rewrite:selector-instruction-language-v1:run $(ARGS)

.PHONY: rewrite-selector-instruction-language-v1-report
rewrite-selector-instruction-language-v1-report:
	@pnpm rewrite:selector-instruction-language-v1:report $(ARGS)

.PHONY: rewrite-selector-instruction-language-v1-test
rewrite-selector-instruction-language-v1-test:
	@pnpm rewrite:selector-instruction-language-v1:test $(ARGS)

.PHONY: rewrite-selector-instruction-language-v1-payload-test
rewrite-selector-instruction-language-v1-payload-test:
	@pnpm rewrite:selector-instruction-language-v1:payload-test $(ARGS)

.PHONY: rewrite-selector-instruction-language-v1-verify
rewrite-selector-instruction-language-v1-verify:
	@pnpm rewrite:selector-instruction-language-v1:verify $(ARGS)

.PHONY: rewrite-selector-reference-turn-v1-run
rewrite-selector-reference-turn-v1-run:
	@pnpm rewrite:selector-reference-turn-v1:run $(ARGS)

.PHONY: rewrite-selector-reference-turn-v1-report
rewrite-selector-reference-turn-v1-report:
	@pnpm rewrite:selector-reference-turn-v1:report $(ARGS)

.PHONY: rewrite-selector-reference-turn-v1-test
rewrite-selector-reference-turn-v1-test:
	@pnpm rewrite:selector-reference-turn-v1:test $(ARGS)

.PHONY: rewrite-selector-reference-turn-v1-payload-test
rewrite-selector-reference-turn-v1-payload-test:
	@pnpm rewrite:selector-reference-turn-v1:payload-test $(ARGS)

.PHONY: rewrite-selector-reference-turn-v1-verify
rewrite-selector-reference-turn-v1-verify:
	@pnpm rewrite:selector-reference-turn-v1:verify $(ARGS)

.PHONY: rewrite-selector-unique-label-v1-run
rewrite-selector-unique-label-v1-run:
	@pnpm rewrite:selector-unique-label-v1:run $(ARGS)

.PHONY: rewrite-selector-unique-label-v1-report
rewrite-selector-unique-label-v1-report:
	@pnpm rewrite:selector-unique-label-v1:report $(ARGS)

.PHONY: rewrite-selector-unique-label-v1-test
rewrite-selector-unique-label-v1-test:
	@pnpm rewrite:selector-unique-label-v1:test $(ARGS)

.PHONY: rewrite-selector-unique-label-v1-payload-test
rewrite-selector-unique-label-v1-payload-test:
	@pnpm rewrite:selector-unique-label-v1:payload-test $(ARGS)

.PHONY: rewrite-selector-unique-label-v1-verify
rewrite-selector-unique-label-v1-verify:
	@pnpm rewrite:selector-unique-label-v1:verify $(ARGS)

.PHONY: rewrite-selector-contrast-demo-v1-run
rewrite-selector-contrast-demo-v1-run:
	@pnpm rewrite:selector-contrast-demo-v1:run $(ARGS)

.PHONY: rewrite-selector-contrast-demo-v1-report
rewrite-selector-contrast-demo-v1-report:
	@pnpm rewrite:selector-contrast-demo-v1:report $(ARGS)

.PHONY: rewrite-selector-contrast-demo-v1-test
rewrite-selector-contrast-demo-v1-test:
	@pnpm rewrite:selector-contrast-demo-v1:test $(ARGS)

.PHONY: rewrite-selector-contrast-demo-v1-payload-test
rewrite-selector-contrast-demo-v1-payload-test:
	@pnpm rewrite:selector-contrast-demo-v1:payload-test $(ARGS)

.PHONY: rewrite-selector-contrast-demo-v1-verify
rewrite-selector-contrast-demo-v1-verify:
	@pnpm rewrite:selector-contrast-demo-v1:verify $(ARGS)

.PHONY: rewrite-selector-minicpm-v1-run
rewrite-selector-minicpm-v1-run:
	@pnpm rewrite:selector-minicpm-v1:run $(ARGS)

.PHONY: rewrite-selector-minicpm-v1-report
rewrite-selector-minicpm-v1-report:
	@pnpm rewrite:selector-minicpm-v1:report $(ARGS)

.PHONY: rewrite-selector-minicpm-v1-test
rewrite-selector-minicpm-v1-test:
	@pnpm rewrite:selector-minicpm-v1:test $(ARGS)

.PHONY: rewrite-selector-minicpm-v1-verify
rewrite-selector-minicpm-v1-verify:
	@pnpm rewrite:selector-minicpm-v1:verify $(ARGS)

.PHONY: rewrite-selector-minicpm-f16-v1-run
rewrite-selector-minicpm-f16-v1-run:
	@pnpm rewrite:selector-minicpm-f16-v1:run $(ARGS)

.PHONY: rewrite-selector-minicpm-f16-v1-report
rewrite-selector-minicpm-f16-v1-report:
	@pnpm rewrite:selector-minicpm-f16-v1:report $(ARGS)

.PHONY: rewrite-selector-minicpm-f16-v1-test
rewrite-selector-minicpm-f16-v1-test:
	@pnpm rewrite:selector-minicpm-f16-v1:test $(ARGS)

.PHONY: rewrite-selector-minicpm-f16-v1-verify
rewrite-selector-minicpm-f16-v1-verify:
	@pnpm rewrite:selector-minicpm-f16-v1:verify $(ARGS)

.PHONY: rewrite-selector-youtu-v1-run
rewrite-selector-youtu-v1-run:
	@pnpm rewrite:selector-youtu-v1:run $(ARGS)

.PHONY: rewrite-selector-youtu-v1-report
rewrite-selector-youtu-v1-report:
	@pnpm rewrite:selector-youtu-v1:report $(ARGS)

.PHONY: rewrite-selector-youtu-v1-test
rewrite-selector-youtu-v1-test:
	@pnpm rewrite:selector-youtu-v1:test $(ARGS)

.PHONY: rewrite-selector-youtu-v1-verify
rewrite-selector-youtu-v1-verify:
	@pnpm rewrite:selector-youtu-v1:verify $(ARGS)

.PHONY: rewrite-selector-lfm25-v1-run
rewrite-selector-lfm25-v1-run:
	@pnpm rewrite:selector-lfm25-v1:run $(ARGS)

.PHONY: rewrite-selector-lfm25-v1-report
rewrite-selector-lfm25-v1-report:
	@pnpm rewrite:selector-lfm25-v1:report $(ARGS)

.PHONY: rewrite-selector-lfm25-v1-test
rewrite-selector-lfm25-v1-test:
	@pnpm rewrite:selector-lfm25-v1:test $(ARGS)

.PHONY: rewrite-selector-lfm25-v1-verify
rewrite-selector-lfm25-v1-verify:
	@pnpm rewrite:selector-lfm25-v1:verify $(ARGS)

.PHONY: rewrite-selector-demo-order-v1-run
rewrite-selector-demo-order-v1-run:
	@pnpm rewrite:selector-demo-order-v1:run $(ARGS)

.PHONY: rewrite-selector-demo-order-v1-report
rewrite-selector-demo-order-v1-report:
	@pnpm rewrite:selector-demo-order-v1:report $(ARGS)

.PHONY: rewrite-selector-demo-order-v1-test
rewrite-selector-demo-order-v1-test:
	@pnpm rewrite:selector-demo-order-v1:test $(ARGS)

.PHONY: rewrite-selector-demo-order-v1-payload-test
rewrite-selector-demo-order-v1-payload-test:
	@pnpm rewrite:selector-demo-order-v1:payload-test $(ARGS)

.PHONY: rewrite-selector-demo-order-v1-verify
rewrite-selector-demo-order-v1-verify:
	@pnpm rewrite:selector-demo-order-v1:verify $(ARGS)

.PHONY: rewrite-d1-evidence-v1-report
rewrite-d1-evidence-v1-report:
	@pnpm rewrite:d1-evidence-v1:report $(ARGS)

.PHONY: rewrite-d1-evidence-v1-test
rewrite-d1-evidence-v1-test:
	@pnpm rewrite:d1-evidence-v1:test $(ARGS)

.PHONY: rewrite-d1-evidence-v1-verify
rewrite-d1-evidence-v1-verify:
	@pnpm rewrite:d1-evidence-v1:verify $(ARGS)

.PHONY: updater-check
updater-check: ## Verify the maintained MyGo update-channel patch
	@pnpm updater:check

.PHONY: rewrite-stages-run rewrite-stages-matrix rewrite-stages-report rewrite-stages-test rewrite-stages-verify
rewrite-stages-run:
	@pnpm rewrite:stages:run $(ARGS)
rewrite-stages-matrix:
	@pnpm rewrite:stages:matrix $(ARGS)
rewrite-stages-report:
	@pnpm rewrite:stages:report $(ARGS)
rewrite-stages-test:
	@pnpm rewrite:stages:test
rewrite-stages-verify:
	@pnpm rewrite:stages:verify $(ARGS)

.PHONY: vocabulary-bench-run vocabulary-bench-report vocabulary-bench-verify vocabulary-bench-test
vocabulary-bench-run: ## Observe production vocabulary correction and recall on controlled fixtures
	@pnpm vocabulary:bench:run $(ARGS)
vocabulary-bench-report: ## Replay a completed vocabulary benchmark without inference
	@pnpm vocabulary:bench:report $(ARGS)
vocabulary-bench-verify: ## Verify vocabulary benchmark hashes and deterministic report replay
	@pnpm vocabulary:bench:verify $(ARGS)
vocabulary-bench-test:
	@pnpm vocabulary:bench:test

.PHONY: qwen17-context-report qwen17-context-test
qwen17-context-report:
	pnpm qwen17:context:report --run "$(RESULTS)" --out "$(OUT)"
qwen17-context-test:
	pnpm qwen17:context:test

.PHONY: qwen17-verify
qwen17-verify:
	pnpm qwen17:verify

.PHONY: native-asr-verify
native-asr-verify: ## Verify the native ASR acceptance archive without models
	pnpm asr:native:verify

.PHONY: ime-rerank
ime-rerank: ## Reproduce or replay the isolated Rime LLM reranker study
	@pnpm ime:rerank $(ARGS)

.PHONY: ime-align
ime-align: ## Run or replay general LLM keyboard-ranking alignment
	pnpm ime:align $(ARGS)

.PHONY: updater-e2e notifications-check notifications-prepare
updater-e2e: ## Run the isolated native packaged updater lifecycle
	pnpm updater:e2e

notifications-check: ## Check notification APIs, registration and platform builds
	pnpm notifications:check

notifications-prepare: ## Build an isolated native notification consent probe
	pnpm notifications:prepare

.PHONY: asr-hotwords-report asr-hotwords-verify
asr-hotwords-report: ## Replay archived paired ASR hotword observations
	@pnpm asr:hotwords:report $(ARGS)
asr-hotwords-verify: ## Verify ASR hotword evidence and report replay
	@pnpm asr:hotwords:verify $(ARGS)
