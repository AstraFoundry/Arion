//go:build darwin

// Command arion-ocr runs the production Vision text recognizer
// (macos.RecognizeImage) over a context corpus's window images at the
// accurate and fast levels and writes one JSON line per image and level.
// It is the screen-text source of the context benchmark.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"arion.local/desktop/infrastructure/macos"
)

type row struct {
	CaseID string  `json:"caseId"`
	Level  string  `json:"level"`
	Text   string  `json:"text"`
	OCRMs  float64 `json:"ocrMs"`
	Error  string  `json:"error,omitempty"`
}

func main() {
	corpusPath := flag.String("corpus", "", "context corpus JSON; images resolve beside it")
	out := flag.String("out", "", "output JSONL path (must not exist)")
	flag.Parse()
	if *corpusPath == "" || *out == "" {
		flag.Usage()
		os.Exit(2)
	}
	data, err := os.ReadFile(*corpusPath)
	check(err)
	var corpus struct {
		Samples []struct {
			ID             string `json:"id"`
			ContextFixture struct {
				Image string `json:"image"`
			} `json:"contextFixture"`
		}
	}
	check(json.Unmarshal(data, &corpus))
	file, err := os.OpenFile(*out, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	check(err)
	defer file.Close()
	encoder := json.NewEncoder(file)
	dir := filepath.Dir(*corpusPath)
	for _, s := range corpus.Samples {
		image, err := os.ReadFile(filepath.Join(dir, s.ContextFixture.Image))
		check(err)
		for _, level := range []struct {
			name  string
			level macos.OCRLevel
		}{{"accurate", macos.OCRAccurate}, {"fast", macos.OCRFast}} {
			text, err := macos.RecognizeImage(image, level.level)
			r := row{CaseID: s.ID, Level: level.name, Text: text.Text, OCRMs: text.OCRMs}
			if err != nil {
				r.Error = err.Error()
			}
			check(encoder.Encode(r))
			fmt.Fprintf(os.Stderr, "%-18s %-8s %6.0f ms %4d chars\n", s.ID, level.name, text.OCRMs, len([]rune(text.Text)))
		}
	}
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
