package main

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"

	// MyGo's own strings in the languages of the gallery that MyGo is
	// translated to; its others show them in English.
	_ "github.com/egoist/mygo/locales/ja"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	_ "github.com/egoist/mygo/locales/zh-Hant"
)

// galleryLanguages are the languages the gallery declares, its own first:
// those MyGo's controls are translated to, and others whose formats and
// layout show. Its own text is English in every one.
var galleryLanguages = []string{"en", "zh-Hans", "zh-Hant", "de", "fr", "ja", "ar", "he", "hi"}

// languages is the Languages page: a menu of languages, as an app's
// settings would have one, numbers and dates in the language the window
// shows and the user's regional formats, and controls that say MyGo's own
// strings in it.
func (g *gallery) languages(c *ui.Context) {
	t := c.Theme()
	locale, region := c.Locale(), c.Region()
	formats := region.FormatLocale(locale)
	ui.Text(c, "The window shows one language: MyGo's controls, its menus and the formats of numbers and dates follow it, and it lays out right to left in Arabic and Hebrew. The gallery's own text stays English.").TextColor(t.TextMuted)
	card(c, "Language", func() {
		// The language chosen: "" follows the system's.
		ui.RadioGroup(c, func() {
			options := append([]string{""}, galleryLanguages...)
			options = append(options, i18n.PseudoAccents, i18n.PseudoBidi)
			for _, tag := range options {
				name := "System"
				switch tag {
				case "":
				case i18n.PseudoAccents:
					name = "Pseudo accents (en-XA)"
				case i18n.PseudoBidi:
					name = "Pseudo right to left (ar-XB)"
				default:
					name = i18n.Autonym(tag) + " (" + tag + ")"
				}
				if ui.Radio(c, &g.language, tag, name).Changed() && g.win != nil {
					mygo.App.SetLocale(g.language)
				}
			}
		}).Label("Language")
		ui.Textf(c, "Shows %s, %s; formats of %s: weeks start on %s, %d hours.",
			orDefault(locale, "en"), i18n.Direction(locale), orDefault(region.Locale, "the language"), region.FirstWeekday, region.HourCycle).
			TextColor(t.TextMuted).FontSize(13)
	})
	card(c, "Numbers and dates", func() {
		now := g.now
		rows := []struct{ label, value string }{
			{"Number", i18n.FormatNumber(formats, 1234567.891, i18n.NumberOptions{})},
			{"Percent", i18n.FormatNumber(formats, 0.256, i18n.NumberOptions{Percent: true})},
			{"Negative", i18n.FormatNumber(formats, -42.5, i18n.NumberOptions{MinFractionDigits: 2})},
			{"Full date", i18n.FormatDate(formats, now, i18n.DateOptions{Date: i18n.FullStyle})},
			{"Short date and time", i18n.FormatDate(formats, now, i18n.DateOptions{Date: i18n.ShortStyle, Time: i18n.ShortStyle})},
			{"Month and year", i18n.FormatDate(formats, now, i18n.DateOptions{Skeleton: "yMMMM"})},
			{"Time", i18n.FormatDate(formats, now, i18n.DateOptions{Skeleton: "jm"})},
		}
		for _, r := range rows {
			ui.Row(c).Gap(12).Children(func() {
				ui.Text(c, r.label).TextColor(t.TextMuted).Width(150).Shrink(0)
				ui.Text(c, r.value).Selectable()
			})
		}
		months := i18n.Months(locale, i18n.Abbreviated, true)
		ui.Row(c).Gap(12).Children(func() {
			ui.Text(c, "Months").TextColor(t.TextMuted).Width(150).Shrink(0)
			ui.Text(c, strings.Join(months[:], " ")).Grow(1)
		})
	})
	card(c, "The same, in other locales", func() {
		for _, l := range []string{"en-US", "en-GB", "de-DE", "fr-FR", "hi-IN", "ar-EG", "fa-IR", "ja-JP", "zh-Hans-CN"} {
			ui.Row(c).Gap(12).Children(func() {
				ui.Text(c, l).TextColor(t.TextMuted).Width(90).Shrink(0)
				ui.Text(c, i18n.FormatNumber(l, 1234567.891, i18n.NumberOptions{})).Width(150).Shrink(0)
				ui.Text(c, i18n.FormatDate(l, g.now, i18n.DateOptions{Date: i18n.MediumStyle, Time: i18n.ShortStyle})).Grow(1)
			})
		}
	})
	card(c, "Controls", func() {
		ui.Row(c).Gap(24).AlignItems(ui.Start).Wrap().Children(func() {
			ui.Calendar(c, &g.meeting)
			ui.Column(c).Gap(12).Children(func() {
				ui.Field(c, "Day", func() { ui.DateInput(c, &g.meeting) })
				ui.Field(c, "Time", func() { ui.TimeInput(c, &g.meeting) })
				ui.SearchField(c, &g.search).Label("Search")
			})
		})
	})
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
