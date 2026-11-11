package i18ngen

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"testing/fstest"
)

// Generated accessors must keep distinct placeholders distinct even when
// their names need escaping in Go or TypeScript.
func TestParameterNamesCompile(t *testing.T) {
	const catalog = `{"example":"{0} {arg0} {type} {type_} {default} {default_} {_} {__}"}`
	c, ds, err := Load(fstest.MapFS{"en.json": {Data: []byte(catalog)}}, Options{})
	if err != nil || ds.Errors() != 0 {
		t.Fatalf("load: %v, %v", err, ds)
	}
	dir := t.TempDir()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	src, err := Go(c, "example")
	if err != nil {
		t.Fatal(err)
	}
	write("catalog_gen.go", src)
	write("en.json", []byte(catalog))
	write("go.mod", []byte(fmt.Sprintf("module example\n\ngo 1.27\nrequire github.com/egoist/mygo v0.0.0\nreplace github.com/egoist/mygo => %q\n", filepath.ToSlash(root))))
	write("catalog_test.go", []byte(`package example
import "testing"
func TestMessage(t *testing.T) {
 if got := Example("a", "b", "c", "d", "e", "f", "g", "h"); got != "a b c d e f g h" { t.Fatal(got) }
}`))
	t.Run("Go", func(t *testing.T) {
		cmd := exec.Command("go", "test", "-mod=mod", ".")
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("generated Go: %v\n%s", err, out)
		}
	})
	t.Run("TypeScript", func(t *testing.T) {
		tsc := filepath.Join(root, "node_modules", ".bin", "tsc")
		if _, err := os.Stat(tsc); err != nil {
			t.Skip("tsc not installed; run bun install")
		}
		src, err := TS(c)
		if err != nil {
			t.Fatal(err)
		}
		write("catalog.ts", src)
		write("runtime.d.ts", []byte(`declare module "mygo-runtime/i18n" {
export function createCatalog<T>(data: unknown): { messages(): T; messagesIn(locale: string): T; languages: string[] };
}`))
		cmd := exec.Command(tsc, "--noEmit", "--strict", "catalog.ts", "runtime.d.ts")
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("generated TypeScript: %v\n%s", err, out)
		}
	})
}
