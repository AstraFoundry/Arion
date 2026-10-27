package typewriter

import (
	"sort"
	"strings"
	"time"
	"unicode"

	"arion.local/desktop/ui/shared/assets"
	"golang.org/x/image/font"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/text/unicode/norm"
)

// glyph is one character's outline in ems, y up from the baseline.
type glyph struct {
	advance  float32
	contours [][][2]float32
}

// face draws the serif's characters, each loaded once.
type face struct {
	font   *sfnt.Font
	buf    sfnt.Buffer
	glyphs map[rune]glyph
}

var serif = func() *face {
	f, err := sfnt.Parse(assets.Serif)
	if err != nil {
		// The typeface is embedded: one that does not parse is a build defect.
		panic("typewriter: " + err.Error())
	}
	return &face{font: f, glyphs: map[rune]glyph{}}
}()

// ppem loads outlines at 64 pixels to the em, so that 26.6 coordinates
// divided by 64 × 64 are ems.
const ppem = fixed.Int26_6(64 << 6)

// has reports whether the face has a glyph for r.
func (f *face) has(r rune) bool {
	i, err := f.font.GlyphIndex(&f.buf, r)
	return err == nil && i != 0
}

func (f *face) glyph(r rune) glyph {
	if g, ok := f.glyphs[r]; ok {
		return g
	}
	var g glyph
	i, err := f.font.GlyphIndex(&f.buf, r)
	if err == nil {
		advance, _ := f.font.GlyphAdvance(&f.buf, i, ppem, font.HintingNone)
		g.advance = float32(advance) / (64 * 64)
		segments, _ := f.font.LoadGlyph(&f.buf, i, ppem, nil)
		g.contours = flatten(segments)
	}
	f.glyphs[r] = g
	return g
}

// flatten turns a glyph's segments into polygons, curves in short chords.
func flatten(segments sfnt.Segments) [][][2]float32 {
	em := func(p fixed.Point26_6) [2]float32 {
		return [2]float32{float32(p.X) / (64 * 64), -float32(p.Y) / (64 * 64)}
	}
	var out [][][2]float32
	var contour [][2]float32
	pen := [2]float32{}
	for _, s := range segments {
		switch s.Op {
		case sfnt.SegmentOpMoveTo:
			if len(contour) > 2 {
				out = append(out, contour)
			}
			pen = em(s.Args[0])
			contour = [][2]float32{pen}
		case sfnt.SegmentOpLineTo:
			pen = em(s.Args[0])
			contour = append(contour, pen)
		case sfnt.SegmentOpQuadTo:
			c, end := em(s.Args[0]), em(s.Args[1])
			for i := 1; i <= 4; i++ {
				t := float32(i) / 4
				a, b, d := (1-t)*(1-t), 2*(1-t)*t, t*t
				contour = append(contour, [2]float32{a*pen[0] + b*c[0] + d*end[0], a*pen[1] + b*c[1] + d*end[1]})
			}
			pen = end
		case sfnt.SegmentOpCubeTo:
			c1, c2, end := em(s.Args[0]), em(s.Args[1]), em(s.Args[2])
			for i := 1; i <= 6; i++ {
				t := float32(i) / 6
				a, b, d, e := (1-t)*(1-t)*(1-t), 3*(1-t)*(1-t)*t, 3*(1-t)*t*t, t*t*t
				contour = append(contour, [2]float32{a*pen[0] + b*c1[0] + d*c2[0] + e*end[0], a*pen[1] + b*c1[1] + d*c2[1] + e*end[1]})
			}
			pen = end
		}
	}
	if len(contour) > 2 {
		out = append(out, contour)
	}
	return out
}

// stroke is one character struck: where on its line it lands, in ems from
// the margin, and when, from the start of a pass through the text.
type stroke struct {
	r  rune
	x  float32
	at time.Duration
}

// line is one typed line: its strokes, when it starts, and when the
// carriage returns after it.
type line struct {
	strokes     []stroke
	start, back time.Duration
	end         float32 // the pen after the last stroke, in ems
}

// The typist's rhythm.
const (
	beat      = 70 * time.Millisecond // between two keys, at least
	swing     = 55 * time.Millisecond // more, at most, by the key
	pause     = 60 * time.Millisecond // more after a space
	lineEnd   = 260 * time.Millisecond
	returning = 520 * time.Millisecond // the carriage's return and line feed
	indent    = 2                      // ems a paragraph starts in
	blank     = 2                      // lines left empty after the text
)

// hash spreads n over 32 bits, for the rhythm and the ink.
func hash(n uint32) uint32 {
	n ^= n >> 16
	n *= 0x7feb352d
	n ^= n >> 15
	n *= 0x846ca68b
	n ^= n >> 16
	return n
}

// script is the text set in lines of width ems, with the time each stroke
// lands; it types again from the top after blank empty lines.
type script struct {
	lines []line
	pass  time.Duration
}

func typeset(text string, width float32) script {
	var s script
	var at time.Duration
	n := uint32(0)
	add := func(words []string, first bool) {
		l := line{start: at}
		pen := float32(0)
		if first {
			pen = indent
		}
		for i, word := range words {
			if i > 0 {
				pen += serif.glyph(' ').advance
				at += pause
			}
			for _, r := range word {
				n++
				at += beat + time.Duration(hash(n)%uint32(swing/time.Millisecond))*time.Millisecond
				l.strokes = append(l.strokes, stroke{r: r, x: pen, at: at})
				pen += serif.glyph(r).advance
			}
		}
		l.end = pen
		l.back = at + lineEnd
		at = l.back + returning
		s.lines = append(s.lines, l)
	}
	for _, paragraph := range strings.Split(strings.TrimSpace(text), "\n") {
		words := strings.Fields(paragraph)
		first := true
		for len(words) > 0 {
			pen := float32(0)
			if first {
				pen = indent
			}
			count := 0
			for count < len(words) {
				w := advance(words[count])
				if count > 0 {
					w += serif.glyph(' ').advance
				}
				if count > 0 && pen+w > width {
					break
				}
				pen += w
				count++
			}
			add(words[:count], first)
			words, first = words[count:], false
		}
	}
	for range blank {
		add(nil, false)
	}
	s.pass = at
	return s
}

func advance(word string) float32 {
	var w float32
	for _, r := range word {
		w += serif.glyph(r).advance
	}
	return w
}

// state is where the typing has got to: whole lines are typed before line,
// count strokes of it are, the last struck since ago, and the carriage
// stands with the pen at pen ems and the paper fed by feed lines. Glide is
// the pen moving evenly along the line instead of stroke by stroke.
type state struct {
	line, count      int
	since            time.Duration
	pen, feed, glide float32
}

// The strike's moments: the type bar flies up and back, the carriage
// steps on once it has.
const (
	strike = 110 * time.Millisecond
	step   = 45 * time.Millisecond
)

func ease(t float32) float32 {
	t = min(1, max(0, t))
	return t * t * (3 - 2*t)
}

// at is the typing t after it starts. Lines count on across passes.
func (s script) at(t time.Duration) state {
	pass := int(t / s.pass)
	local := t % s.pass
	i := max(0, sort.Search(len(s.lines), func(i int) bool { return s.lines[i].start > local })-1)
	l := s.lines[i]
	out := state{line: pass*len(s.lines) + i, since: time.Hour}
	out.count = sort.Search(len(l.strokes), func(j int) bool { return l.strokes[j].at > local })
	switch {
	case local >= l.back:
		// Back to where the next line starts.
		u := ease(float32(local-l.back) / float32(returning))
		var to float32
		if next := s.lines[(i+1)%len(s.lines)]; len(next.strokes) > 0 {
			to = next.strokes[0].x
		}
		out.pen, out.feed = lerp(l.end, to, u), float32(out.line)+u
		out.glide = out.pen
		out.count = len(l.strokes)
	case out.count > 0:
		first := l.strokes[0]
		out.glide = lerp(first.x, l.end, float32(local-first.at)/float32(l.back-first.at))
		k := l.strokes[out.count-1]
		out.since = local - k.at
		from := k.x
		out.pen = from + (k.x+serif.glyph(k.r).advance-from)*ease(float32(out.since)/float32(step))
		out.feed = float32(out.line)
	default:
		out.feed = float32(out.line)
		if len(l.strokes) > 0 {
			out.pen = l.strokes[0].x
		}
		out.glide = out.pen
	}
	return out
}

// next is how long after t the picture next changes, when it stands still
// until then: between strokes, and after a strike has settled.
func (s script) next(t time.Duration) time.Duration {
	local := t % s.pass
	i := max(0, sort.Search(len(s.lines), func(i int) bool { return s.lines[i].start > local })-1)
	l := s.lines[i]
	if local >= l.back {
		return 0
	}
	j := sort.Search(len(l.strokes), func(j int) bool { return l.strokes[j].at > local })
	if j > 0 && local-l.strokes[j-1].at < strike {
		return 0
	}
	next := l.back
	if j < len(l.strokes) {
		next = l.strokes[j].at
	}
	return next - local
}

// base is the letter a character is typed with: its first letter with
// marks taken off, in capitals, as the keys show it.
func base(r rune) rune {
	d := []rune(norm.NFD.String(string(r)))
	return unicode.ToUpper(d[0])
}
