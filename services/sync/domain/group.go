package domain

import "time"

// Group is the sync partition a device enrolls in. KeyEpoch is the only epoch
// the relay will accept for new writes. Rotating it does not re-encrypt
// stored ciphertext; clients do that.
type Group struct {
	ID        string
	ScopeID   string
	KeyEpoch  uint64
	CreatedAt time.Time
}

const initialKeyEpoch uint64 = 1

// NewGroup builds a group at epoch 1.
func NewGroup(id, scopeID string, now time.Time) (Group, error) {
	if !ValidID(id) || !ValidID(scopeID) {
		return Group{}, Fail(CodeInvalidEnvelope)
	}
	return Group{ID: id, ScopeID: scopeID, KeyEpoch: initialKeyEpoch, CreatedAt: now}, nil
}
