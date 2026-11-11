// Markdown-chat is a chat with a pretend language model whose replies
// stream in token by token as Markdown: markdown.Stream shows a reply
// while it comes, markdown.View once it is done, and the list follows the
// end until the user scrolls up.
//
//	go run ./examples/markdown-chat
package main

import (
	"log"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

type message struct {
	id   int
	mine bool
	text string
	done bool
}

type chat struct {
	win      *mygo.Window
	messages []*message
	input    string
	list     ui.ListState
	md       markdown.Options
	next     int
	replies  int
	// reveal is how replies come in: by word, sentence, line or
	// paragraph, or at once.
	reveal int
}

var (
	revealNames = []string{"Word", "Sentence", "Line", "Paragraph", "At once"}
	revealUnits = []markdown.RevealUnit{markdown.ByWord, markdown.BySentence, markdown.ByLine, markdown.ByParagraph, markdown.AtOnce}
)

func (app *chat) view(c *ui.Context) {
	t := c.Theme()
	app.list.Key = func(i int) any { return app.messages[i].id }
	app.list.FollowEnd = true
	app.list.FollowAnimation = &ui.Smooth // glide up as the reply grows
	app.md.Reveal = &markdown.Reveal{Unit: revealUnits[app.reveal]}
	ui.Column(c).Fill().Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(8, 20).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Text(c, "Stream by").TextColor(t.TextMuted)
			ui.Segmented(c, &app.reveal, revealNames...).Label("Stream by")
		})
		ui.List(c, &app.list, len(app.messages), func(i int) {
			app.message(c, app.messages[i])
		}).Grow(1).Padding(16, 20).Gap(16).Justify(ui.End).Children(func() {
			if len(app.messages) == 0 {
				ui.Text(c, "Ask anything: the replies are canned, streamed as a model streams them.").TextColor(t.TextMuted).Margin(ui.Auto)
			}
		})
		ui.Row(c).Gap(8).Padding(12, 20).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Children(func() {
			in := ui.TextInput(c, &app.input).Placeholder("Message").Grow(1).AutoFocus()
			streaming := len(app.messages) > 0 && !app.messages[len(app.messages)-1].done
			if streaming {
				if ui.Button(c, "Stop").Clicked() {
					app.messages[len(app.messages)-1].done = true
				}
			} else if (ui.PrimaryButton(c, "Send").Disabled(strings.TrimSpace(app.input) == "").Clicked() || in.Submitted()) && strings.TrimSpace(app.input) != "" {
				app.send()
			}
		})
	})
}

// message builds a message: the user's in a bubble on the right, the
// model's as Markdown.
func (app *chat) message(c *ui.Context, m *message) {
	t := c.Theme()
	if m.mine {
		ui.Row(c).Children(func() {
			ui.Spacer(c)
			ui.Text(c, m.text).Selectable().Padding(8, 12).Radius(14).Background(t.Surface).MaxWidth(520)
		})
		return
	}
	ui.Column(c).MaxWidth(760).Children(func() {
		if m.done {
			markdown.View(c, m.text, &app.md)
		} else {
			markdown.Stream(c, m.text, &app.md)
		}
	})
}

// send sends the input, and streams the next reply in.
func (app *chat) send() {
	app.next++
	app.messages = append(app.messages, &message{id: app.next, mine: true, text: strings.TrimSpace(app.input), done: true})
	app.input = ""
	app.next++
	reply := &message{id: app.next}
	app.messages = append(app.messages, reply)
	src := replies[app.replies%len(replies)]
	app.replies++
	if app.win == nil {
		reply.text, reply.done = src, true
		return
	}
	go func() {
		for i := 0; i < len(src); {
			// Tokens of a few bytes, at the uneven pace of a model.
			n := min(len(src)-i, 2+rand.IntN(7))
			for n < len(src)-i && src[i+n]&0xC0 == 0x80 {
				n++
			}
			i += n
			text := src[:i]
			time.Sleep(time.Duration(8+rand.IntN(30)) * time.Millisecond)
			stopped := false
			app.win.Update(func() {
				if reply.done {
					stopped = true
					return
				}
				reply.text = text
			})
			if stopped {
				return
			}
		}
		app.win.Update(func() { reply.done = true })
	}()
}

var replies = []string{
	"Here is a minimal **LRU cache** in Go, safe for concurrent use:\n\n" +
		"```go\ntype LRU struct {\n\tmu    sync.Mutex\n\tcap   int\n\titems map[string]*list.Element\n\torder *list.List\n}\n\n" +
		"func (c *LRU) Get(key string) (any, bool) {\n\tc.mu.Lock()\n\tdefer c.mu.Unlock()\n\tif e, ok := c.items[key]; ok {\n\t\tc.order.MoveToFront(e)\n\t\treturn e.Value.(*entry).value, true\n\t}\n\treturn nil, false\n}\n```\n\n" +
		"A few things to note:\n\n1. `container/list` keeps the order of use in O(1).\n2. The mutex makes `Get` a writer too, since it moves the entry.\n3. For heavy read traffic, consider **sharding** by key hash.\n",
	"## Comparing the options\n\n| Strategy | Read latency | Staleness | Complexity |\n|---|--:|:-:|---|\n| No cache | ~41 ms | none | trivial |\n| Cache-aside | ~0.2 ms | medium | low |\n| Write-through | ~0.2 ms | low | medium |\n\n" +
		"> **Tip:** start with *cache-aside*; move to write-through once staleness bites.\n\nSee the [Go blog on profiling](https://go.dev/blog/pprof) to find what to cache first.",
	"Sure. The area of a circle is $$A = \\pi r^2$$ and its circumference $$C = 2\\pi r$$.\n\n" +
		"- [x] Measure the radius\n- [x] Square it\n- [ ] Multiply by π\n\nThat's all there is to it~ unless you need ~~approximations~~ the exact value, which `math.Pi` gives to 15 digits.",
}

func main() {
	app := &chat{}
	mygo.App.WhenReady(func() {
		app.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Markdown chat",
			Width:   820,
			Height:  720,
			Content: ui.View(app.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
