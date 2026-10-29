import { spawn } from 'node:child_process';
import { createInterface } from 'node:readline';

export function requireValue(condition, message) {
  if (!condition) throw new Error(message);
}

// Persistent JSONL driver: models can remain resident between explicit warm requests.
// Stdout is protocol only. Stderr is discarded to avoid leaking fixture content.
export class Driver {
  constructor(config, cwd) {
    requireValue(typeof config.executable === 'string' && config.executable.length > 0 && Array.isArray(config.args) && config.args.every(arg => typeof arg === 'string'), 'Driver requires executable and argument array');
    requireValue(Number.isInteger(config.timeoutMs) && config.timeoutMs > 0 && config.timeoutMs <= 43200000, 'Invalid driver timeout');
    this.timeoutMs = config.timeoutMs;
    this.pending = null;
    this.failure = null;
    this.sequence = 0;
    this.process = spawn(config.executable, config.args, { cwd, stdio: ['pipe', 'pipe', 'ignore'], shell: false });
    this.process.stdin.on('error', () => this.fail(new Error('Driver stdin failed')));
    this.process.on('error', () => this.fail(new Error('Driver could not start')));
    this.process.on('exit', code => this.fail(new Error(`Driver exited (${code})`)));
    // Bound a line before readline accumulates unlimited output.
    let lineBytes = 0;
    this.process.stdout.on('data', chunk => {
      for (const byte of chunk) {
        lineBytes = byte === 10 ? 0 : lineBytes + 1;
        if (lineBytes > 8 * 1024 * 1024) { this.fail(new Error('Driver response exceeds limit')); break; }
      }
    });
    this.lines = createInterface({ input: this.process.stdout, crlfDelay: Infinity });
    this.lines.on('line', line => {
      if (!this.pending) { this.fail(new Error('Unsolicited driver response')); return; }
      try {
        const response = JSON.parse(line);
        requireValue(response.requestId === this.pending.id && response.protocol === 1, 'Driver protocol/identity mismatch');
        const { resolve, timer } = this.pending;
        this.pending = null;
        clearTimeout(timer);
        resolve(response);
      } catch { this.fail(new Error('Invalid driver JSONL response')); }
    });
  }
  fail(error) {
    this.failure ??= error;
    if (this.pending) {
      clearTimeout(this.pending.timer);
      this.pending.reject(error);
      this.pending = null;
    }
    this.process.kill();
  }
  request(payload) {
    requireValue(!this.pending, 'Driver requests must be sequential');
    if (this.failure) return Promise.reject(this.failure);
    const requestId = String(++this.sequence);
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => this.fail(new Error('Driver request timed out')), this.timeoutMs);
      this.pending = { id: requestId, resolve, reject, timer };
      this.process.stdin.write(`${JSON.stringify({ ...payload, protocol: 1, requestId })}\n`);
    });
  }
  close() {
    const closed = this.process.exitCode !== null || this.process.signalCode !== null
      ? Promise.resolve() : new Promise(resolve => this.process.once('close', resolve));
    this.lines.close();
    this.process.kill();
    return closed;
  }
}
