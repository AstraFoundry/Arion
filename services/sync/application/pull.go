package application

import (
	"context"

	"arion.local/sync/domain"
)

// PullInput requests revisions after an opaque cursor.
type PullInput struct {
	Authorization string
	Auth          DeviceProof
	Cursor        string
}

// SnapshotInput requests the validated full log for the device's group.
type SnapshotInput struct {
	Authorization string
	Auth          DeviceProof
}

// LogOutput is a page of ciphertext envelopes.
type LogOutput struct {
	Envelopes []domain.Envelope `json:"-"`
	Cursor    string            `json:"cursor"`
	Snapshot  bool              `json:"snapshot,omitempty"`
	More      bool              `json:"more,omitempty"`
}

// Pull returns envelopes strictly after the cursor. An expired or unknown
// cursor is CURSOR_EXPIRED so the client takes the snapshot path.
func (s *Service) Pull(ctx context.Context, in PullInput) (LogOutput, error) {
	if in.Cursor == "" {
		return LogOutput{}, domain.Fail(domain.CodeInvalidEnvelope)
	}
	after, err := domain.DecodeCursor(in.Cursor)
	if err != nil {
		return LogOutput{}, err
	}
	return s.read(ctx, in.Authorization, in.Auth, after, false)
}

// Snapshot returns the group log from the beginning. It is the resync path
// after cursor expiry and does not delete local client state.
func (s *Service) Snapshot(ctx context.Context, in SnapshotInput) (LogOutput, error) {
	return s.read(ctx, in.Authorization, in.Auth, 0, true)
}

func (s *Service) read(ctx context.Context, authorization string, auth DeviceProof, after int64, snapshot bool) (LogOutput, error) {
	scope, err := s.authorize(ctx, authorization)
	if err != nil {
		return LogOutput{}, err
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return LogOutput{}, err
	}
	defer tx.Rollback(ctx)
	device, err := s.authenticate(ctx, tx, scope, auth)
	if err != nil {
		return LogOutput{}, err
	}
	if !snapshot && after != 0 {
		created, ok, err := tx.CreatedAt(ctx, device.GroupID, after)
		if err != nil {
			return LogOutput{}, err
		}
		if !ok || domain.CursorExpired(created, s.now(), s.cursorTTL) {
			return LogOutput{}, domain.Fail(domain.CodeCursorExpired)
		}
	}
	page, err := tx.ListAfter(ctx, device.GroupID, after, s.pageSize)
	if err != nil {
		return LogOutput{}, err
	}
	more := len(page) == s.pageSize
	head := after
	if len(page) > 0 {
		head = page[len(page)-1].ServerRevision
	} else if snapshot {
		head, err = tx.MaxRevision(ctx, device.GroupID)
		if err != nil {
			return LogOutput{}, err
		}
	}
	cursor, err := domain.EncodeCursor(head)
	if err != nil {
		return LogOutput{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LogOutput{}, err
	}
	return LogOutput{Envelopes: page, Cursor: cursor, Snapshot: snapshot, More: more}, nil
}

// AckInput acknowledges tombstones the device has stored.
type AckInput struct {
	Authorization string
	Auth          DeviceProof
	Revisions     []int64
}

// AckOutput lists the revisions recorded.
type AckOutput struct {
	Acknowledged []int64 `json:"acknowledged"`
}

// Ack records tombstone acknowledgements and collects a tombstone once every
// active device has acknowledged it.
func (s *Service) Ack(ctx context.Context, in AckInput) (AckOutput, error) {
	if len(in.Revisions) == 0 || len(in.Revisions) > s.limits.MaxBatch {
		return AckOutput{}, domain.Fail(domain.CodePayloadTooLarge)
	}
	scope, err := s.authorize(ctx, in.Authorization)
	if err != nil {
		return AckOutput{}, err
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return AckOutput{}, err
	}
	defer tx.Rollback(ctx)
	device, err := s.authenticate(ctx, tx, scope, in.Auth)
	if err != nil {
		return AckOutput{}, err
	}
	for _, rev := range in.Revisions {
		env, err := tx.Envelope(ctx, device.GroupID, rev)
		if err != nil {
			return AckOutput{}, err
		}
		if env.Kind != domain.KindTombstone {
			return AckOutput{}, domain.Fail(domain.CodeInvalidEnvelope)
		}
		if err := tx.Ack(ctx, rev, device.ID); err != nil {
			return AckOutput{}, err
		}
		if err := s.collect(ctx, tx, device.GroupID, env.RecordID); err != nil {
			return AckOutput{}, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return AckOutput{}, err
	}
	return AckOutput{Acknowledged: append([]int64(nil), in.Revisions...)}, nil
}
