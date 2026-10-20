# Rewrite iteration report

`index.html` and `data.json` are about 103MB and 122MB. They are omitted from git because the host rejects blobs over 100MB. Rebuild them locally with `pnpm rewrite:release:report` and validate with `pnpm rewrite:release:test`. The study notes linked below are the committed record.

The sentence below that production prompts were unchanged describes this study checkpoint. [ADR 018](../../../docs/decisions/018-staged-gguf-rewriting.md) later adopted the selected staged prompts.

The current report includes76 rounds and11,513 actual model observations:5,475 intermediate spelling choices,72 native typed decisions with zero generated tokens and67 distinct runtime-probe responses are explicitly separated from final bodies. Warmups retain their original roles. The113 historical auditor outputs and24 macOS Vision observations are separate evidence. Six Granite HTTP errors and the earlier native startup failure have no completed-response credit. One interrupted MiniCPM completion is recovered only from its native log and disclosed separately, without complete-response credit. Every displayed raw response was personally reviewed by the assistant; this is not independent blind review.

Latest selector development evidence:

- [Added semantic contrast demonstrations](../../reproduction/rewrite-selector-contrast-demo-v1/README.md):404 actual choices, conforms92/90/94/92 and harmful3/3/4/6. Four authored examples add one gain and five regressions; both variants rejected. Native template/token-budget checks generate zero answers. No new final bodies.

- [Unique-label exposure](../../reproduction/rewrite-selector-unique-label-v1/README.md):404 actual choices; conforms92/91/94/91, harmful3/4/4/7. Only status:unique is omitted; all four changed choices regress. Both variants rejected, no new final bodies.

- [Complete-reference message separation](../../reproduction/rewrite-selector-reference-turn-v1/README.md):404 actual choices; conforms92/89/94/91 and harmful3/9/4/8. Full OCR moves to preceding tool-role turns, with all source data/rules/answers retained; both variants fail. Four native template applications generate zero outputs. No new final bodies.

- [Trusted system-language comparison](../../reproduction/rewrite-selector-instruction-language-v1/README.md):404 actual choices. English Qwen conforms88/87 with10/12 harmful versus Chinese92/94 and3/4; English hybrid80/79 with21/22 harmful versus81/82 and20/19. Only system text changes; all four variants fail. No new final bodies.

- [Granite1.46B hybrid](../../reproduction/rewrite-selector-hybrid-v1/README.md):12 native probes have valid JSON/template/cache observations but4 task failures;202 selector choices conform81/82 with20/19 harmful, despite47/47 required repairs. Prior same-request Qwen92/94 and3/4 harmful. Both arms fail; runtime probes remain separate from selector/final-body counts.

- [Counterbalanced whole-sentence selection](../../reproduction/rewrite-selector-sentence-choice-v1/README.md):404 actual choices; conformance92/94/81/91, harmful3/4/19/5. Swapping A/B changes24 normalized choices:20 conformance changes,4 both-correct normalization differences.12 forced-original ambiguity cases per arm are disclosed. Both new orders fail; no new final bodies.

- [Source-last serialization](../../reproduction/rewrite-selector-source-last-v1/README.md):404 actual choices; conformance92/93/94/88 and harmful3/5/4/12.202 controls reproduce earlier decisions;202 reordered requests restore exactly after source order only.17 changed pairs reviewed. Both candidates fail; no new final bodies.

- [Public referent-description screen](../../reproduction/rewrite-selector-referent-v1/README.md):404 actual choices across four equal-budget arms; conformance92/92/94/93 and harmful3/4/4/5. All212 public descriptions reviewed;44 fail, including27 cases with correct choices. The202 current controls reproduce old decisions/edited sources202/202. Both new arms fail; no final-body expansion. Independent description badges and filters do not change choice scores or runtime acceptance.

- [Four-arm packaging screen](../../reproduction/rewrite-selector-screen-v1/README.md):404 actual choices, conformance92/88/94/83 of101, harmful3/12/4/17; all fail. All457 actual Go preparations and1,828 source-treatment outcomes are retained. No new final bodies.
- [Granite runtime diagnosis](../../reproduction/rewrite-selector-model-v1/README.md): initial202-call plan stops at the first HTTP500 with0 completed responses. Twelve later diagnostic attempts complete7 responses and5 HTTP errors. Metal without schema outputs repeated8; CPU-layer0 returns valid probe JSON. f32KV/flash-off do not recover meaningful Metal output. No NaN measurement or exact kernel cause is established.
- [Granite CPU-layer comparison](../../reproduction/rewrite-selector-model-cpu-v1/README.md):202 actual choices,84/86 conforming of101, harmful16/13 and missed1/2. Same-request Qwen controls have92/94 conforming and3/4 harmful choices. Thirty choices change:7 improve,23 regress. Provider/tokenizer/checkpoint differ, preventing checkpoint-only or parameter-count causal attribution. Both Granite configurations fail; no formatter/fresh expansion occurs.

Every actual request, fixture-assigned/effective scene, output, source control, candidate, edit and review remains inspectable. Runtime probes have a separate stage/filter and are excluded from final-body and semantic-fidelity counts. HTTP failures retain their complete requests and errors without fabricated bodies. The source-letter overlay permits `read cash` to `readCash`, but unsupported `readCache` remains a failure; original historical judgments and decisions are preserved.

Natural final-body observations remain1,560 over120 historical sources/13 treatments, not1,560 independent sources. The latest contrast-demonstration study performs457 fresh actualGo preparations reused across four arms; the prior instruction-language comparison replays457 original preparations. Neither generates newASR, Vision or final bodies. Existing source uncertainty, historical timing inconsistencies and rejected candidates remain visible rather than being relabeled as release success.

Production prompts, validators and native integration are unchanged. No new4B or olderQwen inference is added. macOS comes first; minimum Apple Silicon16GB remains unverified on the currentM4/32GB development host. Cross-script terminology, complete-chain quality, fresh/currentASR+Vision, native lifecycle/offline/concurrent stop-to-insertion and independent meaning/style review remain mandatory before release.

- [MiniCPM5-1B comparison](../../reproduction/rewrite-selector-minicpm-v1/README.md):202 unchanged-request choices conform80/77, harmful11/4, missed10/20. Both variants fail;12 native responses all pass protocol but fail task meaning. One interrupted native-log completion and the pinned lexer template correction are separately disclosed.

- [MiniCPM F16 export comparison](../../reproduction/rewrite-selector-minicpm-f16-v1/README.md):202 complete choices conform82/82, harmful15/17, missed4/2, required repairs43/45 of47.25 improvements and18 regressions versusQ4; both configurations rejected.12 native responses separate. Zero-generation official/native tokenizer proof is also displayed.

- [Youtu1.96B Q8](../../reproduction/rewrite-selector-youtu-v1/README.md):12 native probes and202 selector choices, conforming75/84 with3/4 harmful and23/13 missed; both arms rejected. Custom-license research, no final bodies.

- [LFM2.5 actual1.17B Q8](../../reproduction/rewrite-selector-lfm25-v1/README.md):12 native probes and202 selector outputs reviewed. Both79/101,22 harmful each,0 missed and47/47 repaired; rejected. No final bodies.

- [Original demonstration order only](../../reproduction/rewrite-selector-demo-order-v1/README.md):404 actual choices, conforms92/78/94/88, harmful3/2/4/2 and missed6/21/3/11.3 improvements,23 regressions,1 allowed normalization change. Both treatments rejected; no new final bodies. Current16GB device unavailable;32GB development verification continues with minimum acceptance still open.

- [Native d1 packaging](../../reproduction/rewrite-d1-evidence-v1/README.md):72 actual typed responses across six layouts/CPU/Metal, zero generated tokens. Trained96/45 piece clipping identified; complete-policy correction remains3/6 per backend. No new final bodies; all configurations unqualified.
