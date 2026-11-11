# Comparison with GPUI, GPUI Fast and egui

The same small apps written with MyGo's native UI, [GPUI](https://www.gpui.rs),
[GPUI Fast](https://github.com/longbridge/gpui-fast) and
[egui](https://github.com/emilk/egui) (through eframe, with its default wgpu
renderer), each through the usual way of its framework. GPUI Fast's list
scrolls from outside its view, as a wheel does, for its cached rows to apply. The website's
[/benchmarks](https://mygo.egoist.dev/benchmarks) page shows `results.json`.

```sh
go run ./benchmarks/run -out benchmarks/results.json   # opens windows for about 15 minutes
```

Run it on a Mac with a real display and GPU, and commit the file.
`.github/workflows/comparison.yml` only checks that the apps still build and
run: GitHub's runners are virtual machines, whose results would not compare.

## Scenes

An app takes its scene as an argument. Every app shows the same text, from
the same generator (`words`, `cell_value`), in windows of the same size.

| Scene      | Window  | What it does                                                                                   |
| ---------- | ------- | ---------------------------------------------------------------------------------------------- |
| `counter`  | 360×260 | A number and three buttons, then nothing: startup, memory and CPU at rest.                     |
| `list`     | 800×600 | A virtualized list of 100,000 rows of three texts, scrolling 8 DIPs a frame, as a log viewer.  |
| `grid`     | 800×600 | 600 cells, each with its own background and a number changing every frame, as a dashboard.     |
| `document` | 800×600 | 40 paragraphs (about 20,000 characters) wrapped to a width changing every frame, as a window resized. |

Every app has a main menu, as an app of each framework would: GPUI and GPUI
Fast, which set none by themselves, are given one. The runner turns off AppKit's debug
menu for each app, which a Mac with `_NS_4445425547` set builds in each app that
sets a menu, taking some 70 ms.

## What is measured

From outside the app, with `proc_pid_rusage`, so that the whole process counts,
the threads of the framework and the graphics driver included:

- **CPU per frame**, the process's user and system time over the frames
  measured, divided by them. GPU work is not included.
- **Peak memory**, the highest physical footprint of the process, as Activity
  Monitor shows it.
- **Idle memory and CPU** of the counter, 5 seconds after its first frame.

And from the app, which says these on standard output:

- **Startup**, from starting the process until its first frame is drawn, as
  soon as each framework can tell, and its window is on screen, as the window
  server tells, whichever comes last. eframe, GPUI and GPUI Fast tell at their
  next frame.
- **Frame rate** and the **median and 99th percentile of the time between
  frames**, which show the frames that missed the display's refresh.

Each scene runs 5 times, the frameworks taking turns, and the results are
medians. CPU per frame has moved by as much as a third between full runs of
the same build, more than the range of one run shows, for reasons not found:
compare runs before reading much into a small gap. The first 120 frames of a
run are not measured. **Binary size** is
that of a release build: for MyGo as `mygo build` compiles it, and for the
others with fat LTO, one codegen unit and symbols stripped.
