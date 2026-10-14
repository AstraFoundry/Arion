package postgres

import (
	"crypto/rand"
	"encoding/hex"
)

func randomID(buf *[16]byte) error {
	_, err := rand.Read(buf[:])
	return err
}

func encodeID(raw []byte) string { return hex.EncodeToString(raw) }
