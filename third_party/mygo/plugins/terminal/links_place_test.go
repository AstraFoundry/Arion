package terminal

import (
	"strings"
	"testing"
)

// Paths as compilers, GitHub and prose in Chinese print them, the place in
// the file written path:line:col however it was printed. Wide characters
// take two columns, as in a terminal.
func TestPathAtPlacesQuotesAndProse(t *testing.T) {
	for _, c := range []struct{ row, at, want string }{
		{"error: src/main.go:42:5: failed", "main", "src/main.go:42:5"},
		{"at (./src/main.ts(12,3))", "main", "./src/main.ts:12:3"},
		{"src/app.ts(12): error TS2304", "app", "src/app.ts:12"},
		{"see ~/work/中文.go#L8C2", "中文", "~/work/中文.go:8:2"},
		{"see src/app.go#L12 for it", "app", "src/app.go:12"},
		{`File "/tmp/my project/中文.go":13:2`, "project", "/tmp/my project/中文.go:13:2"},
		{`open "/tmp/my notes/a.md" now`, "notes", "/tmp/my notes/a.md"},
		{`File '/usr/lib/x y.py', line 7`, "lib", "/usr/lib/x y.py"},
		{"文件 src/日本.go:4 有错", "日本", "src/日本.go:4"},
		{"已保存新版图标 (assets/branding/terminal-icon.png)，使用新版", "terminal", "assets/branding/terminal-icon.png"},
		{"terminal-icon.prompt.txt)也已保存。", "prompt", "terminal-icon.prompt.txt"},
		{"已保存（assets/branding/中文.png），继续", "中文", "assets/branding/中文.png"},
		{"见：src/main.go:42:5，错误", "main", "src/main.go:42:5"},
		{"已保存 (assets/icon_(copy).png)，继续", "icon", "assets/icon_(copy).png"},
		{`文件 "assets/图标（新版），副本.png"，已保存`, "新版", "assets/图标（新版），副本.png"},
		{"(../README.md),", "README", "../README.md"},
		{"Makefile:4", "Make", "Makefile:4"},
		{"目录 /tmp/project/", "project", "/tmp/project/"},
		{"main.go:0", "main", "main.go"},
		{"src/a.go:12:code", "a.go", "src/a.go:12"},
		{"--out=build/app.js", "app", "build/app.js"},
		{"PATH=/usr/bin:/sbin", "usr", "/usr/bin"},
		{"PATH=/usr/bin:/sbin", "sbin", "/sbin"},
		{"PATH=/usr/bin:/sbin", "PATH", ""},
		{"don't open src/a.go, it's broken", "src", "src/a.go"},
		{"don't open src/a.go, it's broken", "open", ""},
		{"don't open src/a.go, it's broken", "broken", ""},
		{`say " a/b.go " twice`, "a/b", "a/b.go"},
		{"meet at 12:30 today", "12", ""},
		{"and / or", "/", ""},
		{"已保存 (assets/x.png)，继续", "保存", ""},
		{"已保存 (assets/x.png)，继续", "，", ""},
	} {
		text := []rune(c.row)
		cols := make([]int, len(text))
		x := 0
		for i, r := range text {
			cols[i] = x
			x++
			if r > 0x2fff {
				x++
			}
		}
		col := cols[len([]rune(c.row[:strings.Index(c.row, c.at)]))]
		if got, _, _ := pathAt(text, cols, col); got != c.want {
			t.Errorf("pathAt(%q at %q) = %q, want %q", c.row, c.at, got, c.want)
		}
	}
}
