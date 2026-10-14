package application

import (
	"context"

	"arion.local/sync/domain"
)

// PushInput is an authenticated batch. Operations were decoded from Auth.Body.
type PushInput struct {
	Authorization string
	Auth          DeviceProof
	Operations    []domain.Incoming
}

// OpResult is one operation's stored outcome.
type OpResult struct {
	OperationID    string `json:"operation_id"`
	Outcome        string `json:"outcome"`
	ServerRevision int64  `json:"server_revision"`
	Sibling        bool   `json:"sibling"`
	Suppressed     bool   `json:"suppressed"`
	KeyEpoch       uint64 `json:"key_epoch"`
}

// PushOutput returns per-operation results and the group's newest cursor.
type PushOutput struct {
	Results []OpResult `json:"results"`
	Cursor  string     `json:"cursor"`
}

// Push validates, deduplicates, and appends a batch in one transaction.
func (s *Service) Push(ctx context.Context, in PushInput) (PushOutput, error) {
	if len(in.Operations) == 0 {
		return PushOutput{}, domain.Fail(domain.CodeInvalidEnvelope)
	}
	if len(in.Operations) > s.limits.MaxBatch {
		return PushOutput{}, domain.Fail(domain.CodePayloadTooLarge)
	}
	seen := make(map[string]struct{}, len(in.Operations))
	for _, op := range in.Operations {
		if _, ok := seen[op.OperationID]; ok {
			return PushOutput{}, domain.Fail(domain.CodeInvalidEnvelope)
		}
		seen[op.OperationID] = struct{}{}
		if err := domain.ValidateIncoming(op, s.limits); err != nil {
			return PushOutput{}, err
		}
	}
	scope, err := s.authorize(ctx, in.Authorization)
	if err != nil {
		return PushOutput{}, err
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return PushOutput{}, err
	}
	defer tx.Rollback(ctx)
	device, err := s.authenticate(ctx, tx, scope, in.Auth)
	if err != nil {
		return PushOutput{}, err
	}
	group, err := tx.Group(ctx, device.GroupID)
	if err != nil {
		return PushOutput{}, err
	}
	results := make([]OpResult, 0, len(in.Operations))
	for _, op := range in.Operations {
		result, err := s.applyOne(ctx, tx, group, device, op)
		if err != nil {
			return PushOutput{}, err
		}
		if result.Outcome != "duplicate" && scope.Quota > 0 {
			used, err := tx.ScopeBytes(ctx, scope.ID)
			if err != nil {
				return PushOutput{}, err
			}
			if used > scope.Quota {
				return PushOutput{}, Fail(CodeQuotaExceeded)
			}
		}
		results = append(results, result)
	}
	head, err := tx.MaxRevision(ctx, group.ID)
	if err != nil {
		return PushOutput{}, err
	}
	cursor, err := domain.EncodeCursor(head)
	if err != nil {
		return PushOutput{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PushOutput{}, err
	}
	s.log.Info("push", "device_id", device.ID, "group_id", group.ID, "operations", len(results))
	return PushOutput{Results: results, Cursor: cursor}, nil
}

func (s *Service) applyOne(ctx context.Context, tx domain.Tx, group domain.Group, device domain.Device, op domain.Incoming) (OpResult, error) {
	fingerprint := domain.Fingerprint(group.ID, op)
	if receipt, ok, err := tx.Receipt(ctx, device.ID, op.OperationID); err != nil {
		return OpResult{}, err
	} else if ok {
		if string(receipt.Fingerprint) != string(fingerprint) {
			return OpResult{}, domain.Fail(domain.CodeOperationMismatch)
		}
		return OpResult{
			OperationID:    op.OperationID,
			Outcome:        "duplicate",
			ServerRevision: receipt.ServerRevision,
			Sibling:        receipt.Sibling,
			Suppressed:     receipt.Suppressed,
			KeyEpoch:       receipt.KeyEpoch,
		}, nil
	}
	exists, err := tx.FingerprintExists(ctx, fingerprint)
	if err != nil {
		return OpResult{}, err
	}
	if exists {
		return OpResult{}, domain.Fail(domain.CodeReplay)
	}
	existing, err := tx.Envelopes(ctx, group.ID, op.RecordID)
	if err != nil {
		return OpResult{}, err
	}
	decision, err := domain.Decide(existing, op, group, device.ID, s.now())
	if err != nil {
		return OpResult{}, err
	}
	revision, err := tx.NextRevision(ctx)
	if err != nil {
		return OpResult{}, err
	}
	decision.Envelope.ServerRevision = revision
	if err := tx.InsertEnvelope(ctx, decision.Envelope); err != nil {
		return OpResult{}, err
	}
	for _, rev := range decision.MarkSuppressed {
		if err := tx.SetSuppressed(ctx, rev, true); err != nil {
			return OpResult{}, err
		}
	}
	if err := tx.InsertFingerprint(ctx, fingerprint, device.ID, op.OperationID); err != nil {
		return OpResult{}, err
	}
	if err := tx.InsertReceipt(ctx, domain.Receipt{
		DeviceID:       device.ID,
		OperationID:    op.OperationID,
		Fingerprint:    fingerprint,
		ServerRevision: revision,
		Sibling:        decision.Sibling,
		Suppressed:     decision.Suppressed,
		KeyEpoch:       decision.Envelope.KeyEpoch,
	}); err != nil {
		return OpResult{}, err
	}
	if decision.Envelope.Kind == domain.KindTombstone {
		if err := tx.Ack(ctx, revision, device.ID); err != nil {
			return OpResult{}, err
		}
		if err := s.collect(ctx, tx, group.ID, op.RecordID); err != nil {
			return OpResult{}, err
		}
	}
	return OpResult{
		OperationID:    op.OperationID,
		Outcome:        "applied",
		ServerRevision: revision,
		Sibling:        decision.Sibling,
		Suppressed:     decision.Suppressed,
		KeyEpoch:       decision.Envelope.KeyEpoch,
	}, nil
}

func (s *Service) collect(ctx context.Context, tx domain.Tx, groupID, recordID string) error {
	record, err := tx.Envelopes(ctx, groupID, recordID)
	if err != nil {
		return err
	}
	devices, err := tx.Devices(ctx, groupID)
	if err != nil {
		return err
	}
	active := make([]string, 0, len(devices))
	for _, device := range devices {
		if device.Status == domain.DeviceActive {
			active = append(active, device.ID)
		}
	}
	var tombstone *domain.Envelope
	for i := range record {
		if record[i].Kind == domain.KindTombstone {
			tombstone = &record[i]
			break
		}
	}
	if tombstone == nil {
		return nil
	}
	acks, err := tx.Acks(ctx, tombstone.ServerRevision)
	if err != nil {
		return err
	}
	if domain.CollectibleTombstone(record, acks, active) == nil {
		return nil
	}
	for _, env := range record {
		if err := tx.DeleteEnvelope(ctx, env.ServerRevision); err != nil {
			return err
		}
	}
	return nil
}
