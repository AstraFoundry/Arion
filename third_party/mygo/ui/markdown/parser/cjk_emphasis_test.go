package parser

import "testing"

// Emphasis closed by CJK punctuation and followed by CJK text, as
// Chinese writes it without spaces.
func TestCJKEmphasis(t *testing.T) {
	for src, want := range map[string]string{
		"**五种方式都能承载。**下面是判断":         "<p><strong>五种方式都能承载。</strong>下面是判断</p>\n",
		"**已有可靠内网：直接 HTTP。**接入成本最低。": "<p><strong>已有可靠内网：直接 HTTP。</strong>接入成本最低。</p>\n",
		"针对你强调的**“简单”**，我会这样选":       "<p>针对你强调的<strong>“简单”</strong>，我会这样选</p>\n",
		"~~删除。~~保留":       "<p><del>删除。</del>保留</p>\n",
		"snake_case_name": "<p>snake_case_name</p>\n",
		"a **b** c":       "<p>a <strong>b</strong> c</p>\n",
		// _ and a single ~ keep CommonMark's rule next to CJK text.
		"中文__粗体。__后文":      "<p>中文__粗体。__后文</p>\n",
		"第1章~“附录”和“序”~第3章": "<p>第1章~“附录”和“序”~第3章</p>\n",
		// Latin text with curly quotes reads as before.
		"He said “*hello*” to me.":       "<p>He said “<em>hello</em>” to me.</p>\n",
		"**Alice**’s and **Bob**’s toys": "<p><strong>Alice</strong>’s and <strong>Bob</strong>’s toys</p>\n",
	} {
		if got := render(src, Default); got != want {
			t.Errorf("%q: got %q, want %q", src, got, want)
		}
	}
}
