//go:build !js

package e2e

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"io"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/egoist/mygo"
)

// The PDFs that WebKit prints on macOS name radicals for many Chinese
// characters in their ToUnicode maps, ⼈ (U+2F08) for 人 in PingFang SC and
// ⼜ (U+2F1C) for 口 in Songti SC; PrintToPDF gives the characters back.
func TestPrintToPDFText(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the radicals in the text of printed PDFs are a quirk of CoreGraphics")
	}
	const text = "人工智能让写作更容易。用户点击按钮后，页面会显示一个对话框。门口停着一辆车。"
	w := newWindow(t, mygo.WindowOptions{Title: "PDF text", Width: 400, Height: 300})
	w.Page().LoadHTML(`<html lang="zh-CN"><p style="font-family: 'PingFang SC'">`+text+`</p><p style="font-family: 'Songti SC'">`+text+`</p></html>`, "")
	waitFor(t, w, `document.readyState === "complete" && document.querySelectorAll("p").length === 2`)
	pdf, err := w.Page().PrintToPDF(mygo.PDFOptions{})
	if err != nil {
		t.Fatal(err)
	}
	texts := printedTexts(t, pdf)
	if len(texts) != 2 {
		t.Fatalf("%d ToUnicode maps, want one for each font", len(texts))
	}
	for _, s := range texts {
		for _, r := range s {
			if !strings.ContainsRune(text, r) {
				t.Errorf("the PDF's text has %q (%U), which the page does not", r, r)
			}
		}
	}
}

// printedTexts returns the texts of the ToUnicode maps of pdf, the newest
// object of each number, as an update to the PDF appends them.
func printedTexts(t *testing.T, pdf []byte) []string {
	t.Helper()
	objects := map[string][]byte{}
	header := regexp.MustCompile(`(?m)^(\d+) \d+ obj\s*`)
	for _, at := range header.FindAllSubmatchIndex(pdf, -1) {
		if end := bytes.Index(pdf[at[1]:], []byte("endobj")); end >= 0 {
			objects[string(pdf[at[2]:at[3]])] = pdf[at[1] : at[1]+end]
		}
	}
	refs := map[string]bool{}
	for _, obj := range objects {
		if m := regexp.MustCompile(`/ToUnicode\s+(\d+)\s+\d+\s+R`).FindSubmatch(obj); m != nil {
			refs[string(m[1])] = true
		}
	}
	var texts []string
	for n := range refs {
		obj := objects[n]
		start := bytes.Index(obj, []byte("stream"))
		end := bytes.LastIndex(obj, []byte("endstream"))
		if start < 0 || end < start {
			t.Fatalf("ToUnicode map %s has no stream", n)
		}
		r, err := zlib.NewReader(bytes.NewReader(bytes.TrimLeft(obj[start+len("stream"):end], "\r\n")))
		if err != nil {
			t.Fatalf("ToUnicode map %s: %v", n, err)
		}
		cmap, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ToUnicode map %s: %v", n, err)
		}
		texts = append(texts, cmapText(t, cmap))
	}
	return texts
}

// cmapText returns the texts that a CMap maps codes to, run together.
func cmapText(t *testing.T, cmap []byte) string {
	t.Helper()
	var b strings.Builder
	sections := regexp.MustCompile(`(?s)begin(bfchar|bfrange)(.*?)end(?:bfchar|bfrange)`)
	strs := regexp.MustCompile(`<([0-9A-Fa-f]*)>|\[|\]`)
	for _, section := range sections.FindAllSubmatch(cmap, -1) {
		tokens := strs.FindAllSubmatch(section[2], -1)
		for i := 0; i < len(tokens); {
			if string(section[1]) == "bfchar" {
				b.WriteString(utf16Hex(t, string(tokens[i+1][1])))
				i += 2
				continue
			}
			lo, _ := strconv.ParseUint(string(tokens[i][1]), 16, 32)
			hi, _ := strconv.ParseUint(string(tokens[i+1][1]), 16, 32)
			i += 2
			if string(tokens[i][0]) == "[" {
				for i++; string(tokens[i][0]) != "]"; i++ {
					b.WriteString(utf16Hex(t, string(tokens[i][1])))
				}
				i++
				continue
			}
			first := string(tokens[i][1])
			last, _ := strconv.ParseUint(first[len(first)-2:], 16, 8)
			for c := lo; c <= hi; c++ {
				b.WriteString(utf16Hex(t, first[:len(first)-2]+strconv.FormatUint(0x100+last+c-lo, 16)[1:]))
			}
			i++
		}
	}
	return b.String()
}

func utf16Hex(t *testing.T, h string) string {
	t.Helper()
	data, err := hex.DecodeString(h)
	if err != nil || len(data)%2 != 0 {
		t.Fatalf("not UTF-16: <%s>", h)
	}
	units := make([]uint16, len(data)/2)
	for i := range units {
		units[i] = uint16(data[2*i])<<8 | uint16(data[2*i+1])
	}
	return string(utf16.Decode(units))
}
