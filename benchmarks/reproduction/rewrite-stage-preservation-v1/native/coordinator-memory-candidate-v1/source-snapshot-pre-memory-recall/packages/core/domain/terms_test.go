package domain

import (
	"strings"
	"testing"
)

func TestContextTerms(t *testing.T) {
	c := RewriteContext{
		Screen:  "飞书 — 研发群\n许晗 发布会的合同发你了\nThe build uses ValidateJWT and PO-2024-0815\n周六去「栖云山房」喝茶\nFrom: Saoirse O'Neill\nThanks, Can you check?",
		History: []string{"Moving the VPC module to Terragrunt today."},
	}
	got := strings.Join(ContextTerms(c, "许涵，合同我签好了，ValidateJWT 先不动。"), ";")
	want := "VPC;Terragrunt;飞书;许晗;PO-2024-0815;栖云山房;Saoirse;O'Neill"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if got := strings.Join(ContextTerms(RewriteContext{Screen: "Nguyen Thao\nUploaded the new onboarding mocks"}, "x"), ";"); got != "Nguyen;Thao" {
		t.Fatalf("short name line: %q", got)
	}
	if terms := ContextTerms(RewriteContext{}, "x"); len(terms) != 0 {
		t.Fatalf("empty context gave %q", terms)
	}
}
