import assert from 'node:assert/strict';
import test from 'node:test';
import { isCheckStep, jobTiming, summarize } from '../ci-timings.mjs';

const run = { databaseId: 1, event: 'push', headBranch: 'main', createdAt: '2026-01-01T00:00:00Z' };
const job = { id: 2, name: 'portable', conclusion: 'success', created_at: '2026-01-01T00:01:00Z', started_at: '2026-01-01T00:01:05Z', completed_at: '2026-01-01T00:02:05Z', steps: [
  { name: 'Run pnpm test', conclusion: 'success', started_at: '2026-01-01T00:01:35Z', completed_at: '2026-01-01T00:01:55Z' },
] };

test('dependency waiting, runner queue, preparation and checks remain distinct', () => {
  const row = jobTiming(run, job);
  assert.equal(row.dependency_wait_seconds, 60);
  assert.equal(row.queue_seconds, 5);
  assert.equal(row.preparation_seconds, 40);
  assert.equal(row.checks_seconds, 20);
  assert.equal(row.runtime_seconds, 60);
  assert.equal(isCheckStep('Run pnpm install --frozen-lockfile'), false);
  assert.equal(isCheckStep('Restore native dependency builds'), false);
});

test('incomplete jobs have no measured runtime; unsuccessful samples do not enter medians', () => {
  const success = jobTiming(run, job);
  const cancelled = jobTiming(run, { ...job, conclusion: 'cancelled', completed_at: null });
  assert.equal(cancelled.runtime_seconds, null);
  assert.equal(cancelled.preparation_seconds, null);
  assert.equal(summarize([success, cancelled])[0].successful_samples, 1);
  assert.equal(summarize([success, cancelled])[0].runtime_seconds, 60);
});

test('API timestamp anomalies are explicit and never become negative queue durations', () => {
  const row = jobTiming(run, { ...job, created_at: '2026-01-02T00:00:00Z' });
  assert.equal(row.timestamp_anomaly, true);
  assert.equal(row.queue_seconds, null);
  assert.equal(row.runtime_seconds, 60);
});


test('retry metadata carrying an earlier successful result is retained but not counted as another execution', () => {
  const original = jobTiming(run, { ...job, run_attempt: 1 });
  const carried = jobTiming(run, { ...job, id: 3, run_attempt: 2, created_at: '2026-01-01T00:03:00Z' });
  assert.equal(carried.reused_result, true);
  assert.equal(carried.runtime_seconds, 60);
  assert.equal(summarize([original, carried])[0].successful_samples, 1);
});
