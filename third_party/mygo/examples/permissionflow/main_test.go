package main

import (
	"testing"

	"github.com/egoist/mygo/plugins/permissionflow"
	"github.com/egoist/mygo/ui"
)

// The window lists the permissions with their status, and its buttons ask
// for those not granted.
func TestPermissions(t *testing.T) {
	var asked, guided []permissionflow.Permission
	m := &model{
		rows: []*row{
			{p: permissionflow.Accessibility, status: permissionflow.Granted},
			{p: permissionflow.Microphone, status: permissionflow.NotDetermined},
		},
		request: func(p permissionflow.Permission) { asked = append(asked, p) },
		guide:   func(p permissionflow.Permission) { guided = append(guided, p) },
	}
	tt := ui.NewTester(m.view, 520, 640)
	for _, s := range []string{"Accessibility", "granted", "Microphone", "not determined"} {
		if _, ok := tt.Find(s); !ok {
			t.Errorf("%q isn't shown", s)
		}
	}
	// Granted, Accessibility's buttons are disabled.
	if err := tt.Click("Request Accessibility"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("Request Microphone"); err != nil {
		t.Fatal(err)
	}
	m.rows[1].busy = false
	tt.Frame()
	if err := tt.Click("Guide to Microphone"); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0] != permissionflow.Microphone || len(guided) != 1 || guided[0] != permissionflow.Microphone {
		t.Errorf("asked %v, guided %v: want the microphone once each, and Accessibility never", asked, guided)
	}

	empty := ui.NewTester((&model{}).view, 520, 640)
	if _, ok := empty.Find("This system keeps none of these permissions for apps."); !ok {
		t.Error("an empty list says nothing")
	}
}
