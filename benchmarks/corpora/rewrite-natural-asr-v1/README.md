# Frozen natural read-speech source preparation

120 existing public read-speech inputs:60 Mandarin AISHELL-1 and60 English
LibriSpeech records. Every audio SHA256 and ASR text is checked against the
original archived acoustic extraction. Original speakers, sessions and split
are retained in the evaluator view. The historical derived raw controls match
the original acoustic rows; their zero generationMs placeholders are not timing.

This is source preparation, with **zero new ASR or LLM inference**. It is historical
exposed regression, not a new held-out set, spontaneous dictation, candidate
quality evidence or independent meaning/style review. Scene is assigned chat for
future read-speech controls; this does not cover real work/email interaction.
`inputs.json` has only source/context/scene/language/identity fields. Published
reference text and original speaker/session identities are in `evaluator.json`,
never model messages. `audio-lineage.json` links original audio and exact raw-line
hashes. The complete historical corpus remains unchanged.

OpenSLR33 lists Apache2 and also describes academic use; both statements and the
existing source-card caveat are preserved without a legal conclusion. OpenSLR12
lists CC-BY4.0 and attribution to LibriSpeech/LibriVox. Original source pages,
retrieval metadata, corpus provenance and hashes are retained. Chinese-only
experimental prompts cannot become production defaults without preserving
English and mixed-language input as required by the production prompt contract.
