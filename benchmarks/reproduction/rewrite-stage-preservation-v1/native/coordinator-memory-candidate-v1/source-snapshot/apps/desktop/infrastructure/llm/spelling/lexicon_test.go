package spelling

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"arion.local/core/domain"
)

func TestEvaluatedProducerParity(t *testing.T) {
	data, err := os.ReadFile("../../../../../benchmarks/reproduction/rewrite-product-integration-v1/producer-parity.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []domain.SpellingPreparation
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	l, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 457 {
		t.Fatal("incomplete producer parity corpus")
	}
	for i, want := range cases {
		got := l.Prepare(want.Source, want.Context, want.Scene)
		if !reflect.DeepEqual(got, want) {
			a, _ := json.Marshal(got)
			b, _ := json.Marshal(want)
			t.Fatalf("case %d\ngot %s\nwant %s", i, a, b)
		}
	}
}

func TestSourceIdentityAndAmbiguity(t *testing.T) {
	l, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	p := l.Prepare("请联系陈雨南。", map[string]string{"Screen": "陈语楠 陈宇南"}, "work")
	if len(p.Hypotheses) != 1 || p.Hypotheses[0].Status != "ambiguous" {
		t.Fatalf("%+v", p)
	}
	if _, err := domain.ApplySpellingChoices(`{"e0":"replace"}`, p); err == nil {
		t.Fatal("accepted ambiguous replacement")
	}
	p.Hypotheses[0].Source = "other source"
	if _, err := domain.ApplySpellingChoices(`{"e0":"keep"}`, p); err == nil {
		t.Fatal("accepted mismatched source")
	}
}
