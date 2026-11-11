//go:build darwin

package darwin

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestCommunicationIntent(t *testing.T) {
	load()
	var avatar bytes.Buffer
	if err := png.Encode(&avatar, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	withPool(func() {
		if communicationAuthorized() {
			t.Fatal("a plain Go test executable has no communication entitlement")
		}
		n := &platform.Notification{ID: "mail:7", Group: "thread:3", Title: "小明", Subtitle: "午饭", Body: "十二点见",
			Sender: &platform.NotificationSender{ID: "ming@example.com", Name: "小明", Avatar: avatar.Bytes()}}
		intent := communicationIntent(n)
		if intent == 0 {
			t.Fatal("no communication intent")
		}
		defer send(intent, "release")
		person := send(intent, "sender")
		if goString(send(person, "displayName")) != "小明" || goString(send(person, "customIdentifier")) != "ming@example.com" || send(person, "image") == 0 || sendBool(person, "isMe") {
			t.Fatal("incorrect sender identity or avatar")
		}
		if goString(send(send(person, "personHandle"), "value")) != "ming@example.com" || goString(send(intent, "conversationIdentifier")) != "thread:3" || send(send(intent, "recipients"), "count") != 0 {
			t.Fatal("incorrect sender handle or direct conversation")
		}
		if goString(send(intent, "content")) != "午饭\n十二点见" {
			t.Fatal("the subject or body was lost")
		}
		interaction := communicationInteraction(intent)
		defer send(interaction, "release")
		if send(interaction, "direction") != 2 {
			t.Fatal("received messages must donate an incoming interaction")
		}
		content := autorelease(alloc("UNMutableNotificationContent"))
		send(content, "setBody:", uintptr(nsString("original")))
		send(content, "setSound:", uintptr(send(class("UNNotificationSound"), "defaultSound")))
		send(content, "setThreadIdentifier:", uintptr(nsString("thread:3")))
		updated := communicationContent(content, intent)
		if goString(send(updated, "body")) != "午饭\n十二点见" || send(updated, "sound") == 0 || goString(send(updated, "threadIdentifier")) != "thread:3" {
			t.Fatal("communication conversion lost content, sound or grouping")
		}
		if goString(send(content, "body")) != "original" {
			t.Fatal("preparing communication changed the ordinary fallback")
		}
		b := new(Backend)
		if b.addCommunicationNotification(n, content, func(error) { t.Fatal("unentitled donation completed") }) || len(b.notifyPreparing) != 0 {
			t.Fatal("an unentitled process started donating a communication")
		}
		n.Group, n.Subtitle = "", ""
		fallback := communicationIntent(n)
		defer send(fallback, "release")
		if goString(send(fallback, "conversationIdentifier")) != "mail:7" || goString(send(fallback, "content")) != "十二点见" || goString(send(send(fallback, "sender"), "displayName")) != "小明" {
			t.Fatal("notification ID default, sender name or empty subject were lost")
		}
		for _, sender := range []*platform.NotificationSender{nil, {Name: "Ada", Avatar: avatar.Bytes()}, {ID: "ada"}, {ID: "ada", Avatar: []byte("not an image")}} {
			n.Sender = sender
			if intent := communicationIntent(n); intent != 0 {
				send(intent, "release")
				t.Fatal("incomplete sender must fall back to a plain notification")
			}
		}
	})
}

func TestCommunicationNotificationCancellation(t *testing.T) {
	load()
	calls := 0
	p := &preparingNotification{done: func(error) { calls++ }}
	b := &Backend{notifyPreparing: map[string]func(){"mail:7": func() { p.finish(nil) }}}
	b.RemoveNotification("mail:7")
	p.finish(nil) // a late donation completion must not complete Show twice
	if calls != 1 || len(b.notifyPreparing) != 0 || p.done != nil {
		t.Fatal("Close left a pending communication or completed it twice")
	}
	p = &preparingNotification{done: func(error) { calls++ }}
	b.notifyPreparing["mail:8"] = func() { p.finish(nil) }
	b.RemoveAllNotifications()
	p.finish(nil)
	if calls != 2 || len(b.notifyPreparing) != 0 || p.done != nil {
		t.Fatal("Clear left a pending communication or completed it twice")
	}
}

func TestCommunicationAvatarPNG(t *testing.T) {
	load()
	withPool(func() {
		svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><rect width="16" height="16" fill="#2563eb"/></svg>`)
		var bitmap bytes.Buffer
		if err := png.Encode(&bitmap, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
			t.Fatal(err)
		}
		for _, input := range [][]byte{svg, bitmap.Bytes()} {
			data := communicationAvatar(input)
			img, err := png.Decode(bytes.NewReader(goBytes(data)))
			if err != nil || img.Bounds().Empty() {
				t.Fatalf("notification avatar is not a raster PNG: %v", err)
			}
		}
		for _, input := range [][]byte{nil, []byte("not an image")} {
			if communicationAvatar(input) != 0 {
				t.Fatal("invalid avatar must keep the ordinary notification")
			}
		}
	})
}
