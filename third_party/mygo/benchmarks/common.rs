//! What the Rust apps of the comparison share: the scenes' text and the
//! recording of frames, the same as MyGo's app (benchmarks/mygo/main.go).

use std::time::{Duration, Instant};

const WORDS: [&str; 32] = [
    "window", "button", "layout", "render", "scroll", "update", "shadow", "cursor", "memory",
    "thread", "signal", "bitmap", "stroke", "column", "editor", "kernel", "packet", "vector",
    "syntax", "buffer", "format", "cipher", "socket", "module", "branch", "commit", "server",
    "client", "screen", "pointer", "display", "palette",
];

/// The text of a seed, the same in every framework's app.
pub fn words(seed: usize, n: usize) -> String {
    let mut x = seed as u32;
    let mut out = String::new();
    for i in 0..n {
        x = x.wrapping_mul(1103515245).wrapping_add(12345);
        if i > 0 {
            out.push(' ');
        }
        out.push_str(WORDS[((x >> 16) % WORDS.len() as u32) as usize]);
    }
    out
}

pub fn cell_value(cell: usize, frame: usize) -> f64 {
    ((cell * 7919 + frame * 31) % 100_000) as f64 / 100.0
}

/// The background of a cell, as (r, g, b).
pub fn cell_color(v: f64) -> (u8, u8, u8) {
    if v < 333.0 {
        (0xf6, 0xc9, 0xc4)
    } else if v < 666.0 {
        (0xf4, 0xe8, 0xb8)
    } else {
        (0xc6, 0xe9, 0xc8)
    }
}

pub fn document_width(frame: usize) -> f32 {
    (600.0 + 150.0 * (frame as f64 / 30.0).sin()).round() as f32
}

fn env(name: &str, fallback: usize) -> usize {
    std::env::var(name)
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(fallback)
}

/// Times frames and says on standard output how the run goes.
pub struct Recorder {
    warmup: usize,
    frames: usize,
    seen: usize,
    last: Instant,
    started: Instant,
    intervals: Vec<Duration>,
}

impl Recorder {
    pub fn new() -> Self {
        let now = Instant::now();
        Recorder {
            warmup: env("BENCH_WARMUP", 120),
            frames: env("BENCH_FRAMES", 600),
            seen: 0,
            last: now,
            started: now,
            intervals: Vec::new(),
        }
    }

    /// Counts a frame, and returns whether the run wants another.
    pub fn tick(&mut self) -> bool {
        let now = Instant::now();
        self.seen += 1;
        if self.seen == self.warmup {
            println!("start");
            self.started = now;
        } else if self.seen > self.warmup && self.seen <= self.warmup + self.frames {
            self.intervals.push(now - self.last);
            if self.intervals.len() == self.frames {
                self.intervals.sort();
                println!(
                    r#"done {{"frames":{},"wallUs":{},"p50Us":{},"p99Us":{},"maxUs":{}}}"#,
                    self.frames,
                    (now - self.started).as_micros(),
                    self.intervals[self.frames / 2].as_micros(),
                    self.intervals[self.frames * 99 / 100].as_micros(),
                    self.intervals[self.frames - 1].as_micros(),
                );
            }
        }
        self.last = now;
        self.seen <= self.warmup + self.frames
    }
}
