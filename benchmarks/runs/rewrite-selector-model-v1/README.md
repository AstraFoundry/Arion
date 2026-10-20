# Granite runtime failure and diagnosis

The original fixed Metal comparison fails on its first actual request and completes0 of202 planned responses. The complete failed run is retained in `failed-schema/`; original and corrected closure metadata are separate. No planned observation is counted as generated.

`diagnostic/` and `precision/` retain12 actual diagnostic attempts: seven completed model responses and five HTTP errors. The initial error is a sixth HTTP failure. `diagnostic-responses/` is the exact checked display projection of those seven completed responses, not new inference. All outputs were personally reviewed. The observed CPU/Metal difference and unresolved numerical/kernel cause remain explicit.

Original model/runtime sources, launchers, protocols and reproduction operations are [documented here](../../reproduction/rewrite-selector-model-v1/README.md). Weights and runtime executables are excluded. This evidence establishes no final-body, minimum16GB or product acceptance.
