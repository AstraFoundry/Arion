// The previews of native UI's docs: docs-demo/ is a Go app of native UI,
// which this plugin compiles to WebAssembly and publishes at /docs-demo,
// with Go's loader and the page loading it. A page of docs/ui shows one of
// its demos in a frame where it has `<!-- preview name height -->`
// (markdown.server.ts). The example apps of `apps` are compiled beside it,
// for pages that run one in a frame, `<!-- app name height -->`.
import { execFile } from "node:child_process"
import crypto from "node:crypto"
import fs from "node:fs/promises"
import os from "node:os"
import path from "node:path"
import { promisify } from "node:util"
import type { Plugin } from "vite"

const run = promisify(execFile)

/** The example apps pages run, by name: examples/<name>. */
export const apps = ["mayoi", "loading"]

/** The package of an app, "" for docs-demo itself. */
const pkg = (app: string) => (app ? `./examples/${app}` : "./website/docs-demo")

/** Compiles an app for browsers. Go must be installed: the build fails without it. */
async function compile(repoDir: string, app = ""): Promise<Buffer> {
  const dir = await fs.mkdtemp(path.join(os.tmpdir(), "docs-demo-"))
  const out = path.join(dir, "app.wasm")
  const env = { ...process.env, GOOS: "js", GOARCH: "wasm", CGO_ENABLED: "0" }
  try {
    await run("go", ["build", "-trimpath", "-ldflags=-s -w", "-o", out, pkg(app)], { cwd: repoDir, env, maxBuffer: 1 << 24 })
    return await fs.readFile(out)
  } catch (error) {
    const e = error as { stderr?: string; message: string }
    throw new Error(`docs-demo: go build of ${pkg(app)} failed: ${e.stderr || e.message}`)
  } finally {
    await fs.rm(dir, { recursive: true, force: true })
  }
}

/** Go's loader of WebAssembly, of the Go that compiles the app. */
async function loader(repoDir: string): Promise<string> {
  const { stdout: goroot } = await run("go", ["env", "GOROOT"], { cwd: repoDir })
  return fs.readFile(path.join(goroot.trim(), "lib", "wasm", "wasm_exec.js"), "utf8")
}

/**
 * The page of a demo or an app, loading it from wasm: files names the
 * file of /docs-demo of each, by app, "" for docs-demo.
 */
async function page(repoDir: string, files: Record<string, string>): Promise<string> {
  return (await fs.readFile(path.join(repoDir, "website", "docs-demo", "index.html"), "utf8")).replace("__WASM__", JSON.stringify(files))
}

const types: Record<string, string> = {
  html: "text/html; charset=utf-8",
  js: "text/javascript; charset=utf-8",
  wasm: "application/wasm",
}

export function docsDemo(repoDir: string): Plugin {
  // The development server compiles an app as frames ask for it, once for
  // frames loading together, so that a reload shows the demos as edited.
  const building = new Map<string, Promise<Buffer>>()
  const all = ["", ...apps]
  return {
    name: "mygo-docs-demo",
    applyToEnvironment: (environment) => environment.name === "client",
    async generateBundle() {
      const files: Record<string, string> = {}
      for (const app of all) {
        const wasm = await compile(repoDir, app)
        // Named by its content, so that browsers keep it as long as it lasts.
        const name = `${app || "docs-demo"}.${crypto.createHash("sha256").update(wasm).digest("hex").slice(0, 12)}.wasm`
        this.emitFile({ type: "asset", fileName: `docs-demo/${name}`, source: wasm })
        files[app] = name
      }
      this.emitFile({ type: "asset", fileName: "docs-demo/index.html", source: await page(repoDir, files) })
      this.emitFile({ type: "asset", fileName: "docs-demo/wasm_exec.js", source: await loader(repoDir) })
    },
    configureServer(server) {
      server.middlewares.use(async (req, res, next) => {
        const m = /^\/docs-demo(?:\/(.*))?$/.exec(req.url?.split("?", 1)[0] ?? "")
        if (!m) return next()
        const file = m[1] || "index.html"
        const app = all.find((a) => file === `${a || "docs-demo"}.wasm`)
        try {
          let source: string | Buffer
          if (file === "index.html") source = await page(repoDir, Object.fromEntries(all.map((a) => [a, `${a || "docs-demo"}.wasm`])))
          else if (file === "wasm_exec.js") source = await loader(repoDir)
          else if (app !== undefined) {
            let b = building.get(app)
            if (!b) {
              b = compile(repoDir, app).finally(() => setTimeout(() => building.delete(app), 1000))
              building.set(app, b)
            }
            source = await b
          } else return next()
          res.setHeader("Content-Type", types[file.split(".").pop() ?? ""] ?? "application/octet-stream")
          res.setHeader("Cache-Control", "no-store")
          res.end(source)
        } catch (error) {
          next(error)
        }
      })
    },
  }
}
