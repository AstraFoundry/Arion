package modelruntime

import (
	"strings"
	"testing"

	"arion.local/core/domain"
	"arion.local/models"
)

func catalogJSON(evaluations string) string {
	sha := strings.Repeat("a", 64)
	artifact := func(id, purpose, platforms string) string {
		return `{"id":"` + id + `","purpose":"` + purpose + `","kind":"online-transducer","runtime":{"engine":"sherpa-onnx","version":"1"},
			"source":{"url":"https://example.invalid/` + id + `","release":"r","sha256":"` + sha + `","sizeBytes":100},
			"license":{"model":"m","runtime":"r","reference":"x"},"languages":["zh","en","mixed"],"platforms":[` + platforms + `],
			"audio":{"sampleRate":16000,"channels":1},"files":{"tokens":{"member":"t","sha256":"` + sha + `","sizeBytes":1}}}`
	}
	return `{"version":1,"artifacts":[` + artifact("s", "streaming-asr", `"darwin-arm64","windows-amd64"`) + `,` + artifact("f", "offline-asr", `"darwin-arm64"`) + `],
		"bundles":[{"id":"stream","name":"Stream","streaming":"s","chunkMs":20,"endpointSilenceMs":800,"evaluations":[` + evaluations + `]},
		{"id":"final","name":"Final","streaming":"s","finalizer":"f","chunkMs":20,"endpointSilenceMs":800,"evaluations":[]}]}`
}

// The catalog maps to selection inputs: platforms are the intersection of a
// bundle's artifacts and download size is their sum.
func TestOptionsFromCatalog(t *testing.T) {
	catalog, err := models.Parse([]byte(catalogJSON(`{"runId":"r","report":"x","hardware":"darwin-arm64","split":"dev","errorRates":{"zh":0.1,"en":0.1,"mixed":0.2},
		"stopToReviewP95Ms":120,"firstPartialP95Ms":900,"realTimeFactor":0.2,"observations":118}`)))
	if err != nil {
		t.Fatal(err)
	}
	options := Options(catalog)
	if len(options) != 2 {
		t.Fatalf("options %d", len(options))
	}
	final := options[1]
	if final.SizeBytes != 200 || len(final.Platforms) != 1 || final.Platforms[0] != "darwin-arm64" {
		t.Fatalf("finalizer bundle mapping wrong: %+v", final)
	}
	selection, err := domain.SelectDefaultModel(options, domain.DefaultSelectionPolicy("darwin-arm64"))
	if err != nil || selection.Option.ID != "stream" || !selection.LatencyMeasured {
		t.Fatalf("selection %+v %v", selection, err)
	}
}

// Without any evaluation the app reports it explicitly instead of picking.
func TestNoEvaluationIsExplicit(t *testing.T) {
	catalog, err := models.Parse([]byte(catalogJSON("")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := domain.SelectDefaultModel(Options(catalog), domain.DefaultSelectionPolicy("darwin-arm64")); domain.CodeOf(err) != domain.CodeEvaluationUnavailable {
		t.Fatalf("got %v", err)
	}
}

// Explicitly adopted native backends do not borrow evaluations from retired runtimes.
func TestEmbeddedNativeAdoption(t *testing.T) {
	catalog, err := models.Load()
	if err != nil {
		t.Fatal(err)
	}
	bundle, selection, err := SelectDefault(catalog)
	if err != nil || !bundle.Adopted || selection.LatencyMeasured {
		t.Fatalf("adoption: %+v %+v %v", bundle, selection, err)
	}
	for _, artifact := range catalog.Artifacts {
		if artifact.Kind != "qwen3-asr-mlx" && artifact.Kind != "qwen3-asr-gguf" {
			t.Fatalf("retired artifact: %s", artifact.Kind)
		}
	}
}
