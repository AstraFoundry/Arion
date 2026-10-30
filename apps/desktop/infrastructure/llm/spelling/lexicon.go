// Package spelling loads the pinned pronunciation resources for the local
// spelling-hypothesis producer. It performs no inference or network access.
package spelling

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"arion.local/core/domain"
)

//go:embed data/pypinyin-character-readings-v1-readings.json
var readings []byte

//go:embed data/cmudict-pronunciation-v1-cmudict.dict
var dictionary string

var loadReadings = sync.OnceValues(func() (map[string][]string, error) {
	var result map[string][]string
	err := json.Unmarshal(readings, &result)
	return result, err
})

var load = sync.OnceValues(func() (*domain.SpellingLexicon, error) {
	l := &domain.SpellingLexicon{English: map[string][][]string{}}
	var err error
	l.Readings, err = CharacterReadings()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(strings.NewReader(dictionary))
	for scanner.Scan() {
		fields := strings.Fields(strings.SplitN(scanner.Text(), "#", 2)[0])
		if len(fields) < 2 {
			continue
		}
		word := strings.ToLower(fields[0])
		if i := strings.LastIndex(word, "("); i >= 0 {
			word = word[:i]
		}
		phones := []string{}
		for _, p := range fields[1:] {
			phones = append(phones, strings.TrimRight(p, "0123456789"))
		}
		duplicate := false
		for _, p := range l.English[word] {
			if strings.Join(p, " ") == strings.Join(phones, " ") {
				duplicate = true
				break
			}
		}
		if !duplicate {
			l.English[word] = append(l.English[word], phones)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("pronunciation dictionary: %w", err)
	}
	return l, nil
})

// Load returns immutable resources shared by all requests in the worker.
func Load() (*domain.SpellingLexicon, error) { return load() }

// CharacterReadings returns the pinned character dictionary in its original
// preference order. Consumers must not mutate the shared map.
func CharacterReadings() (map[string][]string, error) { return loadReadings() }
