# Public product and private evaluation repositories

Arion uses two Git repositories. The public [Arion repository](https://github.com/AstraFoundry/Arion) contains the product and its regression tests. The private [Arion evaluation repository](https://github.com/AstraFoundry/Arion-evaluation) contains research tools, complete corpora, raw observations and historical evidence. Both repositories use Git for their tracked content. No object-storage service is required by this split.

[ADR 032](../decisions/032-public-product-private-evaluation.md) records this boundary. The normative evidence requirements remain in [008 Evaluation](../../specs/008-evaluation.md).

## Ownership

| Public Arion | Private Arion evaluation |
| --- | --- |
| Product Go modules, native hosts and site | Study configuration, scoring, orchestration and research command wrappers |
| Build and release scripts, runtime pins, dictionaries, fonts and notices | Full benchmark corpora, labels, raw results, logs and review records |
| Unit, integration and native regression tests with small licensed `testdata` | Historical experiment replay, exposed development sets and complete acceptance evidence |
| Product observation drivers and the shared [driver contract](../../tools/driver/README.md) | Evaluators that invoke the shared public driver implementation |
| Product specs, ADRs, operating instructions and scoped capability summaries | Complete study procedures and frozen reproduction sources |

Private research code depends on a selected public product revision. Public installation, checks and packaging do not depend on private repository access.

Small public regression inputs retain their source paths, hashes, licenses and provenance. They protect product behavior; they are not new held-out evaluation data. Moving an exposed test set does not make it unseen again.

## Local checkouts and agent entry point

Keep the repositories as sibling directories when practical:

```text
projects/
├── Arion/                  # Product development and its AGENTS.md
└── Arion-evaluation/       # Private studies and its AGENTS.md
```

Open Arion for ordinary agent-assisted development. Its `AGENTS.md` mirrors `CLAUDE.md` and remains the product entry point. A parent-level instruction file is not required. If the agent tool restricts access to the project directory, add the private checkout as a second workspace directory.

From an existing Arion checkout, clone the private repository beside it:

```sh
cd ..
git clone git@github.com:AstraFoundry/Arion-evaluation.git
```

Before crossing into the private checkout, read its `AGENTS.md` and `README.md` explicitly. Run commands from the repository that owns them and commit the two repositories separately. A different private checkout location is supported when its path is explicit.

The private archive already contains the previously committed benchmarks. Preserve local changes before updating the public checkout to main. Git removes the migrated tracked files during that update. Back up untracked or uncommitted local research separately before deciding how to archive it.

## Public development

Use the commands in [local development](../operations/local-dev.md). Ordinary product and driver checks use only the public checkout. Product tests stay beside the implementation they protect. Native IMK, TSF, signing and model checks keep their actual OS, credential and explicit provisioning requirements.

The public `tools/driver` package owns JSONL transport, artifact verification and owned product scenario execution. It does not select research corpora, score model quality or load historical benchmark archives. Private evaluators reuse its implementation instead of maintaining another live copy.

The public model catalog retains approved provenance and measured summaries. Private `scripts/export-catalog.mjs` validates a completed raw run and its hashes, then emits a content-free export with aggregate model evaluations and exact source identities. After review, public `node scripts/catalog-evaluation.mjs <reviewed-summary.json> --reviewed` consumes that JSON, writes `models/evaluations/<runId>.json` and updates catalog report links. The private exporter does not write the public catalog. Raw observations and study publication belong to the private repository.

## Internal execution

Follow the private repository's README for study commands and exact arguments. Its `product.lock.json` pins the public repository and full commit. `ARION_CODE_ROOT` selects an explicit product checkout; the documented product-checkout command can prepare the pinned revision under the private workspace. A missing or mismatched checkout fails explicitly.

The sibling product checkout is not selected automatically. For a new product revision, commit and push the product branch, update and commit the private product lock, and prepare the selected product with `pnpm --dir /path/to/Arion-evaluation product:prepare`. The default preparation fetches that revision into `.workspace/product`. Keep product development in Arion rather than editing this prepared checkout. Both repositories must be clean before a recorded study invocation.

The private workspace can create ignored links to the selected public product paths so historical reproduction code resolves its recorded layout. These links do not create a second maintained product implementation. Setup must validate the selected revision and must not overwrite private-owned files.

Each run records the actual product commit, evaluator commit, dirty state, effective configuration and corpus, runtime, model and artifact hashes. A successful result applies only to those identities. A new product revision does not inherit an earlier pass. Accepted evidence requires recoverable source bytes, including any recorded historical patches.

## Evidence references and integrity

An internal evidence reference such as `benchmarks/runs/<study>/README.md` is relative to the private repository root. Public documents link to the private repository and retain that logical path. It is not a local path in the public checkout. Public capability summaries state what the evidence established and what remains unverified.

Existing `bundle.json` and `evidence.tar.gz` pairs remain immutable Git files with their member identities. Preserve original bytes, path mappings, licenses, prompts, warmups and completed observations. Failed, cancelled and incomplete runs retain their original status. Do not unpack a historical bundle into another maintained source tree or reformat frozen evidence to repair links.

Archive verification and report-only replay run in the private repository. They do not establish new model inference, native system dispatch or release-signing acceptance. Personal recordings, credentials, model weights, runtime environments and unfinished execution remain outside automatic shared-archive migration.

## CI and repository history

Public CI runs portable and actual-platform product checks and development packaging. Public release and signing workflows retain their existing ownership and credential boundaries. Private archive checks, report replay and model studies run locally. Keep GitHub Actions disabled in Arion-evaluation and store no workflow files there. Public CI does not check out private data or execute private studies.

The initial split changes current repository trees. It does not rewrite shared Git history. Original source revisions, signed-release provenance and archive manifests remain distinguishable from later commits. Historical public references can still expose previously published content. Any later history filtering requires a separate verified backup, old-to-new commit map and decision covering every public branch and tag.
