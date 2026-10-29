# ADR 032: Public product and private evaluation repositories

## Status

Accepted.

## Context

Ordinary product checks and large research archives shared one repository. Some public regression tests read research paths directly, while experimental commands and raw records enlarged feature diffs. The team needs a public checkout that external contributors can build and verify, and a private shared repository that preserves complete research evidence.

## Decision

Keep one public product source of truth in `AstraFoundry/Arion`. Retain product code, native adapters, build inputs, licenses, normal regression tests, small provenance-bound fixtures and the shared `tools/driver` contract there.

Move complete benchmark corpora, research command wrappers, scoring, orchestration, original results, frozen reproduction sources and historical replay into private `AstraFoundry/Arion-evaluation`. Store those records and existing immutable evidence bundles in Git. The split requires no S3 service, object store or new storage credentials.

Private execution selects a public repository and exact commit through `product.lock.json` and an explicit checkout. Product code never imports the private repository. Private evaluators consume the public driver implementation, with one owner for transport and artifact validation. Ignored workspace links can preserve declared historical paths without duplicating maintained product source.

Preserve the original byte, hash, path-map, split-isolation and failure-status requirements of EVAL-A03 through EVAL-A08. Small fixtures extracted for public regression tests retain their provenance and existing exposure. Private runs record both product and evaluator identities; published summaries bind to the evaluated code and artifacts.

Public product, signing and release CI remains public. Internal study CI and its raw evidence belong to the private repository. Keep privileged data access separate from execution of unreviewed public code.

## Consequences

External contributors need only the public checkout for ordinary product work. Internal researchers maintain a separate pinned product selection and a private Git archive. The private repository can remain large; its size does not become a public build dependency.

Moving files does not remove earlier Git objects or withdraw content already published. This decision does not authorize rewriting shared history. History filtering and public tag treatment require a separate coordinated decision after archive restoration and public-only verification.

See [repository ownership](../project/repository-split.md) for the retained public boundary, internal evidence notation and execution requirements.

## Alternatives

A private submodule inside the public repository would make public setup depend on inaccessible state. A second maintained copy of product code would create conflicting owners. Separate object storage can be considered later, but it is not part of this migration.
