package runtime

import (
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"testing"
)

// Views map State.Account.ErrorCode through the copy catalog, so every code
// the account service declares must be one the runtime boundary lists.
func TestAccountCodesMatchService(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "../../../internal/account/errors.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var service []string
	ast.Inspect(file, func(node ast.Node) bool {
		value, ok := node.(*ast.ValueSpec)
		if !ok || len(value.Values) != 1 {
			return true
		}
		if kind, ok := value.Type.(*ast.Ident); !ok || kind.Name != "Code" {
			return true
		}
		literal, ok := value.Values[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		code, err := strconv.Unquote(literal.Value)
		if err != nil {
			t.Fatal(err)
		}
		service = append(service, code)
		return true
	})
	declared := slices.Clone(AccountCodes)
	slices.Sort(service)
	slices.Sort(declared)
	if len(service) == 0 || !slices.Equal(service, declared) {
		t.Fatalf("account codes differ:\nservice %v\nruntime %v", service, declared)
	}
}
