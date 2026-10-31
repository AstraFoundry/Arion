import os from 'node:os';

// CMake job counts must follow the machine. Standard GitHub-hosted runners are
// 4 vCPU on Linux and Windows and 3 cores on macOS arm64. A fixed 6 oversubscribes them.
export function buildParallelism(env = process.env, available = os.availableParallelism()) {
  if (!Number.isInteger(available) || available < 1) throw new Error('Host parallelism is unavailable.');
  const raw = env.ARION_BUILD_JOBS || env.ARION_KEYBOARD_BUILD_JOBS || '';
  if (raw === '') return available;
  if (!/^[1-9]\d*$/.test(raw)) throw new Error('ARION_BUILD_JOBS must be a positive integer.');
  return Math.min(Number(raw), available);
}
