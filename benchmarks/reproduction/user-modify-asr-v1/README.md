# Recognition-stage USER_MODIFY diagnostic

These scripts compare actual Qwen3-ASR-1.7B audio recognition with different memory inputs. They never invoke the text rewriter or apply vocabulary replacements. The observed Doubao policy and the explicitly unknown parameters are described in [research](../../../docs/research/user-modify-asr.md).

## Inputs and execution

`build-fixtures.py <new-directory>` creates the developer-authored memory, configuration, 32 test audio cases and a separate-voice development warmup. Exact generated WAV and intermediate AIFF bytes are archived under [the corpus directory](../../corpora/user-modify-asr-v1/); regeneration can differ with system voice versions. Do not overwrite the archived inputs or treat these disclosed synthetic cases as a future unseen test set.

The frozen model metadata files accompany the corpus. Provision `mlx-community/Qwen3-ASR-1.7B-8bit` at revision `a8379a2e2f9e313c9292cdf1af4055ab56d50d55` separately. The runner verifies downloaded LFS and configuration Git-blob hashes, including tokenizer merges, before inference. Weights and environments remain in `.workspace/`.

The experiment uses MLX Audio source commit `dad2087f790dfe96b07d5aac1405db4b578b411f`. Runtime versions and source/native artifact hashes are retained in each run; source-checkpoint BF16 parity is not established. This is an Apple Silicon development diagnostic, not a production runtime decision for Windows.

From the repository root, with an isolated, explicitly provisioned environment:

```sh
HF_HUB_OFFLINE=1 .workspace/user-modify-asr-venv/bin/python \
  benchmarks/reproduction/user-modify-asr-v1/run.py \
  --fixtures benchmarks/corpora/user-modify-asr-v1 \
  --model .workspace/ablation-models/qwen3-asr-1.7b-mlx8 \
  --out .workspace/benchmarks/user-modify-asr-new

node benchmarks/reproduction/user-modify-asr-v1/report.mjs \
  benchmarks/corpora/user-modify-asr-v1 \
  .workspace/benchmarks/user-modify-asr-new \
  .workspace/benchmarks/user-modify-asr-new/summary.json
```

Output paths must be new. `--preflight` runs only separate development warmups and records `preflight_complete`, which the report rejects as an incomplete experiment. A normal run warms every context condition, then randomizes arm order within each test case using a fixed seed. All arms use fresh generation state, identical audio, greedy decoding and the same token ceiling. References, term labels and case kinds are removed from the inference-loop input object; they are used only by the separate report.

`none`, `destinations`, `pairs` and `tagged_pairs` share the same recognition checkpoint. The latter three use identical selected entries, isolating destination-only versus pair serialization. `all_pairs` is an intentionally unfiltered control including expired and contradictory entries, so its contrast changes selection as well as memory size. No effect should be attributed to the literal slot label by comparing arms with different entries.

## Interpretation

The memory and its counts are authored prior-session fixtures, not observed corrections mined from a real user. Selection uses experimental quotas of two today, three by recall count and the remaining slots by use count, with eight total pairs, a 512-character pair-text budget and 30-day expiry. These quotas and serialization are not recovered competitor defaults. Reverse pairs are suppressed. Memory stays frozen throughout paired inference; the run does not measure online counter updates or a server extraction loop.

The scorer reuses the repository normalization, CER/WER/MER alignment and percentile implementation. It verifies input/output hashes, complete pairing and unique rows. It reports per-case outputs, term substring/boundary proxies, error increases against the no-memory arm, corruption of initially correct speech, nonspeech emissions, token-limit hits and compute latency/RTF. RSS/MLX peaks are cumulative process peaks, not isolated per-arm allocations. No confidence intervals or semantic accuracy claims are appropriate for this single-author, two-test-voice development set.
