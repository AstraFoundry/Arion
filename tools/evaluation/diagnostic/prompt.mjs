export function buildPrompt(sample, variant, prompts) {
  const data = {
    transcript: sample.sourceText,
    mode: sample.mode,
    scenario: variant.scenario ? sample.scenario : "generic",
    context: variant.context ? sample.context : "",
    approvedTerminology: variant.memory ? sample.approvedMemory : [],
  };
  return `${systemPrefix(variant, prompts)}<|im_start|>user\n${JSON.stringify(data)}<|im_end|>\n<|im_start|>assistant\n${variant.thinking ? "" : "<think>\n\n</think>\n\n"}`;
}

export function systemPrefix(variant, prompts) {
  return `<|im_start|>system\n${prompts[variant.prompt]}<|im_end|>\n`;
}
