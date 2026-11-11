package ui

import (
	"math"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestField(t *testing.T) {
	name, email, font, size := "", "", "Arial", "Small"
	notify, express := false, false
	emailError := "Enter an email address."
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(12).Children(func() {
			coreField(c, "Name", func() { coreTextInput(c, &name) }).Description("As on your passport.")
			coreField(c, "Email", func() { coreTextInput(c, &email) }).Description("For receipts.").Error(emailError)
			coreField(c, "Notifications", func() { coreCheckbox(c, &notify, "Send emails") })
			coreField(c, "Delivery", func() {
				coreRadioGroup(c, func() {
					coreRadio(c, &size, "Small", "Small")
					coreRadio(c, &size, "Large", "Large")
				})
			})
			coreField(c, "Font", func() { coreCombobox(c, &font, fonts) })
			coreField(c, "Express", func() { coreSwitch(c, &express) })
			coreButton(c, "Cut").Tooltip("Cut the selection")
		})
	}, 500, 700)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	// The label names the control, and the texts below describe it, the
	// error first.
	if n := accessNode(t, tree, platform.RoleTextField, "Name"); n.Description != "As on your passport." || n.States&platform.AccessInvalid != 0 {
		t.Errorf("Name: %+v", n)
	}
	if n := accessNode(t, tree, platform.RoleTextField, "Email"); n.Description != emailError+"\nFor receipts." || n.States&platform.AccessInvalid == 0 {
		t.Errorf("Email: %+v", n)
	}
	if !tt.HasText(emailError) || !tt.HasText("For receipts.") {
		t.Errorf("the texts below do not show: %q", tt.Texts())
	}
	// A check box names itself; the field names the group around it.
	accessNode(t, tree, platform.RoleCheckBox, "Send emails")
	accessNode(t, tree, platform.RoleGroup, "Notifications")
	accessNode(t, tree, platform.RoleRadioGroup, "Delivery")
	accessNode(t, tree, platform.RoleComboBox, "Font")
	accessNode(t, tree, platform.RoleSwitch, "Express")
	if n := accessNode(t, tree, platform.RoleButton, "Cut"); n.Description != "Cut the selection" {
		t.Errorf("the tooltip does not describe the button: %+v", n)
	}
	// A click on the label focuses the control, or clicks it.
	tt.Click("Name")
	if !tt.Focused("Name") {
		t.Error("a click on the label did not focus the input")
	}
	tt.Type("Ada")
	if name != "Ada" {
		t.Errorf("typed %q", name)
	}
	// The text, not the group it names.
	var label Rect
	for _, n := range tt.rt.labels {
		if n.text == "Notifications" {
			label = n.r
		}
	}
	tt.ClickAt(label.X+label.W/2, label.Y+label.H/2)
	if !notify {
		t.Error("a click on the label did not check the check box")
	}
	tt.Click("Express")
	if !express {
		t.Error("a click on the label did not turn the switch on")
	}
	// In a group, the one chosen.
	tt.Click("Delivery")
	if !tt.Focused("Small") || size != "Small" {
		t.Errorf("a click on the label of a radio group: focused %v, chose %q", tt.Focused("Small"), size)
	}
}

func TestFieldDisabled(t *testing.T) {
	name, saved := "", 0
	tt := coreNewTester(func(c *context) {
		coreFieldset(c, "Account", func() {
			coreField(c, "Name", func() { coreTextInput(c, &name) })
			if coreButton(c, "Save").Clicked() {
				saved++
			}
		}).Disabled(true)
	}, 400, 300)
	// The fieldset disables what is inside it, though it does so after
	// building it.
	tt.Click("Name")
	if tt.Focused("Name") {
		t.Error("a click on the label focused a disabled input")
	}
	tt.Click("Save")
	if saved != 0 || tt.Focused("Save") {
		t.Errorf("a disabled button: %d clicks, focused %v", saved, tt.Focused("Save"))
	}
	r, _ := tt.Find("Name")
	tt.ClickAt(r.X+r.W/2, r.Y+r.H+20) // the input
	if tt.Focused("Name") {
		t.Error("a click focused a disabled input")
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if n := accessNode(t, tt.h.access, platform.RoleGroup, "Account"); n.States&platform.AccessDisabled == 0 {
		t.Errorf("the fieldset: %+v", n)
	}
}

// TestLabelledBy checks that a text the view draws, given to a control
// with LabelledBy, names the control and takes a click to it, as HTML's
// <label for> (#1309).
func TestLabelledBy(t *testing.T) {
	card, query, size := "", "", "Small"
	agree, off := false, false
	var texts = map[string]uint64{}
	text := func(c *context, s string) *node {
		l := coreText(c, s)
		texts[s] = l.id
		return l
	}
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Gap(12).Children(func() {
			l := text(c, "Card number")
			coreTextInput(c, &card).Label("Number").LabelledBy(l)
			coreRow(c).Gap(8).Children(func() {
				box := coreCheckbox(c, &agree, "")
				box.LabelledBy(text(c, "I agree")) // a label after its control
			})
			coreRow(c).Gap(8).Children(func() {
				coreRadio(c, &size, "Small", "")
				coreRadio(c, &size, "Large", "").LabelledBy(text(c, "Large size"))
			})
			coreSearchField(c, &query).LabelledBy(text(c, "Find"))
			coreCheckbox(c, &off, "").Disabled(true).LabelledBy(text(c, "Off"))
		})
	}, 500, 600)
	clickText := func(s string) {
		t.Helper()
		for _, n := range tt.rt.labels {
			if n.id == texts[s] {
				tt.ClickAt(n.r.X+n.r.W/2, n.r.Y+n.r.H/2)
				return
			}
		}
		t.Fatalf("no text %q", s)
	}
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	// The label names the control, in place of its own Label.
	if n := accessNode(t, tree, platform.RoleTextField, "Card number"); n.Label != "Card number" {
		t.Errorf("the input: %+v", n)
	}
	accessNode(t, tree, platform.RoleCheckBox, "I agree")
	accessNode(t, tree, platform.RoleRadio, "Large size")
	if n := accessNode(t, tree, platform.RoleTextField, "Find"); n.Label != "Find" {
		t.Errorf("the search field's input: %+v", n)
	}
	// A click on the label focuses a text input, and checks a check box
	// or chooses a radio button.
	clickText("Card number")
	if tt.rt.focused == 0 || !tt.Focused("Card number") {
		t.Fatal("a click on the label did not focus the input")
	}
	tt.Type("4242")
	if card != "4242" {
		t.Errorf("typed %q", card)
	}
	clickText("I agree")
	if !agree {
		t.Error("a click on the label did not check the check box")
	}
	clickText("I agree")
	if agree {
		t.Error("a second click on the label did not uncheck the check box")
	}
	clickText("Large size")
	if size != "Large" {
		t.Errorf("a click on the label chose %q", size)
	}
	clickText("Find")
	tt.Type("go")
	if query != "go" {
		t.Errorf("a click on the label of a search field, then typing, gave %q", query)
	}
	// A disabled control takes no click from its label.
	clickText("Off")
	if off {
		t.Error("a click on the label checked a disabled check box")
	}
}

func TestFormLayout(t *testing.T) {
	name, email, city := "", "", ""
	notify, express := false, false
	tt := coreNewTester(func(c *context) {
		coreForm(c, func() {
			coreField(c, "Name", func() { coreTextInput(c, &name) })
			coreField(c, "Email address", func() { coreTextInput(c, &email) }).Description("For receipts.")
			coreField(c, "Notifications", func() { coreCheckbox(c, &notify, "Send emails") })
			// A description under a control without text leaves its label
			// level with the control, not with the description.
			coreField(c, "Express", func() { coreSwitch(c, &express) }).Description("Ships the next day.")
			// Beside a dot drawn before it, the text of a row lines up
			// with the label.
			coreField(c, "Status", func() {
				coreRow(c).Gap(6).AlignItems(Start).Children(func() {
					coreBox(c).Width(8).Height(8).Background(RGB(0, 200, 0))
					coreText(c, "Online")
				})
			})
			coreFieldset(c, "Shipping", func() {
				coreField(c, "City", func() { coreTextInput(c, &city) })
			})
		}).Width(480)
	}, 500, 500)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	tree := tt.h.access
	box := func(role platform.AccessRole, name string) platform.RectF {
		return accessNode(t, tree, role, name).Bounds
	}
	find := func(s string) Rect {
		r, ok := tt.Find(s)
		if !ok {
			t.Fatalf("no %q", s)
		}
		return r
	}
	near := func(a, b float64) bool { return math.Abs(a-b) < 1 }
	// The controls start at one edge, beside the widest label, those of a
	// fieldset too, and the labels end at one.
	nameIn, emailIn, cityIn := box(platform.RoleTextField, "Name"), box(platform.RoleTextField, "Email address"), box(platform.RoleTextField, "City")
	if !near(nameIn.X, emailIn.X) || !near(nameIn.X, cityIn.X) || !near(nameIn.X, box(platform.RoleCheckBox, "Send emails").X) {
		t.Errorf("the controls start at %v, %v, %v", nameIn.X, emailIn.X, cityIn.X)
	}
	nameL, emailL := find("Name"), find("Email address")
	if !near(float64(nameL.X+nameL.W), float64(emailL.X+emailL.W)) || float64(emailL.X+emailL.W) > emailIn.X {
		t.Errorf("the labels end at %v and %v, the inputs start at %v", nameL.X+nameL.W, emailL.X+emailL.W, emailIn.X)
	}
	// Labels are level with the text of their control: inside the input's
	// border and padding, on the check box's text, and centered on a
	// switch.
	if !near(float64(nameL.Y), nameIn.Y+7) || !near(float64(emailL.Y), emailIn.Y+7) {
		t.Errorf("the labels are at %v and %v, the inputs at %v and %v", nameL.Y, emailL.Y, nameIn.Y, emailIn.Y)
	}
	if notifyL, own := find("Notifications"), find("Send emails"); !near(float64(notifyL.Y), float64(own.Y)) {
		t.Errorf("the label is at %v, the check box's text at %v", notifyL.Y, own.Y)
	}
	sw := box(platform.RoleSwitch, "Express")
	if l := find("Express"); math.Abs(float64(l.Y+l.H/2)-(sw.Y+sw.H/2)) > 1.5 {
		t.Errorf("the label's center is at %v, the switch's at %v", l.Y+l.H/2, sw.Y+sw.H/2)
	}
	if l, v := find("Status"), find("Online"); !near(float64(l.Y), float64(v.Y)) {
		t.Errorf("the label is at %v, the text beside the dot at %v", l.Y, v.Y)
	}
	// The description sits below its input, in the controls' column.
	if d := find("For receipts."); !near(float64(d.X), emailIn.X) || float64(d.Y) < emailIn.Y+emailIn.H {
		t.Errorf("the description is at %v, the input at %v", d, emailIn)
	}
}
