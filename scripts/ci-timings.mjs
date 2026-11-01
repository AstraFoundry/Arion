// Collect reproducible job/step evidence with gh; retain unsuccessful samples.
import { execFileSync } from 'node:child_process';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const seconds = (start, end) => {
  if (!start || !end) return null;
  const value = (Date.parse(end) - Date.parse(start)) / 1000;
  if (!Number.isFinite(value)) throw new Error(`Invalid timestamp: ${start} -> ${end}`);
  if (value < 0) return null;
  return value;
};

export function isCheckStep(name) {
  return /^(Run pnpm (?:check|test|workflows:check|updater:check|notifications:check)|Marketing site|Desktop checks|Keyboard checks|Native window tests|Windows platform tests|Windows desktop,|Windows updater regression|Isolated native updater|Build and verify desktop package|Run node scripts\/(?:check-|run-native-e2e))/.test(name);
}

export function jobTiming(run, job) {
  const runtime = job.conclusion === 'skipped' ? null : seconds(job.started_at, job.completed_at);
  const queue = job.conclusion === 'skipped' ? null : seconds(job.created_at, job.started_at);
  const checks = job.steps.filter(step => isCheckStep(step.name) && step.started_at && step.completed_at && step.conclusion !== 'skipped')
    .reduce((sum, step) => sum + seconds(step.started_at, step.completed_at), 0);
  if (runtime !== null && checks > runtime) throw new Error(`Check intervals exceed job runtime: ${job.id}`);
  return {
    run_id: run.databaseId, job_id: job.id, attempt: job.run_attempt, event: run.event,
    branch: run.headBranch, job: job.name, conclusion: job.conclusion ?? job.status,
    timestamp_anomaly: Boolean(job.started_at && job.completed_at && job.completed_at < job.started_at || job.created_at && job.started_at && job.started_at < job.created_at),
    dependency_wait_seconds: seconds(run.createdAt, job.created_at),
    queue_seconds: queue,
    preparation_seconds: runtime === null ? null : runtime - checks,
    checks_seconds: runtime === null ? null : checks,
    runtime_seconds: runtime, url: job.html_url,
  };
}

export function summarize(rows) {
  const median = values => {
    const sorted = values.sort((a, b) => a - b), middle = Math.floor(sorted.length / 2);
    return sorted.length % 2 ? sorted[middle] : (sorted[middle - 1] + sorted[middle]) / 2;
  };
  return [...new Set(rows.map(row => row.job))].map(job => {
    const samples = rows.filter(row => row.job === job && row.conclusion === 'success' && row.runtime_seconds !== null);
    return { job, successful_samples: samples.length, ...Object.fromEntries(
      ['dependency_wait_seconds', 'queue_seconds', 'preparation_seconds', 'checks_seconds', 'runtime_seconds'].map(key => {
        const values = samples.map(row => row[key]).filter(value => value !== null);
        return [key, values.length ? median(values) : null];
      })) };
  });
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const [directory, mode] = process.argv.slice(2);
  if (!directory || (mode && mode !== '--reuse')) throw new Error('Usage: node scripts/ci-timings.mjs <evidence-directory> [--reuse]');
  mkdirSync(directory, { recursive: true });
  const api = args => JSON.parse(execFileSync('gh', args, { encoding: 'utf8', maxBuffer: 32 * 1024 * 1024 }));
  const runFile = path.join(directory, 'baseline-runs.json');
  const runs = mode ? JSON.parse(readFileSync(runFile, 'utf8')) : api(['run', 'list', '--limit', '50', '--json', 'databaseId,name,event,headBranch,headSha,status,conclusion,createdAt,startedAt,updatedAt,url']);
  if (!mode) writeFileSync(runFile, `${JSON.stringify(runs, null, 2)}\n`);
  const rows = [];
  for (const run of runs) {
    const jobFile = path.join(directory, `${run.databaseId}.json`);
    const jobs = mode ? JSON.parse(readFileSync(jobFile, 'utf8')).jobs : api(['api', '--paginate', '--slurp', `repos/{owner}/{repo}/actions/runs/${run.databaseId}/jobs?filter=all&per_page=100`]).flatMap(page => page.jobs);
    if (!mode) writeFileSync(jobFile, `${JSON.stringify({ jobs }, null, 2)}\n`);
    rows.push(...jobs.map(job => jobTiming(run, job)));
  }
  const keys = Object.keys(rows[0] ?? {});
  const quote = value => `"${String(value ?? '').replaceAll('"', '""')}"`;
  writeFileSync(path.join(directory, 'jobs.csv'), [keys.join(','), ...rows.map(row => keys.map(key => quote(row[key])).join(','))].join('\n') + '\n');
  const summary = summarize(rows);
  writeFileSync(path.join(directory, 'summary.json'), `${JSON.stringify(summary, null, 2)}\n`);
  console.table(summary);
  console.log(`${runs.length} runs, ${rows.length} jobs; cancelled, failed and skipped samples retained in jobs.csv.`);
}
