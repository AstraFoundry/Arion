package notification

import (
	"encoding/xml"
	"github.com/egoist/mygo/internal/platform"
	"strings"
	"testing"
)

func TestActivationEnvelope(t *testing.T) {
	for _, a := range []string{"", "view", "later"} {
		value := Activation("beta/2.0.0-beta.1/available", a)
		id, action, err := ParseActivation(value)
		if err != nil || id != "beta/2.0.0-beta.1/available" || action != a {
			t.Fatalf("round trip = %q %q %v", id, action, err)
		}
	}
	for _, bad := range []string{"https://untrusted.invalid", activationPrefix + "id=x&action=view&url=https://bad.invalid", activationPrefix + "id=a&id=b&action=view", activationPrefix + "id=x&action=View", activationPrefix + "id=x%00&action=view", activationPrefix + "id=" + strings.Repeat("x", 257) + "&action=view", activationPrefix + "id=x&action=../run", activationPrefix + "id=x&action=%00"} {
		if _, _, err := ParseActivation(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}
func TestStableIdentityAndTag(t *testing.T) {
	id := ActivatorCLSID("si.astrafoundry.arion")
	if len(id) != 36 || id != ActivatorCLSID("si.astrafoundry.arion") || id == ActivatorCLSID("si.astrafoundry.arion.notification-probe") {
		t.Fatal(id)
	}
	if len(Tag("stable:1.0.0:ready")) != 16 || Tag("x") == Tag("y") {
		t.Fatal("invalid stable tag")
	}
}
func TestToastXMLIsEscapedAndCOMActivated(t *testing.T) {
	n := &platform.Notification{ID: "beta/1.2.3/available", Title: `Test <"&`, Body: `body & <script>`, Silent: true, Actions: []platform.NotificationAction{{ID: "view", Title: `View <&"`}, {ID: "later", Title: "Later"}}}
	value, err := ToastXML(n)
	if err != nil {
		t.Fatal(err)
	}
	var toast struct {
		Launch     string `xml:"launch,attr"`
		Activation string `xml:"activationType,attr"`
		Audio      struct {
			Silent string `xml:"silent,attr"`
		} `xml:"audio"`
		Actions []struct {
			Arguments string `xml:"arguments,attr"`
			Content   string `xml:"content,attr"`
		} `xml:"actions>action"`
	}
	if err := xml.Unmarshal([]byte(value), &toast); err != nil {
		t.Fatal(err)
	}
	if toast.Activation != "foreground" || toast.Audio.Silent != "true" || len(toast.Actions) != 2 {
		t.Fatalf("%+v", toast)
	}
	if toast.Actions[0].Content != n.Actions[0].Title {
		t.Fatal("label was not XML escaped")
	}
	for _, a := range toast.Actions {
		id, _, err := ParseActivation(a.Arguments)
		if err != nil || id != n.ID {
			t.Fatalf("invalid arguments %q", a.Arguments)
		}
	}
}
