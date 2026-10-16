// Command arion-memory-loop exercises production edit extraction and encrypted
// persistence for authored, offline research. It does not read native fields.
package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/vocabulary"
)

type request struct {
	Operation string           `json:"operation"`
	StorePath string           `json:"storePath"`
	KeyPath   string           `json:"keyPath"`
	Edit      domain.FieldEdit `json:"edit"`
	EntryID   string           `json:"entryId"`
	UnixTime  int64            `json:"unixTime"`
}

func execute(r request) (map[string]any, error) {
	if r.StorePath == "" || r.KeyPath == "" {
		return nil, fmt.Errorf("storePath and keyPath required")
	}
	if r.Operation == "init" {
		if _, err := os.Stat(r.StorePath); !os.IsNotExist(err) {
			return nil, fmt.Errorf("new store path required")
		}
		if err := os.MkdirAll(filepath.Dir(r.KeyPath), 0700); err != nil {
			return nil, err
		}
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(r.KeyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, err
		}
		_, writeErr := f.Write(key)
		closeErr := f.Close()
		if writeErr != nil {
			return nil, writeErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	}
	key, err := os.ReadFile(r.KeyPath)
	if err != nil {
		return nil, err
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("32-byte research key required")
	}
	store, err := vocabulary.Open(r.StorePath, key)
	if err != nil {
		return nil, err
	}
	before, err := store.List()
	if err != nil {
		return nil, err
	}
	result := map[string]any{"beforeCount": len(before)}
	start := time.Now()
	switch r.Operation {
	case "init", "snapshot":
	case "settle":
		outcome := domain.ExtractCorrections(r.Edit)
		result["extraction"] = outcome
		if outcome.Verdict == domain.EditLearned {
			learned, err := store.Learn(outcome.Pairs, time.Unix(r.UnixTime, 0))
			if err != nil {
				return nil, err
			}
			result["learning"] = learned
		}
	case "remove":
		if r.EntryID == "" {
			return nil, fmt.Errorf("entryId required")
		}
		if err := store.Remove(r.EntryID); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unknown operation %q", r.Operation)
	}
	result["mutationMs"] = float64(time.Since(start).Nanoseconds()) / 1e6
	// Reopen the actual encrypted file; returning the in-memory learning result
	// alone would not exercise durable persistence and restart decoding.
	reopened, err := vocabulary.Open(r.StorePath, key)
	if err != nil {
		return nil, err
	}
	entries, err := reopened.List()
	if err != nil {
		return nil, err
	}
	result["entries"] = entries
	result["afterCount"] = len(entries)
	if data, err := os.ReadFile(r.StorePath); err == nil {
		h := sha256.Sum256(data)
		result["ciphertextSha256"] = hex.EncodeToString(h[:])
		result["ciphertextBytes"] = len(data)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	return result, nil
}

func main() {
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 1<<20))
	decoder.DisallowUnknownFields()
	var r request
	if err := decoder.Decode(&r); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	result, err := execute(r)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
