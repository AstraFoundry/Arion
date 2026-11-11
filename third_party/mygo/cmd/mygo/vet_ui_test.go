package main

import (
	"os/exec"
	"strings"
	"testing"
)

// TestVetUICgo checks the lifetimes of packages with cgo files (#378):
// what those files declare is defined, and they are checked too.
func TestVetUICgo(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles programs")
	}
	if out, err := exec.Command("go", "env", "CGO_ENABLED", "CC").Output(); err != nil {
		t.Fatal(err)
	} else if env := strings.Fields(string(out)); len(env) < 2 || env[0] != "1" {
		t.Skip("cgo is disabled")
	} else if _, err := exec.LookPath(env[1]); err != nil {
		t.Skip("no C compiler")
	} else if out, err := exec.Command("go", "build", "runtime/cgo").CombinedOutput(); err != nil {
		// A compiler for another architecture, as the x64 GCC of a
		// Windows ARM64 runner, can't build cgo.
		t.Skipf("cgo does not build: %v\n%s", err, out)
	}
	answer := "package main\n\n// int answer(void) { return 42; }\nimport \"C\"\n\nfunc answer() int { return int(C.answer()) }\n"
	for _, tt := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{name: "clean", files: map[string]string{
			"main.go":   "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(answer()) }\n",
			"answer.go": answer,
		}},
		{name: "stored element", files: map[string]string{
			"main.go":   "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(answer(), saved) }\n",
			"answer.go": strings.Replace(answer, "import \"C\"\n", "import \"C\"\n\nimport \"github.com/egoist/mygo/ui\"\n\nvar saved ui.Element\n", 1),
		}, want: "1 native UI lifetime issue(s)"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := ""
			if err := runVetUI([]string{testModule(t, tt.files)}); err != nil {
				got = err.Error()
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
