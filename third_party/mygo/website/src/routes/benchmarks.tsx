import { createFileRoute } from "@tanstack/react-router"
import * as React from "react"

import { formatValue, frameworks, results, scenes, sourceUrl, startMeasures, type Format, type Measure, type Series } from "@/lib/benchmarks"

const title = "Benchmarks · MyGo"
const description = "MyGo's native UI, GPUI, GPUI Fast and egui on the same apps: startup, memory, scrolling a long list, a live dashboard and wrapping text, measured on the same machine."
const link = "font-medium underline decoration-gopher/45 underline-offset-[5px] hover:decoration-gopher"

export const Route = createFileRoute("/benchmarks")({
  head: () => ({
    meta: [
      { title },
      { name: "description", content: description },
      { property: "og:title", content: title },
      { property: "og:description", content: description },
    ],
  }),
  component: Benchmarks,
})

function Benchmarks() {
  return (
    <main className="flex-1">
      <section className="border-b px-4 pt-16 pb-12 sm:px-10 md:pt-20">
        <p className="label">Benchmarks</p>
        <h1 className="mt-5 text-3xl leading-[1.1] font-semibold tracking-[-0.035em] sm:text-4xl">The same apps, in four frameworks.</h1>
        <p className="mt-5 max-w-2xl text-base leading-7 text-muted-foreground">
          The same small apps written with MyGo&apos;s native UI, GPUI, GPUI Fast and egui, each the usual way of its framework, showing the same text in windows of the same size, and
          measured on the same Mac.
        </p>
        <p className="mt-6 flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
          <a href={sourceUrl} className={link}>
            The apps
          </a>
          <a href={`${sourceUrl}#what-is-measured`} className={link}>
            How they are measured
          </a>
          <a href={`${sourceUrl}/results.json`.replace("/tree/", "/blob/")} className={link}>
            The data
          </a>
        </p>
        <Legend />
      </section>

      <Section title="Starting and resting" text="A number and three buttons, then nothing to do: what an app costs before it does anything.">
        <Bars title="App size" text="A release build, with its symbols stripped." format="bytes" values={frameworks.map((f) => ({ id: f.id, series: flat(results.frameworks[f.id]?.size) }))} />
        {startMeasures.map((m) => (
          <MeasureBars key={m.id} measure={m} scene="counter" />
        ))}
      </Section>

      {scenes.map((s) => (
        <Section key={s.id} title={s.title} text={s.text}>
          {s.measures.map((m) => (
            <MeasureBars key={m.id} measure={m} scene={s.id} />
          ))}
        </Section>
      ))}

      <section className="px-4 py-10 sm:px-10">
        <h2 className="text-xl font-semibold tracking-tight">Reading the results</h2>
        <ul className="mt-4 max-w-2xl list-disc space-y-2 pl-5 text-sm leading-6 text-muted-foreground">
          <li>Shorter bars are better, but for frame rate. Each bar is the median of the runs: hover it for the lowest and the highest.</li>
          <li>Frame rate stops at the display&apos;s refresh rate: an app below it missed frames. Time between frames is taken as each frame starts, not as it reaches the screen.</li>
          <li>CPU time leaves out what the GPU does: two apps taking the same CPU may not cost the same to draw.</li>
          <li>The Mac was in ordinary use during the runs. The frameworks took turns, so that this weighs on them alike.</li>
          <li>CPU per frame has moved by as much as a third between full runs of the same build, more than the range of one run shows: read small gaps as ties.</li>
          <li>MyGo&apos;s list measures its rows, which may differ in height, where the uniform lists of GPUI and GPUI Fast and egui&apos;s rows take one height for all.</li>
          <li>GPUI Fast is a fork of GPUI that keeps what did not change between frames. Its list scrolls from outside the view, as a wheel does, so that its cached rows apply; the other scenes change every frame.</li>
          <li>MyGo, GPUI and GPUI Fast draw text with the system&apos;s font and egui with its own. The crates of GPUI and GPUI Fast come without widgets, so their buttons are styled elements, and without scroll bars.</li>
        </ul>
        <Setup />
      </section>
    </main>
  )
}

function flat(value: number | undefined): Series | undefined {
  return value === undefined ? undefined : { median: value, low: value, high: value }
}

function Legend() {
  return (
    <ul className="mt-8 flex flex-wrap gap-x-6 gap-y-2 text-sm">
      {frameworks.map((f) => (
        <li key={f.id} className="flex items-center gap-2">
          <span className="size-2.5 rounded-[3px]" style={{ background: f.color }} />
          <a href={f.url} className="font-medium hover:underline hover:underline-offset-4">
            {f.label}
          </a>
        </li>
      ))}
    </ul>
  )
}

function Section({ title, text, children }: { title: string; text: string; children: React.ReactNode }) {
  return (
    <section className="border-b">
      <header className="px-4 pt-10 pb-6 sm:px-10">
        <h2 className="text-xl font-semibold tracking-tight">{title}</h2>
        <p className="mt-2 max-w-2xl text-sm leading-6 text-muted-foreground">{text}</p>
      </header>
      <div className="grid gap-x-12 gap-y-10 px-4 pb-10 sm:px-10 md:grid-cols-2">{children}</div>
    </section>
  )
}

function MeasureBars({ measure: m, scene }: { measure: Measure; scene: string }) {
  const scaled = (s?: Series): Series | undefined => s && { median: s.median * (m.scale ?? 1), low: s.low * (m.scale ?? 1), high: s.high * (m.scale ?? 1) }
  return (
    <Bars
      title={m.title}
      text={m.text}
      format={m.format}
      higherIsBetter={m.higherIsBetter}
      values={frameworks.map((f) => ({ id: f.id, series: scaled(results.frameworks[f.id]?.scenes[scene]?.[m.id]) }))}
    />
  )
}

/** One measure: a bar for each framework, from zero, on a scale of its own, with its value beside it. */
function Bars({
  title,
  text,
  format,
  higherIsBetter,
  values,
}: {
  title: string
  text: string
  format: Format
  higherIsBetter?: boolean
  values: { id: string; series?: Series }[]
}) {
  const top = Math.max(...values.map((v) => v.series?.median ?? 0), 0)
  return (
    <figure className="min-w-0">
      <figcaption>
        <h3 className="text-sm font-medium">
          {title}
          {higherIsBetter && <span className="ml-2 font-normal text-muted-foreground">higher is better</span>}
        </h3>
        <p className="mt-1 text-[13px] leading-5 text-muted-foreground">{text}</p>
      </figcaption>
      <ul className="mt-3 space-y-0.5">
        {frameworks.map((f) => {
          const s = values.find((v) => v.id === f.id)?.series
          return (
            <li key={f.id} tabIndex={s ? 0 : undefined} className="group relative grid grid-cols-[4.5rem_1fr_5rem] items-center gap-2 rounded-sm py-1 outline-none focus-visible:ring-2 focus-visible:ring-ring">
              <span className="text-[13px] text-muted-foreground">{f.label}</span>
              <span className="h-2.5 min-w-0">
                {s && <span className="block h-full rounded-r-[4px]" style={{ width: top > 0 ? `max(${(s.median / top) * 100}%, 2px)` : "2px", background: f.color }} />}
              </span>
              <span className="text-right font-mono text-[13px] tabular-nums">{s ? formatValue(s.median, format) : "–"}</span>
              {s && s.low !== s.high && (
                <span className="pointer-events-none absolute right-0 bottom-full z-10 mb-1 hidden rounded-md border bg-popover px-2 py-1 text-xs whitespace-nowrap text-popover-foreground shadow-sm group-hover:block group-focus-visible:block">
                  {f.label}: {formatValue(s.low, format)} to {formatValue(s.high, format)} over {results.runs} runs
                </span>
              )}
            </li>
          )
        })}
      </ul>
    </figure>
  )
}

function Setup() {
  const { machine: m, versions: v } = results
  const rows: [string, string][] = [
    ["Measured", new Date(results.date).toLocaleDateString("en", { year: "numeric", month: "long", day: "numeric", timeZone: "UTC" })],
    ["Machine", `${m.cpu}, ${m.cpus} CPUs, ${Math.round(m.memory / 2 ** 30)} GB of memory, ${m.display}`],
    ["System", m.os],
    ["Frameworks", `MyGo ${v.mygo}, GPUI ${v.gpui}, GPUI Fast ${v["gpui-fast"]}, egui ${v.egui}`],
    ["Compilers", `Go ${v.go}, Rust ${v.rustc}`],
    ["Runs", `${results.runs} of each scene, ${results.frames} frames each, frameworks taking turns`],
  ]
  return (
    <dl className="mt-8 grid max-w-2xl grid-cols-[6rem_1fr] gap-x-4 gap-y-1.5 text-sm">
      {rows.map(([k, val]) => (
        <React.Fragment key={k}>
          <dt className="text-muted-foreground">{k}</dt>
          <dd className="min-w-0">{val}</dd>
        </React.Fragment>
      ))}
    </dl>
  )
}
