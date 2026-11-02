package ranking

import (
	"arion.local/core/contracts"
	"arion.local/core/domain"
	"context"
	"math"
	"testing"
)

type scoreFunc func(context.Context, string, string, []string) ([]float64, error)

func (f scoreFunc) ScoreKeyboard(c context.Context, b, a string, t []string) ([]float64, error) {
	return f(c, b, a, t)
}
func TestComparisonScoresAndStableMargin(t *testing.T) {
	req := contracts.KeyboardRankRequest{Identity: contracts.Identity{SessionID: "test", Generation: 1}, Revision: 1, Input: "shishi", Before: "请", Candidates: []domain.KeyboardCandidate{
		{Candidate: domain.Candidate{ID: "c0", Text: "实施", Source: "phrase"}, End: 6}, {Candidate: domain.Candidate{ID: "c1", Text: "事实", Source: "phrase"}, End: 6}}}
	for _, test := range []struct {
		scores []float64
		want   string
	}{{[]float64{1, 1}, "c0"}, {[]float64{1, 1.1}, "c0"}, {[]float64{1, 2}, "c1"}, {[]float64{1}, ""}, {[]float64{math.NaN(), 1}, ""}, {[]float64{1, math.Inf(1)}, ""}} {
		order, err := Order(context.Background(), scoreFunc(func(context.Context, string, string, []string) ([]float64, error) { return test.scores, nil }), req, "")
		if test.want == "" {
			if err == nil {
				t.Fatal("invalid scores admitted")
			}
		} else if err != nil || order[0] != test.want {
			t.Fatal(order, err)
		}
	}
}
