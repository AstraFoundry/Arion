//go:build !js

package gpu

import "testing"

func TestSourceSum(t *testing.T) {
	lf, crlf := "float4 ps() {\n\treturn 1;\n}\n", "float4 ps() {\r\n\treturn 1;\r\n}\r\n"
	if SourceSum(lf) != SourceSum(crlf) {
		t.Error("a checkout with CRLF line endings has another sum")
	}
	if SourceSum(lf) == SourceSum(lf+" ") || len(SourceSum(lf)) != 64 {
		t.Error("the sum does not tell sources apart")
	}
}
