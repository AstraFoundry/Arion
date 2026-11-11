package tsgen

import (
	"reflect"
	"strings"
	"testing"
)

func TestServiceExportNames(t *testing.T) {
	for _, tc := range []struct {
		name          string
		events, valid bool
	}{
		{"await", false, false}, {"default", false, false}, {"class", false, false},
		{"call", false, false}, {"event", true, false}, {"events", true, false},
		{"event", false, true}, {"events", false, true}, {"Tasks", true, true},
	} {
		name := tc.name
		if tc.events {
			name += "WithEvents"
		}
		t.Run(name, func(t *testing.T) {
			m := Model{Services: []Service{{Name: tc.name, Type: reflect.TypeFor[struct{}](), Methods: []Method{{Name: "Run"}}}}}
			if tc.events {
				m.Events = []Event{{Name: "changed", Type: reflect.TypeFor[string]()}}
			}
			out, err := Generate(m)
			if tc.valid {
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(out), `export const `+tc.name+` =`) || !strings.Contains(string(out), `"`+tc.name+`.Run"`) {
					t.Fatalf("lost service name: %s", out)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.name) {
				t.Fatalf("Generate must report service %q rather than write an invalid binding: %v\n%s", tc.name, err, out)
			}
		})
	}
}
