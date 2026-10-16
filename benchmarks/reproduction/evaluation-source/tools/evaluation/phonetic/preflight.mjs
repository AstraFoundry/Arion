export function outputContractFailure(response) {
  return typeof response?.text !== "string" || response.truncated === true ||
    /^\s*[\[{]/u.test(response.text) || /<\|im_|<\/?think>/u.test(response.text);
}

export function validatePreflight(meta, rows, config, promptHash, developmentIds) {
  if (meta.kind !== "development-output-contract-smoke" || meta.status !== "complete" ||
      meta.testCases !== 0 || meta.promptControllerHash !== promptHash)
    throw new Error("Completed development preflight for the exact prompt required");
  const expected = new Set(config.models.flatMap(model => config.variants.flatMap(variant =>
    ["zh", "en", "mixed"].map(language => `${model}:${variant}:${language}`))));
  const seen = new Set();
  let negativeControlFailures = 0;
  for (const row of rows) {
    const key = `${row.model}:${row.variant}:${row.language}`;
    if (row.split !== "dev" || !developmentIds.has(row.caseId)) throw new Error("Preflight must use development cases only");
    if (!expected.has(key) || seen.has(key)) throw new Error("Unexpected or duplicate preflight observation");
    seen.add(key);
    if (outputContractFailure(row.response)) {
      if (row.variant !== "ipa_only") throw new Error("Text-evidence output contract failed during development");
      negativeControlFailures++;
    }
  }
  if (seen.size !== expected.size) throw new Error("Incomplete preflight coverage");
  return { observations: rows.length, textEvidenceContractFailures: 0, negativeControlFailures };
}
