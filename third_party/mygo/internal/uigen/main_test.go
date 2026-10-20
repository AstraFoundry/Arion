package main

import (
	"go/parser"
	"go/token"
	"testing"
)

func TestConvertArgCallbacks(t *testing.T) {
	for _, tc := range []struct{ name, signature, want string }{
		{"empty", "func()", "fn"},
		{"plain", "func(p *Painter, r Rect)", "fn"},
		{"unnamed", "func(string, int) bool", "fn"},
		{"grouped", "func(a, b int) (ok bool, err error)", "fn"},
		{"generic", "func(K) []K", "fn"},
		{"variadic", "func(prefix string, values ...int) (int, error)", "fn"},
		{"namedType", "InputHandler", "fn"},
		{"context", "func(c *context)", "func(c *context) {fn(makeContext(c))}"},
		{"node", "func(n *node)", "func(n *node) {fn(wrapElement(n))}"},
		{"mixed", "func(c *context, n *node, i int) bool", "func(c *context,n *node,i int) bool {return fn(makeContext(c),wrapElement(n),i)}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := token.NewFileSet()
			typ, err := parser.ParseExprFrom(fs, "", tc.signature, 0)
			if err != nil {
				t.Fatal(err)
			}
			if got := convertArg(fs, typ, "fn", "ctx"); got != tc.want {
				t.Fatalf("convertArg(%s) = %s, want %s", tc.signature, got, tc.want)
			}
		})
	}
}
