# Actual MiniCPM observations

See the [complete protocol and result](../../reproduction/rewrite-selector-minicpm-v1/README.md). `comparison-run/` retains202 complete actual choices and914 source-treatment outcomes. `preflight/` retains12 actual native responses. `runtime-view/` is a derived view of those12 responses, not extra inference. `aborted-before-qualification/` preserves the interrupted attempt:0 captured responses,1 actual completion only in native log, separately reviewed. No fabricated envelope or final body.

`path-mappings.json` maps the original private execution paths. Weights, runtime environments and native binaries remain private. The archive binds original inputs, failed and completed attempts, raw responses, decisions, sources and reviews by SHA256. Original source manifests are retained even when their running status predates an authoritative interrupted closure.
