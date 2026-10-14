package domain

import (
	"slices"
	"time"
)

// Decision is the revision-policy result for one operation. ServerRevision is
// assigned by the store after Decide returns.
type Decision struct {
	Envelope       Envelope
	Sibling        bool
	Suppressed     bool
	MarkSuppressed []int64
}

// Decide applies SYNC-05. Concurrent writers that share a base both become
// heads. A delete beside an edit suppresses the record until a resolution
// names every current head. Wall-clock order is not an input.
func Decide(existing []Envelope, in Incoming, group Group, deviceID string, now time.Time) (Decision, error) {
	if in.KeyEpoch != group.KeyEpoch {
		return Decision{}, Fail(CodeStaleKeyEpoch)
	}
	current := heads(existing)
	parents, err := parentsFor(existing, current, in)
	if err != nil {
		return Decision{}, err
	}
	env := Envelope{
		GroupID:          group.ID,
		ScopeID:          group.ScopeID,
		RecordID:         in.RecordID,
		DeviceID:         deviceID,
		OperationID:      in.OperationID,
		SchemaVersion:    in.SchemaVersion,
		KeyEpoch:         in.KeyEpoch,
		Algorithm:        in.Algorithm,
		Nonce:            append([]byte(nil), in.Nonce...),
		Ciphertext:       append([]byte(nil), in.Ciphertext...),
		Tag:              append([]byte(nil), in.Tag...),
		AAD:              append([]byte(nil), in.AAD...),
		LogicalRevision:  in.LogicalRevision,
		Kind:             in.Kind,
		ExpectedRevision: in.ExpectedRevision,
		Parents:          parents,
		CreatedAt:        now,
	}
	next := futureHeads(current, env)
	suppressed := mixedDeleteEdit(next)
	env.Suppressed = suppressed
	mark := []int64{}
	if suppressed {
		for _, head := range next {
			if head.ServerRevision != 0 {
				mark = append(mark, head.ServerRevision)
			}
		}
		slices.Sort(mark)
	}
	return Decision{
		Envelope:       env,
		Sibling:        len(next) > 1,
		Suppressed:     suppressed,
		MarkSuppressed: mark,
	}, nil
}

func parentsFor(existing, current []Envelope, in Incoming) ([]int64, error) {
	if in.Kind == KindResolution {
		if len(current) == 0 || !sameIDs(in.Parents, headIDs(current)) {
			return nil, Fail(CodeConflict)
		}
		parents := append([]int64(nil), in.Parents...)
		slices.Sort(parents)
		return parents, nil
	}
	if len(existing) == 0 {
		if in.ExpectedRevision != 0 || len(in.Parents) != 0 {
			return nil, Fail(CodeConflict)
		}
		return nil, nil
	}
	if containsRevision(current, in.ExpectedRevision) {
		return []int64{in.ExpectedRevision}, nil
	}
	if directParentOfHead(current, in.ExpectedRevision) {
		if in.ExpectedRevision == 0 {
			return nil, nil
		}
		return []int64{in.ExpectedRevision}, nil
	}
	return nil, Fail(CodeConflict)
}

func heads(existing []Envelope) []Envelope {
	referenced := map[int64]bool{}
	for _, env := range existing {
		for _, parent := range env.Parents {
			referenced[parent] = true
		}
	}
	out := make([]Envelope, 0, 1)
	for _, env := range existing {
		if !referenced[env.ServerRevision] {
			out = append(out, env)
		}
	}
	return out
}

func futureHeads(current []Envelope, env Envelope) []Envelope {
	drop := map[int64]bool{}
	for _, parent := range env.Parents {
		drop[parent] = true
	}
	out := make([]Envelope, 0, len(current)+1)
	for _, head := range current {
		if !drop[head.ServerRevision] {
			out = append(out, head)
		}
	}
	out = append(out, env)
	return out
}

func mixedDeleteEdit(hs []Envelope) bool {
	tomb, live := false, false
	for _, head := range hs {
		if head.Kind == KindTombstone {
			tomb = true
		} else {
			live = true
		}
	}
	return tomb && live
}

func containsRevision(hs []Envelope, rev int64) bool {
	for _, head := range hs {
		if head.ServerRevision == rev {
			return true
		}
	}
	return false
}

func directParentOfHead(hs []Envelope, expected int64) bool {
	for _, head := range hs {
		if expected == 0 {
			if len(head.Parents) == 0 {
				return true
			}
			continue
		}
		if slices.Contains(head.Parents, expected) {
			return true
		}
	}
	return false
}

func headIDs(hs []Envelope) []int64 {
	ids := make([]int64, len(hs))
	for i, head := range hs {
		ids[i] = head.ServerRevision
	}
	return ids
}

func sameIDs(a, b []int64) bool {
	if len(a) != len(b) || len(a) == 0 {
		return false
	}
	left := append([]int64(nil), a...)
	right := append([]int64(nil), b...)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

// CollectibleTombstone returns the sole tombstone head once every active
// device has acknowledged it. A delete/edit conflict is not collectible.
// Revoked devices are omitted from active by the caller.
func CollectibleTombstone(record []Envelope, acks []string, active []string) []int64 {
	current := heads(record)
	if len(current) != 1 || current[0].Kind != KindTombstone || current[0].ServerRevision == 0 {
		return nil
	}
	if len(active) == 0 {
		return nil
	}
	have := make(map[string]struct{}, len(acks))
	for _, id := range acks {
		have[id] = struct{}{}
	}
	for _, id := range active {
		if _, ok := have[id]; !ok {
			return nil
		}
	}
	return []int64{current[0].ServerRevision}
}
