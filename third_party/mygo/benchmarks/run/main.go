// Command run builds the apps of the comparison (see benchmarks/README.md),
// runs each scene of each app in turn and writes what it measures as JSON.
// It needs a desktop session, and windows open while it runs.
//
//	go run ./benchmarks/run [-runs 5] [-frames 600] [-warmup 120] [-out results.json]
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

// frameworks are measured in this order within each round, so that a machine
// slowing down over a run weighs on them alike.
var frameworks = []string{"mygo", "gpui", "gpui-fast", "egui"}

// scenes that draw every frame, as opposed to the counter.
var animated = []string{"list", "grid", "document"}

// Series is a measure's median over the runs, and its lowest and highest.
type Series struct {
	Median float64 `json:"median"`
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
}

func seriesOf(samples []float64) Series {
	s := slices.Clone(samples)
	slices.Sort(s)
	m := s[len(s)/2]
	if len(s)%2 == 0 {
		m = (s[len(s)/2-1] + s[len(s)/2]) / 2
	}
	return Series{Median: m, Low: s[0], High: s[len(s)-1]}
}

// Results is the file the website reads.
type Results struct {
	Date       string            `json:"date"`
	Commit     string            `json:"commit"`
	Machine    Machine           `json:"machine"`
	Versions   map[string]string `json:"versions"`
	Runs       int               `json:"runs"`
	Frames     int               `json:"frames"`
	Frameworks map[string]*App   `json:"frameworks"`
}

// Machine is where the results come from.
type Machine struct {
	OS      string `json:"os"`
	Model   string `json:"model"`
	CPU     string `json:"cpu"`
	CPUs    int    `json:"cpus"`
	Memory  uint64 `json:"memory"`
	Display string `json:"display"`
}

// App is what one framework's app measured.
type App struct {
	Size   uint64                       `json:"size"`
	Scenes map[string]map[string]Series `json:"scenes"`
}

func main() {
	runs := flag.Int("runs", 5, "how many times each scene runs")
	frames := flag.Int("frames", 600, "frames measured in a run of an animated scene")
	warmup := flag.Int("warmup", 120, "frames not measured at the start of a run")
	idle := flag.Duration("idle", 5*time.Second, "how long the counter settles before it is measured, and how long its idle CPU is measured")
	out := flag.String("out", "", "the file to write, or standard output")
	flag.Parse()
	log.SetFlags(0)
	log.SetPrefix("run: ")

	root, err := moduleRoot()
	if err != nil {
		log.Fatal(err)
	}
	bin, err := os.MkdirTemp("", "mygo-comparison-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(bin)

	res := &Results{
		Date:       time.Now().UTC().Format(time.RFC3339),
		Commit:     output(root, "git", "rev-parse", "HEAD"),
		Machine:    machine(),
		Versions:   versions(root),
		Runs:       *runs,
		Frames:     *frames,
		Frameworks: map[string]*App{},
	}
	paths := map[string]string{}
	for _, f := range frameworks {
		log.Printf("building %s", f)
		path, err := build(root, bin, f)
		if err != nil {
			log.Fatalf("building %s: %v", f, err)
		}
		info, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}
		paths[f] = path
		res.Frameworks[f] = &App{Size: uint64(info.Size()), Scenes: map[string]map[string]Series{}}
	}

	// The system checks a program the first time it starts, and its pages are
	// not in memory yet: one launch of each, not measured.
	for _, f := range frameworks {
		p, err := start(paths[f], "counter")
		if err != nil {
			log.Fatal(err)
		}
		if _, err := p.wait("ready"); err != nil {
			log.Fatalf("%s: %v", f, err)
		}
		p.stop()
	}

	samples := map[string]map[string]map[string][]float64{}
	add := func(f, scene string, m map[string]float64) {
		if samples[f] == nil {
			samples[f] = map[string]map[string][]float64{}
		}
		if samples[f][scene] == nil {
			samples[f][scene] = map[string][]float64{}
		}
		for k, v := range m {
			samples[f][scene][k] = append(samples[f][scene][k], v)
		}
	}
	for round := range *runs {
		for _, f := range frameworks {
			m, err := runCounter(paths[f], *idle)
			if err != nil {
				log.Fatalf("%s counter: %v", f, err)
			}
			log.Printf("round %d %s counter: %v", round+1, f, m)
			add(f, "counter", m)
		}
		for _, scene := range animated {
			for _, f := range frameworks {
				m, err := runAnimated(paths[f], scene, *warmup, *frames)
				if err != nil {
					log.Fatalf("%s %s: %v", f, scene, err)
				}
				log.Printf("round %d %s %s: %v", round+1, f, scene, m)
				add(f, scene, m)
			}
		}
	}
	for f, scenes := range samples {
		for scene, measures := range scenes {
			res.Frameworks[f].Scenes[scene] = map[string]Series{}
			for k, v := range measures {
				res.Frameworks[f].Scenes[scene][k] = seriesOf(v)
			}
		}
	}

	b, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if *out == "" {
		fmt.Println(string(b))
	} else if err := os.WriteFile(*out, append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
}

func moduleRoot() (string, error) {
	b, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return "", err
	}
	return filepath.Dir(strings.TrimSpace(string(b))), nil
}

// build compiles a framework's app as it ships: MyGo's as mygo build and
// scripts/bench.ts do, the others with the profile in their Cargo.toml.
func build(root, bin, f string) (string, error) {
	path := filepath.Join(bin, f)
	var cmd *exec.Cmd
	if f == "mygo" {
		cmd = exec.Command("go", "build", "-trimpath", "-tags", "mygo_noinspector",
			"-ldflags", "-s -w -X github.com/egoist/mygo.production=1", "-o", path, "./benchmarks/mygo")
		cmd.Dir = root
	} else {
		cmd = exec.Command("cargo", "build", "--release", "--locked")
		cmd.Dir = filepath.Join(root, "benchmarks", f)
	}
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	if f != "mygo" {
		dir := filepath.Join(root, "benchmarks", f, "target", "release", "bench-"+f)
		b, err := os.ReadFile(dir)
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(path, b, 0o755); err != nil {
			return "", err
		}
	}
	return path, nil
}

// process is a running app, with the lines it says.
type process struct {
	cmd   *exec.Cmd
	lines chan string
}

func start(path, scene string, env ...string) (*process, error) {
	// A Mac with AppKit's debug menu turned on (the default _NS_4445425547,
	// "DEBUG") spends some 70 ms building it in each app that sets a main menu,
	// which the apps' users would not.
	cmd := exec.Command(path, scene, "-_NS_4445425547", "NO")
	cmd.Env = append(os.Environ(), env...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	p := &process{cmd: cmd, lines: make(chan string, 16)}
	go func() {
		defer close(p.lines)
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			p.lines <- s.Text()
		}
	}()
	return p, nil
}

// wait returns the next line that starts with prefix. Its guard only turns an
// app that hung or crashed into an error.
func (p *process) wait(prefix string) (string, error) {
	guard := time.After(3 * time.Minute)
	for {
		select {
		case line, ok := <-p.lines:
			if !ok {
				return "", fmt.Errorf("exited before saying %q", prefix)
			}
			if strings.HasPrefix(line, prefix) {
				return line, nil
			}
		case <-guard:
			return "", fmt.Errorf("did not say %q", prefix)
		}
	}
}

func (p *process) stop() {
	p.cmd.Process.Kill()
	p.cmd.Wait()
}

// runCounter starts the counter, and measures how long until its first frame
// is drawn and its window is on screen, whichever comes last, and its memory
// and CPU once idle.
func runCounter(path string, idle time.Duration) (map[string]float64, error) {
	began := time.Now()
	p, err := start(path, "counter")
	if err != nil {
		return nil, err
	}
	defer p.stop()
	shown := make(chan time.Duration, 1)
	go func() {
		// Its guard only ends the wait of an app that never shows a window.
		for time.Since(began) < time.Minute {
			if windowOnScreen(p.cmd.Process.Pid) {
				shown <- time.Since(began)
				return
			}
			time.Sleep(2 * time.Millisecond)
		}
		shown <- 0
	}()
	if _, err := p.wait("ready"); err != nil {
		return nil, err
	}
	drawn := time.Since(began)
	onScreen := <-shown
	if onScreen == 0 {
		return nil, errors.New("no window came on screen")
	}
	startup := max(drawn, onScreen)
	time.Sleep(idle)
	before, err := usageOf(p.cmd.Process.Pid)
	if err != nil {
		return nil, err
	}
	time.Sleep(idle)
	after, err := usageOf(p.cmd.Process.Pid)
	if err != nil {
		return nil, err
	}
	return map[string]float64{
		"startupMs":       float64(startup) / float64(time.Millisecond),
		"idleMemory":      float64(after.footprint),
		"idleCpuMsPerSec": float64(after.cpu-before.cpu) / float64(time.Millisecond) / idle.Seconds(),
	}, nil
}

// runAnimated runs a scene that draws every frame, and measures the CPU the
// whole process took for each frame, how frames were paced, and its memory.
func runAnimated(path, scene string, warmup, frames int) (map[string]float64, error) {
	p, err := start(path, scene, "BENCH_WARMUP="+strconv.Itoa(warmup), "BENCH_FRAMES="+strconv.Itoa(frames))
	if err != nil {
		return nil, err
	}
	defer p.stop()
	if _, err := p.wait("start"); err != nil {
		return nil, err
	}
	before, err := usageOf(p.cmd.Process.Pid)
	if err != nil {
		return nil, err
	}
	line, err := p.wait("done ")
	if err != nil {
		return nil, err
	}
	after, err := usageOf(p.cmd.Process.Pid)
	if err != nil {
		return nil, err
	}
	var done struct {
		Frames int     `json:"frames"`
		WallUs float64 `json:"wallUs"`
		P50Us  float64 `json:"p50Us"`
		P99Us  float64 `json:"p99Us"`
		MaxUs  float64 `json:"maxUs"`
	}
	if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "done ")), &done); err != nil {
		return nil, err
	}
	if done.Frames != frames {
		return nil, errors.New("the app drew another number of frames")
	}
	return map[string]float64{
		"cpuMsPerFrame": float64(after.cpu-before.cpu) / float64(time.Millisecond) / float64(frames),
		"fps":           float64(frames) / (done.WallUs / 1e6),
		"p50Ms":         done.P50Us / 1e3,
		"p99Ms":         done.P99Us / 1e3,
		"maxMs":         done.MaxUs / 1e3,
		"peakMemory":    float64(after.peak),
	}, nil
}

func output(dir string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	b, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// versions are those of the frameworks as the apps pin them, and of the
// compilers.
func versions(root string) map[string]string {
	v := map[string]string{
		"go":    strings.TrimPrefix(strings.Fields(output(root, "go", "version"))[2], "go"),
		"rustc": strings.Fields(output(root, "rustc", "--version"))[1],
	}
	if b, err := os.ReadFile(filepath.Join(root, "mygo.go")); err == nil {
		if m := regexp.MustCompile(`const Version = "([^"]+)"`).FindSubmatch(b); m != nil {
			v["mygo"] = string(m[1])
		}
	}
	for f, crate := range map[string]string{"gpui": "gpui", "gpui-fast": "gpui-fast", "egui": "eframe"} {
		b, err := os.ReadFile(filepath.Join(root, "benchmarks", f, "Cargo.lock"))
		if err != nil {
			log.Fatal(err)
		}
		m := regexp.MustCompile(`name = "` + crate + `"\nversion = "([^"]+)"`).FindSubmatch(b)
		if m == nil {
			log.Fatalf("no %s in %s's Cargo.lock", crate, f)
		}
		v[f] = string(m[1])
	}
	return v
}

func machine() Machine {
	mem, _ := strconv.ParseUint(output("", "sysctl", "-n", "hw.memsize"), 10, 64)
	return Machine{
		OS:      output("", "sw_vers", "-productName") + " " + output("", "sw_vers", "-productVersion"),
		Model:   output("", "sysctl", "-n", "hw.model"),
		CPU:     output("", "sysctl", "-n", "machdep.cpu.brand_string"),
		CPUs:    runtime.NumCPU(),
		Memory:  mem,
		Display: display(),
	}
}

// display returns the resolution of the display, as System Profiler says it.
func display() string {
	for _, line := range strings.Split(output("", "system_profiler", "SPDisplaysDataType"), "\n") {
		if s := strings.TrimSpace(line); strings.HasPrefix(s, "Resolution:") {
			return strings.TrimSpace(strings.TrimPrefix(s, "Resolution:"))
		}
	}
	return ""
}
