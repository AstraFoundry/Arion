package models

import "testing"

func TestEmbeddedCatalogValid(t *testing.T) {
	catalog, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, bundle := range catalog.Bundles {
		if catalog.DownloadBytes(bundle) <= 0 {
			t.Fatalf("%s has no download size", bundle.ID)
		}
	}
}

// MOD-A02: missing provenance is rejected rather than defaulted.
func TestCatalogRejectsMissingProvenance(t *testing.T) {
	bad := []string{
		`{"version":1,"artifacts":[{"id":"a","purpose":"streaming-asr","source":{"url":"u","release":"r","sha256":"x","sizeBytes":1},"license":{"model":"m","runtime":"r","reference":"x"}}],"bundles":[]}`,
		`{"version":2,"artifacts":[],"bundles":[]}`,
		`{"version":1,"artifacts":[],"bundles":[{"id":"b","streaming":"missing","chunkMs":20,"endpointSilenceMs":800}]}`,
	}
	for _, data := range bad {
		if _, err := Parse([]byte(data)); err == nil {
			t.Fatalf("accepted %s", data)
		}
	}
}
