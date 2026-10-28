package domain

import "testing"

// Explicit adoption selects the requested native backend without borrowing
// unmeasured speed or accuracy from another backend/platform.
func TestExplicitAdoptionDoesNotInventMeasurements(t *testing.T) {
	option := ModelOption{ID: "native", Adopted: true, Platforms: []string{"darwin-arm64"}, Languages: []string{"zh", "en", "mixed"}}
	selection, err := SelectDefaultModel([]ModelOption{option}, DefaultSelectionPolicy("darwin-arm64"))
	if err != nil || selection.Option.ID != "native" || selection.LatencyMeasured || selection.Evaluation.RunID != "" {
		t.Fatalf("%+v %v", selection, err)
	}
	if _, err = SelectDefaultModel([]ModelOption{option}, DefaultSelectionPolicy("windows-amd64")); err == nil {
		t.Fatal("selected wrong platform")
	}
	if _, err = SelectDefaultModel([]ModelOption{option, option}, DefaultSelectionPolicy("darwin-arm64")); err == nil {
		t.Fatal("ambiguous adoption accepted")
	}
}
