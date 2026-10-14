import test from "node:test";
import assert from "node:assert/strict";
import { makePrompt, diagnosticMetrics } from "../qwen-diagnostic.mjs";

const sample = {
  sourceText: "Wednesday, no Thursday.",
  mode: "clean",
  scenario: "email",
  context: "Permitted context",
  approvedMemory: ["Approved spelling"],
  language: "en",
  references: ["Thursday."],
  requiredStrings: ["Thursday"],
  forbiddenStrings: ["Wednesday"],
  protected: ["Thursday"],
  tags: ["self-correction"],
};

test("diagnostic prompt withholds gold labels and independently removes treatment inputs", () => {
  const prompt = makePrompt(
    {
      ...sample,
      references: ["SECRET_GOLD"],
      requiredStrings: ["SECRET_REQUIRED"],
    },
    { prompt: "full", context: false, memory: false, scenario: false },
  );
  for (const secret of [
    "SECRET_GOLD",
    "SECRET_REQUIRED",
    "Permitted context",
    "Approved spelling",
    '"email"',
  ])
    assert.equal(prompt.includes(secret), false);
  assert.ok(prompt.includes("Wednesday, no Thursday."));
  assert.ok(prompt.includes('"scenario":"generic"'));
  assert.ok(prompt.endsWith("<think>\n\n</think>\n\n"));
});

test("fluent output can fail final-intent preservation", () => {
  const m = diagnosticMetrics(sample, {
    text: "Wednesday.",
    generationMs: 12,
    firstTokenMs: 4,
    truncated: false,
    peakRssBytes: 100,
  });
  assert.equal(m.constraintPass, false);
  assert.equal(m.requiredMisses, 1);
  assert.equal(m.forbiddenHits, 1);
  assert.equal(m.protectedMisses, 1);
});

test("token-budget truncation and verbatim changes fail even when strings match", () => {
  assert.equal(
    diagnosticMetrics(sample, {
      text: "Thursday.",
      generationMs: 1,
      firstTokenMs: 1,
      truncated: true,
      peakRssBytes: 1,
    }).constraintPass,
    false,
  );
  assert.equal(
    diagnosticMetrics(
      { ...sample, mode: "verbatim" },
      {
        text: "Thursday.",
        generationMs: 1,
        firstTokenMs: 1,
        truncated: false,
        peakRssBytes: 1,
      },
    ).verbatimViolation,
    true,
  );
});

test("thinking treatment removes forced non-thinking prefix without adding outcome labels", () => {
  const prompt = makePrompt(sample, {
    prompt: "full",
    context: true,
    memory: true,
    scenario: true,
    thinking: true,
  });
  assert.ok(prompt.endsWith("<|im_start|>assistant\n"));
  assert.equal(prompt.includes("<think>"), false);
  assert.equal(prompt.includes('"references":'), false);
});

test("Make-style empty optional environment values preserve declared CLI defaults", async () => {
  const { mkdtemp, writeFile, rm, stat } = await import("node:fs/promises");
  const { tmpdir } = await import("node:os");
  const { spawnSync } = await import("node:child_process");
  const path = await import("node:path");
  const { fileURLToPath } = await import("node:url");
  const dir = await mkdtemp(path.join(tmpdir(), "arion-diagnostic-"));
  try {
    const corpus = path.join(dir, "invalid-corpus.json"),
      config = path.join(dir, "config.json"),
      out = path.join(dir, "out");
    await writeFile(corpus, "{}");
    await writeFile(config, "{}");
    const entry = fileURLToPath(
      new URL("../../../scripts/qwen-diagnostic-command.mjs", import.meta.url),
    );
    const result = spawnSync(process.execPath, [entry, "run"], {
      encoding: "utf8",
      env: {
        ...process.env,
        MODELS: "",
        EVAL_PYTHON: "",
        CORPUS: corpus,
        QWEN_CONFIG: config,
        OUT: out,
        QWEN_PROMPTS: "",
      },
    });
    assert.notEqual(result.status, 0);
    assert.match(result.stderr, /honestly labeled/);
    assert.doesNotMatch(result.stderr, /--models required|--python required/);
    await assert.rejects(stat(out), { code: "ENOENT" });
  } finally {
    await rm(dir, { recursive: true, force: true });
  }
});

test("prefix prefill contains only the fixed system instruction and is an exact prompt boundary", async () => {
  const { buildPrompt, systemPrefix } = await import(
    "../diagnostic/prompt.mjs"
  );
  const rules = { full: "Only edit quoted dictation." };
  const v = { prompt: "full", context: true, memory: true, scenario: true };
  const prefix = systemPrefix(v, rules),
    full = buildPrompt(sample, v, rules);
  assert.ok(full.startsWith(prefix));
  assert.equal(prefix.includes(sample.sourceText), false);
  assert.equal(prefix.includes(sample.context), false);
  assert.equal(prefix.includes(sample.approvedMemory[0]), false);
  assert.ok(full.slice(prefix.length).startsWith("<|im_start|>user\n"));
});
