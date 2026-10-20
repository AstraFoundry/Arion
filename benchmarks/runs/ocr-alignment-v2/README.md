# OCR alignment prompt study evidence

`dev/` retains 32 reviewed outputs and the diagnostic selection frozen before
evaluation. The selection's timing-wording correction is explicit in
`selection-clarification.json`; selected prompt and evidence hashes are unchanged.
`evaluation/` retains all 96 paired prior/selected observations, exact contexts,
cleanup values and complete assistant judgments. Worker loads/stderr and
model/runtime/build/prompt provenance remain with each run.

`parity/` retains 12 supplementary cold/forward/reverse requests and complete
review. It found no order mismatch on the four probed inputs. `report/` contains
all 128 main reviews, 384 exact strict/envelope/none deliveries, scene/template
metadata and the full parity appendix. All source/OCR-like texts are authored;
no actual speech recognition, screenshot OCR or native insertion is claimed.

See [results](../../results/ocr-alignment-v2/README.md) and
[reproduction](../../reproduction/ocr-alignment-v2/README.md). Archive hashes are
integrity checks, not an external signature or independent timestamp.
