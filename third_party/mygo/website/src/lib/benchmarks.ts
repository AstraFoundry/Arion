import measured from "../../../benchmarks/results.json"

import { site } from "@/lib/site"

/** benchmarks/results.json, which benchmarks/run writes. */
export interface Results {
  date: string
  commit: string
  machine: { os: string; model: string; cpu: string; cpus: number; memory: number; display: string }
  versions: Record<string, string>
  runs: number
  frames: number
  frameworks: Record<string, { size: number; scenes: Record<string, Record<string, Series>> }>
}

/** A measure's median over the runs, and its lowest and highest. */
export interface Series {
  median: number
  low: number
  high: number
}

export const results: Results = measured

export const sourceUrl = `${site.repo}/tree/main/benchmarks`

/** Always in this order, each with its own color wherever it shows. */
export const frameworks = [
  { id: "mygo", label: "MyGo", url: site.repo, color: "var(--bench-mygo)" },
  { id: "gpui", label: "GPUI", url: "https://www.gpui.rs", color: "var(--bench-gpui)" },
  { id: "gpui-fast", label: "GPUI Fast", url: "https://github.com/longbridge/gpui-fast", color: "var(--bench-gpui-fast)" },
  { id: "egui", label: "egui", url: "https://github.com/emilk/egui", color: "var(--bench-egui)" },
] as const

export type Format = "ms" | "bytes" | "fps" | "percent"

export interface Measure {
  id: string
  title: string
  text: string
  format: Format
  /** Frame rate is the one where more is better. */
  higherIsBetter?: boolean
  /** Scales the raw value to the unit its format shows. */
  scale?: number
}

export interface Scene {
  id: string
  title: string
  text: string
  measures: Measure[]
}

const framePacing: Measure[] = [
  { id: "cpuMsPerFrame", title: "CPU per frame", text: "The time the whole process spent on the CPU, its threads and the graphics driver's, for each frame.", format: "ms" },
  { id: "fps", title: "Frame rate", text: "Frames a second. It stops at the display's refresh rate.", format: "fps", higherIsBetter: true },
  { id: "p99Ms", title: "Slowest frames", text: "The 99th percentile of the time between frames: what the worst 1% of frames look like.", format: "ms" },
  { id: "peakMemory", title: "Peak memory", text: "The highest physical footprint of the process, as Activity Monitor shows it.", format: "bytes" },
]

export const scenes: Scene[] = [
  {
    id: "list",
    title: "Scrolling a long list",
    text: "A log viewer: 100,000 virtualized rows of three texts each, scrolling 8 DIPs every frame.",
    measures: framePacing,
  },
  {
    id: "grid",
    title: "A live dashboard",
    text: "600 cells, each with its own background and a number that changes every frame.",
    measures: framePacing,
  },
  {
    id: "document",
    title: "Resizing a document",
    text: "40 paragraphs, about 20,000 characters, wrapped again to a width that changes every frame.",
    measures: framePacing,
  },
]

/** The counter's, where the app starts and then does nothing. */
export const startMeasures: Measure[] = [
  { id: "startupMs", title: "Startup", text: "From starting the process until its first frame is drawn and its window is on screen.", format: "ms" },
  { id: "idleMemory", title: "Memory at rest", text: "The physical footprint of the process 5 seconds after its first frame.", format: "bytes" },
  { id: "idleCpuMsPerSec", title: "CPU at rest", text: "The share of a core the process takes over the next 5 seconds, drawing nothing.", format: "percent", scale: 0.1 },
]

export function formatValue(value: number, format: Format): string {
  switch (format) {
    case "ms":
      return value >= 100 ? `${value.toFixed(0)} ms` : `${value.toFixed(value >= 10 ? 1 : 2)} ms`
    case "bytes":
      return `${(value / 1e6).toFixed(value >= 1e8 ? 0 : 1)} MB`
    case "fps":
      return `${value.toFixed(1)} fps`
    case "percent":
      return `${value.toFixed(2)}%`
  }
}
