# Independently authored instruction-only prompt candidate

Author: OpenCode model `opencode-go/gemini-4-argon-contributor` (provider/model confirmed by the exported session). The accepted session has an explicitly bound empty temporary project directory outside Arion and deny-all tool permissions. The complete authoring brief contains only the product interfaces and general behavior requirements: no old templates, demonstrations, benchmark cases, reference answers, measurements or previous model drafts.

- `selection.json`: zero-shot source-relative spelling selection; the external keep/replace schema remains the interface contract.
- `format.json`: zero-shot source-only formatting with work, email, prompt and chat instructions, plus Han/Latin/mixed script routes.
- `candidate.json`: original complete model-authored JSON answer, including its design notes. Prompt strings are not manually rewritten after generation; the split files are JSON serialization only.

All demonstration arrays are empty. This deliberately differs from the shipped few-shot recipe. The current production formatter builder requires at least one routed example and therefore needs an explicit zero-example mode before this candidate can be wired into that product path. The candidate itself can be sent as system instructions plus the final data turn; no runtime integration or production replacement is claimed here.

The author's initial correctly isolated draft was reviewed only against the generic interface specification. A follow-up clarified that the formatter returns the plain edited body without preface, explanation, JSON envelope or surrounding code fences. No evaluation case, output or score was used in that revision. Original drafts and the follow-up are preserved.

This establishes the declared authoring procedure, not proof of generalization or absence of pretraining contamination. No local Qwen inference, new quality score or latency/budget-fit measurement is claimed. Evaluate a frozen candidate on independently collected unseen data before using results as generalization evidence.

[Authoring provenance and original responses](../../runs/rewrite-independent-prompts-v1/README.md).
