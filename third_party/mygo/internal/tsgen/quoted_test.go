package tsgen

import (
	"encoding/json/v2"
	"reflect"
	"strings"
	"testing"
)

type quotedNumber int64

type customNumber int64

func (customNumber) MarshalJSON() ([]byte, error) { return []byte(`{"value":1}`), nil }

func TestQuotedNumberPointers(t *testing.T) {
	type payload struct {
		Count    *int64        `json:"count,string"`
		Nested   **int64       `json:"nested,string"`
		Optional *int64        `json:"optional,string,omitempty"`
		Named    *quotedNumber `json:"named,string"`
		Custom   *customNumber `json:"custom,string"`
	}
	n := int64(9007199254740993)
	p := &n
	named, custom := quotedNumber(n), customNumber(1)
	value := payload{Count: p, Nested: &p, Named: &named, Custom: &custom}
	wire, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"count":"9007199254740993","nested":"9007199254740993","named":"9007199254740993","custom":{"value":1}}`; string(wire) != want {
		t.Fatalf("wire = %s", wire)
	}
	wire, err = json.Marshal(payload{})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"count":null,"nested":null,"named":null,"custom":null}`; string(wire) != want {
		t.Fatalf("nil wire = %s", wire)
	}
	out, err := Generate(Model{Events: []Event{{Name: "payload", Type: reflect.TypeFor[payload]()}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"count: string | null;", "nested: string | null;", "optional?: string;", "named: string | null;", "custom: unknown | null;"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("client lacks %s:\n%s", want, out)
		}
	}
}
