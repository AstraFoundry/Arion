package mygo

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"io"
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"
)

// printedIdeographs is what RadicalIdeographs answers on macOS 26 for the
// fonts of testdata/radicals.pdf, which WebKit printed there from
//
//	<p style="font-family: 'PingFang SC'">人工智能让写作更容易。用户点击按钮后，页面会显示一个对话框。门口停着一辆车。</p>
//	<p style="font-family: 'Songti SC'">(the same)</p>
//
// Songti SC draws ⼜, the radical of 又, with the glyph of 口.
var printedIdeographs = map[string]map[rune]rune{
	"PingFangSC-Regular":  {'⼀': '一', '⼈': '人', '⼝': '口', '⼯': '工', '⽤': '用', '⻋': '车', '⻔': '门', '⾯': '面', '⻚': '页'},
	"STSongti-SC-Regular": {'⼀': '一', '⼈': '人', '⼜': '口', '⼯': '工', '⽤': '用', '⽰': '示', '⾯': '面'},
}

// ideographsFrom returns a lookup that answers from fonts and records what
// it was asked.
func ideographsFrom(fonts map[string]map[rune]rune, asked map[string]string) func(string, []rune) map[rune]rune {
	return func(font string, radicals []rune) map[rune]rune {
		asked[font] += string(radicals)
		return fonts[font]
	}
}

func TestFixPDFTextPrinted(t *testing.T) {
	original, err := os.ReadFile("testdata/radicals.pdf")
	if err != nil {
		t.Fatal(err)
	}
	const page = "人工智能让写作更容易。用户点击按钮后，页面会显示一个对话框。门口停着一辆车。"
	before := testTextMaps(t, original)
	if !strings.ContainsRune(before[13].text(), '⻔') || !strings.ContainsRune(before[18].text(), '⼜') {
		t.Fatalf("the fixture's maps no longer name ⻔ for 门 in PingFang SC and ⼜ for 口 in Songti SC: %q, %q", before[13].text(), before[18].text())
	}
	asked := map[string]string{}
	fixed := fixPDFText(original, ideographsFrom(printedIdeographs, asked))
	if !bytes.HasPrefix(fixed, original) || len(fixed) == len(original) {
		t.Fatal("the fix is not an update appended to the PDF")
	}
	// Once per font, with the radicals of its maps in order.
	want := map[string]string{"PingFangSC-Regular": "⻋⻔⻚⼀⼈⼝⼯⽤⾯", "STSongti-SC-Regular": "⼀⼈⼜⼯⽤⽰⾯"}
	if fmt.Sprint(asked) != fmt.Sprint(want) {
		t.Errorf("asked %q, want %q", asked, want)
	}
	if objects := testObjects(fixed[len(original):]); len(objects) != 2 || objects[13] == nil || objects[18] == nil {
		t.Errorf("the update has objects %v, want the two maps", slices.Sorted(maps.Keys(objects)))
	}
	trailer := testCheckXref(t, fixed)
	for _, key := range []string{"Root", "Info", "ID"} {
		entry := regexp.MustCompile(`/` + key + `\s*(\[[^\]]*\]|\d+\s+\d+\s+R)`).Find(original)
		if entry == nil || !strings.Contains(trailer, string(entry)) {
			t.Errorf("the trailer lost /%s: %s", key, trailer)
		}
	}
	if !strings.Contains(trailer, fmt.Sprintf("/Prev %d", testStartXref(t, original))) {
		t.Errorf("the trailer does not point to the original's: %s", trailer)
	}
	after := testTextMaps(t, fixed)
	for n, m := range before {
		if len(after[n]) != len(m) {
			t.Errorf("map %d has %d mappings, had %d", n, len(after[n]), len(m))
		}
		for code, text := range after[n] {
			for _, r := range text {
				if !strings.ContainsRune(page, r) {
					t.Errorf("map %d maps <%s> to %q, which is not on the page (was %q)", n, code, text, m[code])
				}
			}
		}
	}
	if again := fixPDFText(fixed, ideographsFrom(printedIdeographs, map[string]string{})); !bytes.Equal(again, fixed) {
		t.Error("fixing the PDF again changed it")
	}
}

// testFont is the lookup of the synthetic PDFs, whose font is Test-Regular.
var testFont = map[string]map[rune]rune{"Test-Regular": {'⽂': '文', '⻓': '长', '⼀': '一', '⼆': '二', '⼗': '十'}}

func TestFixPDFTextFonts(t *testing.T) {
	const cmap = "1 beginbfchar\n<01> <2F42>\nendbfchar"
	for _, tc := range []struct {
		name, font string
		fonts      []string // other fonts with the same map
		asked      string
	}{
		{"subset", "/BaseFont /ABCDEF+Test-Regular", nil, "Test-Regular"},
		{"whole font", "/BaseFont /Test-Regular", nil, "Test-Regular"},
		{"escaped name", "/BaseFont /Test#2DRegular", nil, "Test-Regular"},
		{"another subset of the font", "/BaseFont /ABCDEF+Test-Regular", []string{"/BaseFont /GHIJKL+Test-Regular"}, "Test-Regular"},
		{"fonts of other names", "/BaseFont /ABCDEF+Test-Regular", []string{"/BaseFont /Test-Bold"}, ""},
		{"no tag", "/BaseFont /abcdef+Test-Regular", nil, "abcdef+Test-Regular"},
		{"no name", "", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testPDF(cmap, "", false, append([]string{tc.font}, tc.fonts...)...)
			asked := map[string]string{}
			fixed := fixPDFText(p, ideographsFrom(testFont, asked))
			if tc.asked == "" {
				if len(asked) != 0 || !bytes.Equal(fixed, p) {
					t.Errorf("asked %q for a map it cannot tell the font of", asked)
				}
				return
			}
			if len(asked) != 1 || asked[tc.asked] != "⽂" {
				t.Errorf("asked %q, want ⽂ of %s", asked, tc.asked)
			}
			want := "⽂"
			if testFont[tc.asked] != nil {
				want = "文"
			}
			if got := testTextMaps(t, fixed)[4]["01"]; got != want {
				t.Errorf("<01> maps to %q, want %q", got, want)
			}
		})
	}
}

func TestReadToUnicode(t *testing.T) {
	for _, tc := range []struct{ name, sections, want string }{
		{"bfchar", "2 beginbfchar\n<01><2f42>\n<02><00412f00>\nendbfchar", "2 beginbfchar\n<01> <6587>\n<02> <00414E00>\nendbfchar\n"},
		{"range of one", "1 beginbfrange\n<21><21><2ed3>\nendbfrange", "1 beginbfchar\n<21> <957F>\nendbfchar\n"},
		{"range", "1 beginbfrange\n<01><03><2f05>\nendbfrange", "3 beginbfchar\n<01> <2F05>\n<02> <4E8C>\n<03> <2F07>\nendbfchar\n"},
		{"range of texts", "1 beginbfrange\n<01><03>[<2f42><0041><2ed3>]\nendbfrange", "3 beginbfchar\n<01> <6587>\n<02> <0041>\n<03> <957F>\nendbfchar\n"},
		{"two-byte codes", "1 beginbfchar\n<00ab> <2F42>\nendbfchar", "1 beginbfchar\n<00AB> <6587>\nendbfchar\n"},
		{"surrogate pair", "2 beginbfchar\n<01><d840dc00>\n<02><2f42>\nendbfchar", "2 beginbfchar\n<01> <D840DC00>\n<02> <6587>\nendbfchar\n"},
		{"sections", "1 beginbfchar\n<01><2f42>\nendbfchar\n1 beginbfrange\n<02><02><2ed3>\nendbfrange", "2 beginbfchar\n<01> <6587>\n<02> <957F>\nendbfchar\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const head, tail = "% head\n1 begincodespacerange\n<00> <FF>\nendcodespacerange\n", "\nendcmap\n% tail\n"
			c, err := readToUnicode([]byte(head + tc.sections + tail))
			if err != nil {
				t.Fatal(err)
			}
			got, changed := c.rewrite(testFont["Test-Regular"])
			if !changed || string(got) != head+tc.want+tail {
				t.Errorf("rewrote to\n%s\nwant\n%s", got, head+tc.want+tail)
			}
		})
	}
}

func TestReadToUnicodeSections(t *testing.T) {
	// A section has at most 100 mappings.
	c, err := readToUnicode([]byte("1 beginbfrange\n<0000> <00C8> [" + strings.Repeat("<2F42> ", 201) + "]\nendbfrange"))
	if err != nil {
		t.Fatal(err)
	}
	got, _ := c.rewrite(testFont["Test-Regular"])
	if n := strings.Count(string(got), "beginbfchar"); n != 3 || !strings.HasPrefix(string(got), "100 beginbfchar") || !strings.Contains(string(got), "1 beginbfchar\n<00C8> <6587>") {
		t.Errorf("%d sections:\n%s", n, got)
	}
	if n := len(testMappings(t, got)); n != 201 {
		t.Errorf("%d mappings, want 201", n)
	}
}

func TestReadToUnicodeUnchanged(t *testing.T) {
	for _, s := range []string{"1 beginbfchar <01> <D840DC00> endbfchar", "1 beginbfchar <01> <6587> endbfchar", "% beginbfchar in a comment\n"} {
		c, err := readToUnicode([]byte(s))
		if err != nil {
			t.Fatalf("%q: %v", s, err)
		}
		if _, changed := c.rewrite(testFont["Test-Regular"]); changed {
			t.Errorf("%q changed", s)
		}
	}
	for _, s := range []string{
		"1 beginbfchar <01> <2F42>",
		"2 beginbfchar <01> <2F42> endbfchar",
		"1 beginbfrange <02> <01> <2F42> endbfrange",
		"1 beginbfrange <01> <02> [<2F42>] endbfrange",
		"1 beginbfrange <01> <02> <2FFF> endbfrange", // the last byte would pass 255
		"1 beginbfrange <0001> <01> <2F42> endbfrange",
		"1 beginbfchar <01> <D800> endbfchar",
		"1 beginbfchar <01> <2F4> endbfchar",
		"1 beginbfchar <01> <> endbfchar",
		"1 beginbfchar <01> <2F42> endbfchar def 1 beginbfchar <02> <2F42> endbfchar", // a rewrite would drop "def"
		"endbfchar",
	} {
		if _, err := readToUnicode([]byte(s)); err == nil {
			t.Errorf("read %q", s)
		}
	}
}

func TestFixPDFTextStreams(t *testing.T) {
	for _, filter := range []string{"", "/Filter /FlateDecode", "/Filter [/FlateDecode]"} {
		for _, indirect := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s indirect=%v", filter, indirect), func(t *testing.T) {
				p := testPDF("1 beginbfchar\n<01> <2F42>\nendbfchar", filter, indirect, "/BaseFont /Test-Regular")
				fixed := fixPDFText(p, ideographsFrom(testFont, map[string]string{}))
				if len(fixed) == len(p) {
					t.Fatal("no update")
				}
				if trailer := testCheckXref(t, fixed); !strings.Contains(trailer, "/Custom << /Text (escaped \\( value \\) /Prev) /Values [1 true null 2.5] >>") {
					t.Errorf("the trailer lost an entry: %s", trailer)
				}
				obj := testObjects(fixed)[4]
				if !bytes.Contains(obj, []byte("/Filter /FlateDecode")) || bytes.Contains(obj, []byte("/Length 5 0 R")) {
					t.Errorf("the new map has %s", obj[:bytes.Index(obj, []byte("stream"))])
				}
				if got := testTextMaps(t, fixed)[4]["01"]; got != "文" {
					t.Errorf("<01> maps to %q", got)
				}
			})
		}
	}
}

func TestFixPDFTextNewestXref(t *testing.T) {
	p := testPDF("1 beginbfchar\n<01> <2ED3>\nendbfchar", "", false, "/BaseFont /Test-Regular")
	prev := testStartXref(t, p)
	const cmap = "1 beginbfchar\n<01> <2F42>\nendbfchar"
	var update bytes.Buffer
	fmt.Fprintf(&update, "4 3 obj\n<< /Length %d >>\nstream\n%s\nendstream\nendobj\n", len(cmap), cmap)
	xref := len(p) + update.Len()
	fmt.Fprintf(&update, "xref\n4 1\n%010d 00003 n \ntrailer\n<< /Size 6 /Root 1 0 R /Info 5 0 R /Prev %d /XRefStm 0 >>\nstartxref\n%d\n%%%%EOF\n", len(p), prev, xref)
	p = append(p, update.Bytes()...)
	fixed := fixPDFText(p, ideographsFrom(testFont, map[string]string{}))
	if !bytes.HasPrefix(fixed, p) || len(fixed) == len(p) {
		t.Fatal("no update")
	}
	if got := testTextMaps(t, fixed)[4]["01"]; got != "文" {
		t.Errorf("<01> maps to %q: the fix read the older map", got)
	}
	if trailer := testCheckXref(t, fixed); !strings.Contains(trailer, fmt.Sprintf("/Prev %d", xref)) || strings.Contains(trailer, "/XRefStm") || strings.Count(trailer, "/Prev") != 1 {
		t.Errorf("trailer %s", trailer)
	}
}

func TestFixPDFTextUnchanged(t *testing.T) {
	const radical = "1 beginbfchar\n<01> <2F42>\nendbfchar"
	p := testPDF(radical, "", false, "/BaseFont /Test-Regular")
	compressed := testPDF(radical, "/Filter /FlateDecode", false, "/BaseFont /Test-Regular")
	compressed[bytes.Index(compressed, []byte("stream\n"))+len("stream\n")] ^= 0xFF
	for name, p := range map[string][]byte{
		"not a PDF":          []byte("hello"),
		"empty":              nil,
		"no radical":         testPDF("1 beginbfchar\n<01> <6587>\nendbfchar", "", false, "/BaseFont /Test-Regular"),
		"font not installed": testPDF(radical, "", false, "/BaseFont /Other-Regular"),
		"xref stream":        []byte("%PDF-1.5\n1 0 obj\n<< /Type /XRef >>\nendobj\nstartxref\n9\n%%EOF\n"),
		"no startxref":       []byte("%PDF-1.3\n%%EOF"),
		"other filter":       testPDF(radical, "/Filter /ASCIIHexDecode", false, "/BaseFont /Test-Regular"),
		"damaged map":        testPDF("1 beginbfchar <01> <2F42>", "", false, "/BaseFont /Test-Regular"),
		"damaged stream":     compressed,
		"other generation":   bytes.Replace(p, []byte("/ToUnicode 4 3 R"), []byte("/ToUnicode 4 2 R"), 1),
		"wrong offset":       bytes.Replace(p, []byte("1 0 obj"), []byte("9 0 obj"), 1),
		"wrong length":       bytes.Replace(p, fmt.Appendf(nil, "/Length %d", len(radical)), []byte("/Length 99"), 1),
		"encrypted":          bytes.Replace(p, []byte("/Size 6"), []byte("/Size 6 /Encrypt 5 0 R"), 1),
		"object stream":      bytes.Replace(p, []byte("/Type /Pages"), []byte("/Type /ObjStm"), 1),
		"Prev loop":          bytes.Replace(p, []byte("/Size 6"), []byte(fmt.Sprintf("/Prev %d /Size 6", testStartXref(t, p))), 1),
	} {
		if got := fixPDFText(p, ideographsFrom(testFont, map[string]string{})); !bytes.Equal(got, p) {
			t.Errorf("%s: changed", name)
		}
	}
	for i := range len(bytes.TrimSpace(p)) {
		if got := fixPDFText(p[:i], ideographsFrom(testFont, map[string]string{})); !bytes.Equal(got, p[:i]) {
			t.Fatalf("changed the first %d bytes", i)
		}
	}
}

// FuzzFixPDFText: whatever the input, the output begins with it.
func FuzzFixPDFText(f *testing.F) {
	printed, err := os.ReadFile("testdata/radicals.pdf")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(printed)
	f.Add(printed[:len(printed)/2])
	f.Add(testPDF("1 beginbfrange\n<01> <02> [<2F42> <2ED3>]\nendbfrange", "/Filter /FlateDecode", true, "/BaseFont /Test-Regular"))
	fonts := map[string]map[rune]rune{}
	for name, m := range printedIdeographs {
		fonts[name] = m
	}
	fonts["Test-Regular"] = testFont["Test-Regular"]
	f.Fuzz(func(t *testing.T, p []byte) {
		if got := fixPDFText(p, ideographsFrom(fonts, map[string]string{})); !bytes.HasPrefix(got, p) {
			t.Fatal("the output does not begin with the input")
		}
	})
}

// testPDF returns a PDF whose fonts, objects 3 and from 6 on with the
// entries of fonts, share the ToUnicode map cmap, object 4 of generation 3.
// Object 5 is the map's length.
func testPDF(cmap, filter string, indirect bool, fonts ...string) []byte {
	var stream bytes.Buffer
	if filter == "" {
		stream.WriteString(cmap)
	} else {
		w := zlib.NewWriter(&stream)
		w.Write([]byte(cmap))
		w.Close()
	}
	length := strconv.Itoa(stream.Len())
	if indirect {
		length = "5 0 R"
	}
	font := func(entries string) string {
		return fmt.Sprintf("<< /Type /Font /Subtype /TrueType %s /ToUnicode 4 3 R >>", entries)
	}
	objects := []string{
		1: "<< /Type /Catalog /Pages 2 0 R >>",
		2: "<< /Type /Pages /Count 0 /Kids [] >>",
		3: font(fonts[0]),
		4: fmt.Sprintf("<< /Length %s %s >>\nstream\n%s\nendstream", length, filter, stream.String()),
		5: strconv.Itoa(stream.Len()),
	}
	for _, entries := range fonts[1:] {
		objects = append(objects, font(entries))
	}
	gen := func(n int) int {
		if n == 4 {
			return 3
		}
		return 0
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.3\n")
	offsets := make([]int, len(objects))
	for n := 1; n < len(objects); n++ {
		offsets[n] = b.Len()
		fmt.Fprintf(&b, "%d %d obj\n%s\nendobj\n", n, gen(n), objects[n])
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n0000000000 65535 f \n", len(objects))
	for n := 1; n < len(objects); n++ {
		fmt.Fprintf(&b, "%010d %05d n \n", offsets[n], gen(n))
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R /Info 5 0 R /ID [<ABCD><1234>] /Custom << /Text (escaped \\( value \\) /Prev) /Values [1 true null 2.5] >> >>\nstartxref\n%d\n%%%%EOF\n", len(objects), xref)
	return b.Bytes()
}

// The readers below read what the tests need with regular expressions,
// not with the code they test.

var (
	testObjectRE  = regexp.MustCompile(`(?m)^(\d+) (\d+) obj\s*`)
	testSectionRE = regexp.MustCompile(`(?s)\d+\s+begin(bfchar|bfrange)\s*(.*?)\s*end(?:bfchar|bfrange)`)
	testHexRE     = regexp.MustCompile(`<([0-9A-Fa-f\s]+)>|\[|\]`)
)

func testStartXref(t *testing.T, p []byte) int {
	t.Helper()
	m := regexp.MustCompile(`startxref\s+(\d+)\s+%%EOF\s*$`).FindSubmatch(p)
	if m == nil {
		t.Fatal("no startxref at the end")
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

// testObjects returns the objects of p by number, the newest of each.
func testObjects(p []byte) map[int][]byte {
	objects := map[int][]byte{}
	for _, at := range testObjectRE.FindAllSubmatchIndex(p, -1) {
		if end := bytes.Index(p[at[1]:], []byte("endobj")); end >= 0 {
			n, _ := strconv.Atoi(string(p[at[2]:at[3]]))
			objects[n] = p[at[1] : at[1]+end]
		}
	}
	return objects
}

// testTextMap maps the codes of a ToUnicode map to their texts.
type testTextMap map[string]string

func (m testTextMap) text() string {
	var b strings.Builder
	for _, code := range slices.Sorted(maps.Keys(m)) {
		b.WriteString(m[code])
	}
	return b.String()
}

// testTextMaps returns the ToUnicode maps of p by object number.
func testTextMaps(t *testing.T, p []byte) map[int]testTextMap {
	t.Helper()
	objects := testObjects(p)
	maps := map[int]testTextMap{}
	for _, obj := range objects {
		if m := regexp.MustCompile(`/ToUnicode\s+(\d+)\s+\d+\s+R`).FindSubmatch(obj); m != nil {
			n, _ := strconv.Atoi(string(m[1]))
			maps[n] = testMappings(t, testStream(t, objects[n]))
		}
	}
	return maps
}

func testStream(t *testing.T, obj []byte) []byte {
	t.Helper()
	start := bytes.Index(obj, []byte("stream\n"))
	if start < 0 {
		t.Fatal("no stream")
	}
	data := obj[start+len("stream\n") : bytes.LastIndex(obj, []byte("endstream"))]
	if !bytes.Contains(obj[:start], []byte("/FlateDecode")) {
		return bytes.TrimSuffix(data, []byte("\n"))
	}
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if data, err = io.ReadAll(r); err != nil {
		t.Fatal(err)
	}
	return data
}

func testMappings(t *testing.T, cmap []byte) testTextMap {
	t.Helper()
	text := func(h string) string {
		b, err := hex.DecodeString(strings.Join(strings.Fields(h), ""))
		if err != nil || len(b)%2 != 0 {
			t.Fatalf("not UTF-16: %q", h)
		}
		units := make([]uint16, len(b)/2)
		for i := range units {
			units[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
		}
		return string(utf16.Decode(units))
	}
	m := testTextMap{}
	for _, section := range testSectionRE.FindAllSubmatch(cmap, -1) {
		tokens := testHexRE.FindAllStringSubmatch(string(section[2]), -1)
		for i := 0; i < len(tokens); {
			code := strings.ToUpper(strings.Join(strings.Fields(tokens[i][1]), ""))
			if string(section[1]) == "bfchar" {
				m[code] = text(tokens[i+1][1])
				i += 2
				continue
			}
			lo, _ := strconv.ParseUint(code, 16, 64)
			hi, _ := strconv.ParseUint(tokens[i+1][1], 16, 64)
			i += 2
			if tokens[i][0] == "[" {
				for c := lo; c <= hi; c++ {
					i++
					m[fmt.Sprintf("%0*X", len(code), c)] = text(tokens[i][1])
				}
				i += 2
				continue
			}
			first, _ := strconv.ParseUint(tokens[i][1], 16, 64)
			for c := lo; c <= hi; c++ {
				m[fmt.Sprintf("%0*X", len(code), c)] = text(fmt.Sprintf("%0*X", len(tokens[i][1]), first+c-lo))
			}
			i++
		}
	}
	return m
}

// testCheckXref checks the newest cross-reference section of p and returns
// its trailer.
func testCheckXref(t *testing.T, p []byte) string {
	t.Helper()
	section := string(p[testStartXref(t, p):])
	if !strings.HasPrefix(section, "xref\n") {
		t.Fatal("no cross-reference table")
	}
	lines := strings.Split(section, "\n")
	for i := 1; lines[i] != "trailer"; {
		var first, count int
		if _, err := fmt.Sscanf(lines[i], "%d %d", &first, &count); err != nil {
			t.Fatal(err)
		}
		i++
		for n := first; n < first+count; n++ {
			line := lines[i] + "\n"
			i++
			var at, gen int
			var kind string
			fmt.Sscanf(line, "%d %d %s", &at, &gen, &kind)
			if len(line) != 20 || kind != "n" {
				t.Fatalf("entry %q", line)
			}
			if at >= len(p) || !bytes.HasPrefix(p[at:], fmt.Appendf(nil, "%d %d obj", n, gen)) {
				t.Fatalf("object %d %d is not at %d", n, gen, at)
			}
		}
	}
	return section[strings.Index(section, "trailer")+len("trailer") : strings.Index(section, "startxref")]
}
