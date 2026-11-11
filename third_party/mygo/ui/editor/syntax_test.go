package editor

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"
)

var tokenNames = [...]string{"text", "kw", "type", "fn", "var", "prop", "const", "num", "str", "esc", "re", "comment", "op", "punct",
	"tag", "attr", "meta", "heading", "em", "strong", "link", "code", "ins", "del", "invalid"}

// lexLines tokenizes lines and writes each token as kind(text), leaving
// out spaces and plain variables.
func lexLines(t Tokenizer, lines ...string) string {
	var b strings.Builder
	var state uint64
	for i, line := range lines {
		if i > 0 {
			b.WriteString(" / ")
		}
		var toks []Token
		toks, state = t.Tokenize(line, state, nil)
		from := 0
		for _, tk := range toks {
			if tk.End <= from || tk.End > len(line) {
				panic(fmt.Sprintf("token %+v after %d in %q", tk, from, line))
			}
			s := line[from:tk.End]
			from = tk.End
			if strings.TrimSpace(s) == "" || tk.Kind == TokenText && len(strings.Fields(s)) == 0 {
				continue
			}
			fmt.Fprintf(&b, "%s(%s)", tokenNames[tk.Kind], strings.TrimSpace(s))
		}
		if from != len(line) {
			panic(fmt.Sprintf("tokens end at %d of %q", from, line))
		}
	}
	return b.String()
}

func TestTokenizers(t *testing.T) {
	for _, c := range []struct {
		lang  string
		lines []string
		want  string
	}{
		{"go", []string{`func main() { fmt.Println("hi\n", 42) } // done`},
			`kw(func)fn(main)punct(())punct({)var(fmt)punct(.)fn(Println)punct(()str("hi)esc(\n)str(")punct(,)num(42)punct())punct(})comment(// done)`},
		{"go", []string{"x := `raw", "still` + 1 /* a", "b */"},
			"var(x)op(:=)str(`raw) / str(still`)op(+)num(1)comment(/* a) / comment(b */)"},
		{"ts", []string{"const s = `a${b + `c${d}`}e`"},
			"kw(const)var(s)op(=)str(`a)esc(${)var(b)op(+)str(`c)esc(${)var(d)esc(})str(`)esc(})str(e`)"},
		{"js", []string{"x = /a[/]b/g.test(y)"},
			"var(x)op(=)re(/a[/]b/g)punct(.)fn(test)punct(()var(y)punct())"},
		{"python", []string{`@app.route("/")`, `def f(x): return f"{x!r}" + r'\d'  # c`},
			`meta(@app.route)punct(()str("/")punct()) / kw(def)fn(f)punct(()var(x)punct())op(:)kw(return)str(f")esc({)var(x)op(!)var(r)esc(})str(")op(+)str(r'\d')comment(# c)`},
		// Each of Python's f-strings, the single-quoted ones last of all.
		{"python", []string{`x = f'{a}' + F'{a}{b}' + rf'{c}' + Fr"""{d}"""`},
			`var(x)op(=)str(f')esc({)var(a)esc(})str(')op(+)str(F')esc({)var(a)esc(}{)var(b)esc(})str(')op(+)str(rf')esc({)var(c)esc(})str(')op(+)str(Fr""")esc({)var(d)esc(})str(""")`},
		{"json", []string{`{"a": [1, true, "b"]}`},
			`punct({)prop("a")op(:)punct([)num(1)punct(,)const(true)punct(,)str("b")punct(]})`},
		{"json", []string{`{"a\nb" : "c\nd", "e":{"f":1}}`},
			`punct({)prop("a\nb")op(:)str("c)esc(\n)str(d")punct(,)prop("e")op(:)punct({)prop("f")op(:)num(1)punct(}})`},
		{"rust", []string{"/* a /* b */ c */ fn x() {}"},
			"comment(/* a /* b */ c */)kw(fn)fn(x)punct(())punct({})"},
		{"c", []string{"#include <stdio.h>", "int x = 0x1fu;"},
			"meta(#include)op(<)var(stdio)punct(.)prop(h)op(>) / type(int)var(x)op(=)num(0x1fu)punct(;)"},
		{"css", []string{".a:hover, #b { color: red; margin: -4px; }"},
			"type(.a)kw(:hover)punct(,)type(#b)punct({)prop(color)punct(:)const(red)punct(;)prop(margin)punct(:)num(-4px)punct(;)punct(})"},
		{"html", []string{`<div class="x">a &amp; b<script>let y = 1</script>`},
			`punct(<)tag(div)attr(class)op(=)str("x")punct(>)text(a)esc(&amp;)text(b)punct(<)tag(script)punct(>)kw(let)var(y)op(=)num(1)punct(</)tag(script)punct(>)`},
		{"markdown", []string{"# Title", "Some *em* and **strong** `code` [link](url)", "```go", "func x() {}", "```"},
			"heading(# Title) / text(Some)em(*em*)text(and)strong(**strong**)code(`code`)link([link])str((url)) / punct(```)type(go) / kw(func)fn(x)punct(())punct({}) / punct(```)"},
		{"diff", []string{"@@ -1,2 +1,2 @@ func x", "-a", "+b", " c"},
			"kw(@@ -1,2 +1,2 @@)fn(func x) / del(-a) / ins(+b) / text(c)"},
		{"yaml", []string{"key: 'v' # c", "- true"},
			"prop(key)op(:)str('v')comment(# c) / op(-)const(true)"},
		{"toml", []string{"[server]", "port = 8080"},
			"type([server]) / prop(port)op(=)num(8080)"},
		{"sh", []string{`echo "$HOME/x" $1 # c`},
			`fn(echo)str(")var($HOME)str(/x")var($1)comment(# c)`},
		{"sql", []string{"SELECT id FROM users WHERE x = 'a'"},
			"kw(SELECT)var(id)kw(FROM)var(users)kw(WHERE)var(x)op(=)str('a')"},
		{"dart", []string{`final s = r'\d' + "a${b}c$d"; // c`, "/* a /* b */ c */ var t = '''x", "y''';"},
			`kw(final)var(s)op(=)str(r'\d')op(+)str("a)esc(${)var(b)esc(})str(c)var($d)str(")punct(;)comment(// c) / ` +
				"comment(/* a /* b */ c */)kw(var)var(t)op(=)str('''x) / str(y''')punct(;)"},
		{"scala", []string{`val s = s"Hi $name ${a + 1}" + raw"\d" // c`, `/* a /* b */ c */ def f[T](x: T): Unit = """multi`, `line"""`},
			`kw(val)var(s)op(=)str(s"Hi)var($name)esc(${)var(a)op(+)num(1)esc(})str(")op(+)str(raw"\d")comment(// c) / ` +
				`comment(/* a /* b */ c */)kw(def)var(f)punct([)type(T)punct(]()var(x)op(:)type(T)punct())op(:)type(Unit)op(=)str("""multi) / str(line""")`},
		{"r", []string{`x <- c(1L, NA) %in% r"(a\b)" # c`, "if (TRUE) df$col.name |> is.na()"},
			`var(x)op(<-)fn(c)punct(()num(1L)punct(,)const(NA)punct())op(%in%)str(r"(a\b)")comment(# c) / ` +
				"kw(if)punct(()const(TRUE)punct())var(df)op($)var(col.name)op(|>)fn(is.na)punct(())"},
		{"graphql", []string{`query Get($id: ID!) @cached { user(id: $id) { ...F ... on User { name } } } # c`, `"""Doc`, `"""`},
			`kw(query)type(Get)punct(()var($id)op(:)type(ID)op(!)punct())meta(@cached)punct({)var(user)punct(()prop(id)op(:)var($id)punct())punct({)` +
				`punct(...)type(F)punct(...)kw(on)type(User)punct({)var(name)punct(})punct(})punct(})comment(# c) / str("""Doc) / str(""")`},
	} {
		l := LanguageNamed(c.lang)
		if l == nil {
			t.Fatalf("no language %q", c.lang)
		}
		if got := lexLines(l.Tokenizer, c.lines...); got != c.want {
			t.Errorf("%s %q:\n got %s\nwant %s", c.lang, c.lines, got, c.want)
		}
	}
}

// A minified JSON file's one line, of many keys, lexes in time linear in
// its length.
func TestJSONLongLine(t *testing.T) {
	line := "{" + strings.Repeat(`"key":[1.5,"v"],`, 100_000) + `"end":0}`
	start := time.Now()
	toks, _ := LanguageNamed("json").Tokenizer.Tokenize(line, 0, nil)
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("a line of %d keys took %v to lex", 100_000, d)
	}
	if n := len(toks); n < 700_000 {
		t.Errorf("%d tokens, want each key's, number's and string's", n)
	}
}

// A string with interpolations is one the lexer's stack can hold: its
// index in five bits.
func TestInterpolatedStringsFit(t *testing.T) {
	for _, l := range Languages() {
		lx, ok := l.Tokenizer.(*lexer)
		if !ok {
			continue
		}
		for k, s := range lx.strings {
			if s.interpOpen != "" && k+1 >= 1<<5 {
				t.Errorf("%s's string %s%s… is number %d, past the stack's 31", l.Name, s.prefix, s.open, k)
			}
		}
	}
}

func TestLanguageFor(t *testing.T) {
	for name, want := range map[string]string{"main.go": "Go", "a/b/Makefile": "Makefile", "x.TSX": "TypeScript", "Dockerfile": "Dockerfile", "x.unknown": "",
		"main.dart": "Dart", "schema.graphql": "GraphQL", "q.gql": "GraphQL", "plot.r": "R", "plot.R": "R", "Main.scala": "Scala", "build.sc": "Scala"} {
		l := LanguageFor(name)
		got := ""
		if l != nil {
			got = l.Name
		}
		if got != want {
			t.Errorf("LanguageFor(%q) = %q, want %q", name, got, want)
		}
	}
}

// LanguageNamed finds a language by its name, an extension or an alias,
// ignoring case, and each has a tokenizer.
func TestLanguageNamed(t *testing.T) {
	for want, names := range map[string][]string{
		"Go":      {"go", "Go", "golang"},
		"Dart":    {"dart", "Dart"},
		"GraphQL": {"graphql", "GraphQL", "gql"},
		"R":       {"r", "R", " r "},
		"Scala":   {"scala", "Scala", "sc"},
	} {
		for _, name := range names {
			l := LanguageNamed(name)
			if l == nil || l.Name != want {
				t.Errorf("LanguageNamed(%q) = %v, want %s", name, l, want)
				continue
			}
			if l.Tokenizer == nil {
				t.Errorf("%s has no tokenizer", l.Name)
			}
		}
	}
	if l := LanguageNamed("nope"); l != nil {
		t.Errorf("LanguageNamed(\"nope\") = %s", l.Name)
	}
}

// Each language's comments and brackets.
func TestLanguageCommentsAndBrackets(t *testing.T) {
	for name, want := range map[string][3]string{
		"dart": {"//", "/*", "*/"}, "scala": {"//", "/*", "*/"}, "graphql": {"#", "", ""}, "r": {"#", "", ""},
	} {
		l := LanguageNamed(name)
		if got := [3]string{l.LineComment, l.BlockComment[0], l.BlockComment[1]}; got != want {
			t.Errorf("%s's comments are %q, want %q", l.Name, got, want)
		}
		var pairs []string
		for _, b := range l.Brackets {
			pairs = append(pairs, b.Open+b.Close)
		}
		if got := strings.Join(pairs, " "); got != "() [] {}" {
			t.Errorf("%s's brackets are %s", l.Name, got)
		}
	}
}

// After any edits, the states the syntax keeps are those of tokenizing
// the text from the top.
func TestSyntaxIncremental(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 8))
	pieces := []string{"x", "\n", "/*", "*/", "`", "\"", " ", "func", "{", "}", "// c\n", "${", "a\nb"}
	b := NewBuffer(strings.Repeat("func f() {\n\tx := `a\nb` /* c */\n}\n", 20))
	b.SetLanguage(LanguageNamed("go"))
	for round := range 300 {
		if round%3 == 0 {
			b.SetLanguage(LanguageNamed([]string{"go", "javascript", "markdown"}[round/3%3]))
		}
		text, _ := b.snapshot()
		for range r.IntN(3) + 1 {
			at := r.IntN(text.Len() + 1)
			end := min(text.Len(), at+r.IntN(4))
			b.Edit(Edit{Start: at, End: end, Text: pieces[r.IntN(len(pieces))]})
			text, _ = b.snapshot()
		}
		s := b.highlighter().(*tokenHighlighter)
		// Sometimes only part of the text.
		upTo := text.LineCount() - 1
		if r.IntN(2) == 0 {
			upTo = r.IntN(text.LineCount())
		}
		if !s.Highlight(upTo, time.Hour) {
			t.Fatal("advance stopped")
		}
		var state uint64
		tok := b.Language().Tokenizer
		for i := 0; i <= upTo; i++ {
			got, ok := s.stateOf(i)
			if !ok || got != state {
				t.Fatalf("round %d: line %d starts in %x (%v), want %x", round, i, got, ok, state)
			}
			_, state = tok.Tokenize(text.Line(i), state, nil)
		}
	}
}

func BenchmarkTokenizeGo(b *testing.B) {
	src := strings.Repeat("func (s *server) handle(w http.ResponseWriter, r *http.Request) error {\n\tif err := s.check(r.Context(), \"x\\n\", 0x1f); err != nil { // why\n\t\treturn fmt.Errorf(`bad: %w`, err)\n\t}\n\treturn nil\n}\n", 200)
	lines := strings.Split(src, "\n")
	tok := LanguageNamed("go").Tokenizer
	var toks []Token
	b.SetBytes(int64(len(src)))
	for range b.N {
		var state uint64
		for _, l := range lines {
			toks, state = tok.Tokenize(l, state, toks[:0])
		}
	}
}
