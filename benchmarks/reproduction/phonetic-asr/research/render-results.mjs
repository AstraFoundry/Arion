import fs from 'node:fs/promises';
import path from 'node:path';
import { hashFile } from '../../../tools/evaluation/src/manifest.mjs';
const base = path.resolve('.workspace/benchmarks/phonetic-asr-v2');
const read = async name => JSON.parse(await fs.readFile(name, 'utf8'));
const summary = await read(path.join(base, 'summary.json'));
const supplemental = await read(path.join(base, 'supplemental.json'));
const confidenceFile = path.resolve('.workspace/benchmarks/phonetic-asr-v1/confidence.json');
const confidence = await read(confidenceFile);
if (summary.status !== 'complete' || supplemental.status !== 'complete' || supplemental.primaryReportHash !== await hashFile(path.join(base, 'summary.json')) || confidence.corpusHash !== summary.corpusHash) throw new Error('Verified complete aggregates required');
const out = path.resolve('benchmarks/results/phonetic-asr-1.7b-v2');
const percent = value => value === null ? 'n/a' : `${(value * 100).toFixed(2)}%`;
const seconds = value => Number.isFinite(value) ? (value / 1000).toFixed(2) : 'n/a';
const labels = { text: 'Text', text_pinyin: 'Text + text-derived pinyin', text_scores: 'Text + token scores', text_pinyin_scores: 'Text + pinyin + scores', text_ipa: 'Text + audio IPA', ipa_only: 'Audio IPA only' };
const rateCells = result => ['cer', 'wer', 'mer'].map(unit => percent(result.rates[unit].rate)).join(' | ');
const rows = [['Raw streaming ASR', summary.summaries['raw/raw'].overall], ['Offline acoustic control', supplemental.offlineControl.overall]];
for (const model of summary.config.models) for (const variant of summary.config.variants) rows.push([`${model} / ${labels[variant]}`, summary.summaries[`${model}/${variant}`].overall]);
const table = rows.map(([name, result]) => `| ${name} | ${rateCells(result)} | ${result.improved} / ${result.worsened} | ${result.corrupted} / ${result.originallyCorrect} | ${result.falseEmissions} / ${result.silenceCount} | ${result.truncated} |`).join('\n');
const latency = rows.slice(2).map(([name, result]) => `| ${name} | ${seconds(result.generationMs.p50)} | ${seconds(result.generationMs.p95)} | ${result.outputContractFailures} |`).join('\n');
const routed = Object.entries(supplemental.routing.treatments).map(([key, result]) => `| ${key} | ${rateCells(result.overall)} | ${result.overall.corrupted} / ${result.overall.originallyCorrect} |`).join('\n');
const detector = ['all', 'zh', 'en', 'mixed'].map(language => {
 const result = confidence.groups[`test/${language}`];
 return `| ${language} | ${result.samples} / ${result.missingScores} | ${result.auc === null ? 'n/a' : result.auc.toFixed(3)} | ${percent(result.precision)} | ${percent(result.recall)} | ${percent(result.falsePositiveRate)} |`;
}).join('\n');
const raw = summary.summaries['raw/raw'].overall;
const text = `# Phonetic evidence ablation: complete Qwen3-1.7B checkpoint

Status: completed exploratory engineering benchmark. All 154 test cases have raw streaming ASR, an offline acoustic control and all six correction treatments for Qwen3-1.7B: 924 measured generations, 154 raw controls, 12 development generation warmups and two separate offline-control warmups. The user cancelled Qwen3-4B; its two completed test generations and warmups are retained privately and excluded from this report. The original two-model execution remains marked cancelled_by_user. This report uses a hash-linked, coverage-verified view of the complete 1.7B checkpoint, without new inference. No product correction or desktop insertion behavior changed.

## Quality

Lower error rates are better. CER uses 60 natural Mandarin read-speech cases, WER 60 natural English read-speech cases, and MER 30 synthetic technical mixed-language cases. Four nonspeech cases are counted separately. Improved/worsened compare utterance edit counts with the same streaming ASR source. The originally-correct denominator includes correctly silent cases; these counts are normalized transcript proxies, without independent semantic review.

| Condition | Mandarin CER | English WER | Mixed MER | Improved / worsened vs raw | Originally correct corrupted | Silence emissions | Output-limit hits |
| --- | --- | --- | --- | --- | --- | --- | --- |
${table}

Raw denominators are ${raw.rates.cer.referenceUnits} Chinese characters, ${raw.rates.wer.referenceUnits} English words and ${raw.rates.mer.referenceUnits} mixed units. Insertions can make error rates exceed 100%; results are not clamped. No failed or truncated proposal is replaced with its source. The automatic output-contract count detects leading JSON, chat/reasoning markers and truncation; it is not a complete semantic validator. Supplemental literal-field diagnostics also count evidence markers emitted inside a response. Exact errors, denominators, strata, format failures and paired speaker-cluster bootstrap intervals are in [summary.json](summary.json). Offline-control paired comparisons are in [supplemental.json](supplemental.json).

## Confidence and selective correction

The fixed threshold is ${summary.threshold}, the development emitted-token lower quartile; no reference was used to fit it. Scores are conditional log-softmax values on the selected transducer path, not calibrated word-correctness probabilities. Risk is the negative minimum emitted-token score. The table evaluates error detection for spoken cases; missing scores are excluded from AUC and counted explicitly.

| Stratum | Scored / missing spoken cases | AUC | Precision | Recall | False-positive rate |
| --- | --- | --- | --- | --- | --- | --- |
${detector}

An utterance-level counterfactual invokes correction for any below-threshold emitted token or missing scores, and otherwise retains raw recognition. The policy invokes ${supplemental.routing.invoked}/${supplemental.routing.samples} cases; ${supplemental.routing.missingScores} have no scores. It does not inspect reference text or choose between treatments based on their outputs. This is an exploratory routing calculation over existing observations, not actual selective execution or uncertain-span phonetic decoding.

| Routed treatment | Mandarin CER | English WER | Mixed MER | Originally correct corrupted |
| --- | --- | --- | --- | --- |
${routed}

The detector is utterance-level. These results do not establish accurate word localization, calibrated confidence or a safe automatic replacement threshold. Language, length and rare correct tokens can affect its minimum-score heuristic. Complete detector counts are in [confidence.json](confidence.json).

## Timing

Hardware: ${summary.host.cpu}, ${(summary.host.ramBytes / 1024 ** 3).toFixed(0)} GiB RAM, ${summary.host.os}/${summary.host.architecture}; four CPU threads, warm model residency, fresh uncached prefill per request, greedy non-thinking INT4 ONNX generation with input/output ceilings of 4,096/256 tokens. The checkpoint has its own process; conditions are shuffled within it.

| Correction condition | Generation p50 (s) | Generation p95 (s) | Output-contract failures |
| --- | --- | --- | --- |
${latency}

Generation is engine prefill plus decode, without model load, real audio pacing, native UI, target insertion or cross-request prefix reuse. The frozen configuration described intended timing as isolated, but observed background workloads mean that condition was not achieved. Timings do not establish a controlled latency ranking. Acoustic component distributions in the primary aggregate cover all 272 development/test files, not only the 154 test files. The offline acoustic control has compute p50/p95 ${seconds(supplemental.offlineControl.computeMs.p50)}/${seconds(supplemental.offlineControl.computeMs.p95)} s over test cases; it is whole-file audio processing. Separate component percentiles must not be summed into an end-to-end percentile.

## Method and limits

- X-ASR streaming INT8, greedy decoding, 20 ms accelerated chunks, 0.8 s endpoint silence and 0.4 s tail padding provide the fixed raw recognition and real token log scores. A separately pinned offline X-ASR INT8 checkpoint supplies the acoustic-only second-pass comparator.
- Text-derived pinyin uses pypinyin 0.55.0 TONE3, neutral tone 5; English runs remain text. It provides a pronunciation constraint, without a new audio observation.
- Allosaurus 1.0.2 uni2005 recognizes a single universal IPA path directly from original audio, without ASR text, reference, language hint or vocabulary context. Its frontend internally uses 8 kHz MFCCs. This is not a Mandarin-tone-aware multitask model or verified phonetic ground truth.
- The six conditions share the conservative editor prompt and three generic text-copy demonstrations. IPA-only removes the current ASR transcript from that editor; its result does not evaluate a separately trained phoneme-to-grapheme decoder. No context/memory or real N-best lattice is supplied.
- The initial prompt systematically echoed evidence/JSON. That attempt was stopped and retained as failed, and its numbers are excluded from representation-efficacy conclusions. Actual newline bytes were correct. After a development-only 36-observation output-contract preflight, the revised frozen prompt reran the same test set. This is reused-test exploratory evidence, not an untouched confirmatory test.
- The corpus has 44 test speaker/voice clusters, including only four natural Mandarin speakers and four synthetic mixed-language voices. Read speech and synthetic technical fixtures do not establish spontaneous-dictation performance. Model exposure to upstream public benchmarks is not ruled out.
- Paired cluster bootstrap intervals are exploratory with unadjusted multiple comparisons. Automatic error rates and correct-source corruption do not establish intent, protected-name, number or negation preservation. No release-quality acceptance is claimed.

See the [full procedure](../../../docs/research/phonetic-asr-benchmark.md), [paper and competitor investigation](../../../docs/research/phonetic-asr-correction.md), [frozen configuration](../../configs/phonetic-asr.json) and [008 evaluation contract](../../../specs/008-evaluation.md).

## Evidence

Only content-free aggregates are checked in. Audio, exact prompts, recognition/phone outputs, generated proposals, token traces, warmups, artifact manifests and host-contention samples remain private under .workspace. [supplemental.json](supplemental.json) records completed run/result hashes, model and code hashes, runtime versions, preflight results and invalid-attempt provenance. The exact original complete primary report SHA-256 is ${supplemental.primaryReportHash}.
`;
await fs.mkdir(out);
for (const [source, destination] of [[path.join(base, 'summary.json'), 'summary.json'], [path.join(base, 'supplemental.json'), 'supplemental.json'], [confidenceFile, 'confidence.json']]) await fs.copyFile(source, path.join(out, destination), fs.constants.COPYFILE_EXCL);
await fs.writeFile(path.join(out, 'README.md'), text, { flag: 'wx' });
console.log(out);
