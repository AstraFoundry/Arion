import { spawn } from "node:child_process";
import readline from "node:readline";

export class Worker {
  constructor(python, diagnostics, workerFile) {
    this.id = 0;
    this.pending = new Map();
    this.child = spawn(python, ["-I", workerFile], {
      stdio: ["pipe", "pipe", diagnostics],
      env: { ...process.env, HF_HUB_OFFLINE: "1", TRANSFORMERS_OFFLINE: "1" },
    });
    readline
      .createInterface({ input: this.child.stdout })
      .on("line", (line) => {
        try {
          const r = JSON.parse(line);
          const waiter = this.pending.get(r.id);
          if (!waiter) throw new Error("Unexpected worker response");
          this.pending.delete(r.id);
          clearTimeout(waiter.timer);
          if (Date.now() > waiter.wallDeadline) {
            waiter.reject(
              new Error("Inference response exceeded its wall-clock deadline"),
            );
            this.child.kill();
            return;
          }
          r.ok ? waiter.resolve(r) : waiter.reject(new Error(r.error));
        } catch (error) {
          this.fail(error);
        }
      });
    this.child.on("error", (e) => this.fail(e));
    this.child.on("exit", (code) =>
      this.fail(new Error(`Worker exited ${code}`)),
    );
  }
  fail(e) {
    for (const w of this.pending.values()) {
      clearTimeout(w.timer);
      w.reject(e);
    }
    this.pending.clear();
  }
  request(request) {
    const id = String(++this.id);
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        this.child.kill();
        reject(new Error("Inference timeout"));
      }, 180000);
      this.pending.set(id, {
        resolve,
        reject,
        timer,
        wallDeadline: Date.now() + 180000,
      });
      this.child.stdin.write(JSON.stringify({ id, ...request }) + "\n");
    });
  }
  async close() {
    await this.request({ op: "close" });
    this.child.stdin.end();
  }
}
