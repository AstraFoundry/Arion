package domain

import (
	"testing"
	"time"
)

func groupFixture() Group {
	return Group{ID: "grp", ScopeID: "local", KeyEpoch: 1}
}

func incoming(kind Kind, record string, expected int64, parents []int64) Incoming {
	return Incoming{
		OperationID:      "op-" + record + "-" + string(kind),
		RecordID:         record,
		SchemaVersion:    1,
		KeyEpoch:         1,
		Algorithm:        "aes-256-gcm",
		Nonce:            bytesN(12),
		Ciphertext:       []byte("cipher"),
		Tag:              bytesN(16),
		LogicalRevision:  1,
		Kind:             kind,
		ExpectedRevision: expected,
		Parents:          parents,
	}
}

func bytesN(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(i + 1)
	}
	return out
}

func apply(t *testing.T, existing []Envelope, in Incoming) (Decision, []Envelope) {
	t.Helper()
	in.OperationID = "op-" + itoa(len(existing)+1)
	decision, err := Decide(existing, in, groupFixture(), "dev-a", time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	decision.Envelope.ServerRevision = int64(len(existing) + 1)
	return decision, append(append([]Envelope{}, existing...), decision.Envelope)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

func TestDecideLinearThenSibling(t *testing.T) {
	first, log := apply(t, nil, incoming(KindWrite, "rec", 0, nil))
	if first.Sibling || first.Envelope.ServerRevision != 1 {
		t.Fatalf("first = %+v", first)
	}
	second, log := apply(t, log, incoming(KindWrite, "rec", 1, nil))
	if second.Sibling {
		t.Fatal("linear extension stored a sibling")
	}
	concurrent := incoming(KindWrite, "rec", 1, nil)
	concurrent.OperationID = "op-concurrent"
	concurrent.Ciphertext = []byte("other")
	decision, err := Decide(log, concurrent, groupFixture(), "dev-b", time.Unix(101, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Sibling || decision.Suppressed {
		t.Fatalf("concurrent edit = sibling %v suppressed %v", decision.Sibling, decision.Suppressed)
	}
	if second.Envelope.CreatedAt.After(decision.Envelope.CreatedAt) {
		t.Fatal("test clock went backwards")
	}
}

func TestDecideStaleBaseConflicts(t *testing.T) {
	_, log := apply(t, nil, incoming(KindWrite, "rec", 0, nil))
	_, log = apply(t, log, incoming(KindWrite, "rec", 1, nil))
	stale := incoming(KindWrite, "rec", 0, nil)
	stale.OperationID = "op-stale"
	_, err := Decide(log, stale, groupFixture(), "dev-b", time.Unix(102, 0).UTC())
	if !IsCode(err, CodeConflict) {
		t.Fatalf("err = %v", err)
	}
}

func TestDecideDeleteEditSuppressesUntilResolution(t *testing.T) {
	_, log := apply(t, nil, incoming(KindWrite, "rec", 0, nil))
	edit := incoming(KindWrite, "rec", 1, nil)
	edit.OperationID = "op-edit"
	editDecision, err := Decide(log, edit, groupFixture(), "dev-a", time.Unix(100, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	editDecision.Envelope.ServerRevision = 2
	log = append(log, editDecision.Envelope)
	tomb := incoming(KindTombstone, "rec", 1, nil)
	tomb.OperationID = "op-tomb"
	tomb.Ciphertext = []byte("tomb")
	tombDecision, err := Decide(log, tomb, groupFixture(), "dev-b", time.Unix(101, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !tombDecision.Suppressed || !tombDecision.Sibling {
		t.Fatalf("delete/edit = %+v", tombDecision)
	}
	tombDecision.Envelope.ServerRevision = 3
	log = append(log, tombDecision.Envelope)
	for _, rev := range tombDecision.MarkSuppressed {
		for i := range log {
			if log[i].ServerRevision == rev {
				log[i].Suppressed = true
			}
		}
	}
	resolution := incoming(KindResolution, "rec", 2, []int64{3, 2})
	resolution.OperationID = "op-resolve"
	resolved, err := Decide(log, resolution, groupFixture(), "dev-a", time.Unix(102, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Sibling || resolved.Suppressed {
		t.Fatalf("resolution = sibling %v suppressed %v", resolved.Sibling, resolved.Suppressed)
	}
	partial := incoming(KindResolution, "rec", 2, []int64{2})
	partial.OperationID = "op-partial"
	if _, err := Decide(log, partial, groupFixture(), "dev-a", time.Unix(103, 0).UTC()); !IsCode(err, CodeConflict) {
		t.Fatalf("partial resolution err = %v", err)
	}
}

func TestDecideStaleEpoch(t *testing.T) {
	in := incoming(KindWrite, "rec", 0, nil)
	in.KeyEpoch = 1
	group := groupFixture()
	group.KeyEpoch = 2
	_, err := Decide(nil, in, group, "dev-a", time.Unix(1, 0).UTC())
	if !IsCode(err, CodeStaleKeyEpoch) {
		t.Fatalf("err = %v", err)
	}
}

func TestCollectibleTombstoneWaitsForActiveDevices(t *testing.T) {
	_, log := apply(t, nil, incoming(KindTombstone, "rec", 0, nil))
	if got := CollectibleTombstone(log, []string{"dev-a"}, []string{"dev-a", "dev-b"}); got != nil {
		t.Fatalf("collected early: %v", got)
	}
	got := CollectibleTombstone(log, []string{"dev-a", "dev-b"}, []string{"dev-a", "dev-b"})
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("got %v", got)
	}
	if got := CollectibleTombstone(log, nil, nil); got != nil {
		t.Fatal("empty active set collected")
	}
}

func TestValidateIncomingOutcomes(t *testing.T) {
	lim := DefaultLimits()
	ok := incoming(KindWrite, "rec", 0, nil)
	if err := ValidateIncoming(ok, lim); err != nil {
		t.Fatal(err)
	}
	unknown := ok
	unknown.SchemaVersion = 9
	if !IsCode(ValidateIncoming(unknown, lim), CodeUnknownSchema) {
		t.Fatal("schema")
	}
	badTag := ok
	badTag.Tag = []byte{1}
	if !IsCode(ValidateIncoming(badTag, lim), CodeInvalidTag) {
		t.Fatal("tag")
	}
	empty := ok
	empty.Ciphertext = nil
	if !IsCode(ValidateIncoming(empty, lim), CodeInvalidEnvelope) {
		t.Fatal("ciphertext")
	}
}

func TestFingerprintIgnoresOperationID(t *testing.T) {
	a := incoming(KindWrite, "rec", 0, nil)
	b := a
	b.OperationID = "other"
	if string(Fingerprint("grp", a)) != string(Fingerprint("grp", b)) {
		t.Fatal("operation id changed the fingerprint")
	}
	b.Ciphertext = []byte("different")
	if string(Fingerprint("grp", a)) == string(Fingerprint("grp", b)) {
		t.Fatal("ciphertext did not change the fingerprint")
	}
}
