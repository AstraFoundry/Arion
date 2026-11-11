# Streaming chart

Three waveforms scroll by, each the latest 15,000 samples of a signal
sampled 1,000 times a second, with noise and isolated spikes. Every frame
takes in the samples that arrived since the last, keeps the extremes of
each pixel column with `chart.MinMax`, and strokes them with
`Painter.StrokePolyline`.

```sh
go run ./examples/chart-streaming
go run ./examples/chart-streaming -mode path   # MinMax, then StrokePath
go run ./examples/chart-streaming -mode raw    # every sample, StrokePath
MYGO_FRAME_STATS=all go run ./examples/chart-streaming   # each frame's timing
```
