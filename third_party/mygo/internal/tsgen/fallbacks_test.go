package tsgen

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"
)

type fallbackMap struct {
	Name  string         `json:"name"`
	Extra map[string]int `json:",embed"`
}

type fallbackRaw struct {
	Name  string         `json:"name"`
	Extra jsontext.Value `json:",embed"`
}

func TestEmbeddedFallbacks(t *testing.T) {
	extra := map[string]int{"count": 2}
	for _, value := range []any{
		fallbackMap{Name: "notes", Extra: extra},
		fallbackRaw{Name: "notes", Extra: jsontext.Value(`{"count":2}`)},
		struct {
			Name  string          `json:"name"`
			Extra *map[string]int `json:",embed"`
		}{"notes", &extra},
	} {
		wire, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if string(wire) != `{"name":"notes","count":2}` {
			t.Fatalf("wire = %s", wire)
		}
		typ := reflect.TypeOf(value)
		if err := Validate(typ); err != nil {
			t.Fatal(err)
		}
		out, err := Generate(Model{Events: []Event{{Name: "payload", Type: typ}}})
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"name: string", "[key: string]: unknown"} {
			if !strings.Contains(string(out), want) {
				t.Errorf("%v: client lacks %s:\n%s", typ, want, out)
			}
		}
		if strings.Contains(string(out), "Extra:") {
			t.Errorf("%v: client exposes a nested fallback:\n%s", typ, out)
		}
	}
}

func TestEmbeddedFallbackValidation(t *testing.T) {
	type bad struct {
		Extra map[string]func() `json:",embed"`
	}
	if err := Validate(reflect.TypeFor[bad]()); err == nil {
		t.Fatal("embedded map values bypassed validation")
	}
}
