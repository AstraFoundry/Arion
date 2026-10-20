# Independent prompt authoring provenance

The accepted candidate was authored by `opencode-go/gemini-4-argon-contributor` through OpenCode v2.0.32. Its exported session confirms both the provider/model and an explicitly supplied temporary directory outside the Arion repository. The session has deny-all tool permissions; accepted response events contain no tool use.

The archived material preserves the supplied general product brief, output-contract clarification, raw model responses, exported session, exact generation/extraction orchestration and final schema/provenance verification. Old product prompts, test cases, expected answers, case review, scores and earlier model drafts were not supplied to this session. All example arrays in the candidate are empty. No product file was replaced and no local Qwen evaluation was performed.

## Discarded attempts

The first custom-agent invocation failed before generation because the agent was not found. Subsequent implicit-location drafting and revision completed, but the exported session still pointed to Arion despite a temporary launch cwd. Those drafts were disqualified as independent candidates. They are preserved under `discarded-implicit-location`; neither draft was input to the accepted, separately created session. The custom-agent failure is preserved separately.

The accepted session was created with an explicit location and its directory/model/permissions were checked from actual session metadata. Its first answer and a subsequent interface-only clarification remain distinguishable in the raw records. This is an authoring artifact, not an experiment demonstrating rewrite accuracy or an independently verified production release.

[Candidate templates](../../reproduction/rewrite-independent-prompts-v1/README.md). `archive.json` binds the completed authoring bytes by SHA-256; the candidate's generation and lineage summary is in `summary.json`.
