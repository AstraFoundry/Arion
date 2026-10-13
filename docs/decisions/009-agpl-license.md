# ADR 009: AGPL-3.0 license

Status: Accepted.

## Context

Arion is published at [AstraFoundry/Arion](https://github.com/AstraFoundry/Arion) and had no license. AstraFoundry wants the source open while keeping the option of a commercial hosted (SaaS) edition, for example a managed sync relay ([003 Memory sync](../../specs/003-memory-sync.md)).

## Decision

The repository is licensed under the **GNU Affero General Public License v3.0** (`LICENSE`, SPDX `AGPL-3.0-only`). Settings › About names the license and links to it (`ui/shared/config`).

## Consequences

- Anyone may use, modify and redistribute Arion; a third party that offers a modified Arion or relay over a network must publish its source under the same license.
- AstraFoundry, as copyright holder, can run its own hosted edition or license the code commercially. This holds only while AstraFoundry owns, or is licensed to relicense, every contribution: external contributions need a contributor license agreement (CLA) before merge.
- Bundled third-party components keep their own licenses (sherpa-onnx and ONNX Runtime under Apache-2.0, Lucide under ISC, see `ui/shared/icons/svg/LICENSE`); all are compatible with AGPL-3.0 distribution.

## Alternatives considered

- **MIT / Apache-2.0:** maximal adoption, but a competitor could host a closed SaaS fork with no obligation.
- **GPL-3.0:** copyleft applies only on distribution, so hosted use escapes it.
- **Source-available (BSL, FSL):** stronger commercial protection but not open source, which conflicts with the local-first, inspectable privacy promise.

## References

- [GNU AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.html)
