package application

import (
	"context"

	"arion.local/sync/domain"
)

// RegisterInput enrolls a device. Recovery is rejected: the recovery format
// is a gated decision and this relay does not invent one.
type RegisterInput struct {
	Authorization    string
	Auth             DeviceProof
	DeviceID         string
	GroupID          string
	PublicKey        []byte
	PublicKeyB64     string
	SponsorDeviceID  string
	SponsorSignature []byte
	Recovery         string
}

// RegisterOutput is the enrolled identity.
type RegisterOutput struct {
	DeviceID string `json:"device_id"`
	GroupID  string `json:"group_id"`
	KeyEpoch uint64 `json:"key_epoch"`
}

// Register creates a group for the first device or admits a device sponsored
// by an active member. The account gate has already chosen the scope.
func (s *Service) Register(ctx context.Context, in RegisterInput) (RegisterOutput, error) {
	if in.Recovery != "" {
		return RegisterOutput{}, domain.Fail(domain.CodeRecoveryUnsupported)
	}
	scope, err := s.authorize(ctx, in.Authorization)
	if err != nil {
		return RegisterOutput{}, err
	}
	if !domain.ValidID(in.DeviceID) || !domain.ValidID(in.GroupID) || in.Auth.DeviceID != in.DeviceID {
		return RegisterOutput{}, domain.Fail(domain.CodeInvalidEnvelope)
	}
	if !s.withinSkew(in.Auth.Timestamp) {
		return RegisterOutput{}, domain.Fail(domain.CodeUnauthorized)
	}
	message := domain.CanonicalRequest(in.Auth.DeviceID, in.Auth.Timestamp.UnixMilli(), in.Auth.Method, in.Auth.Path, in.Auth.Body)
	if err := domain.VerifySignature(in.PublicKey, message, in.Auth.Signature); err != nil {
		return RegisterOutput{}, err
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return RegisterOutput{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Device(ctx, in.DeviceID); err == nil {
		return RegisterOutput{}, domain.Fail(domain.CodeConflict)
	} else if !domain.IsCode(err, domain.CodeNotFound) {
		return RegisterOutput{}, err
	}
	group, err := tx.Group(ctx, in.GroupID)
	if domain.IsCode(err, domain.CodeNotFound) {
		if in.SponsorDeviceID != "" {
			return RegisterOutput{}, domain.Fail(domain.CodeUnauthorized)
		}
		group, err = domain.NewGroup(in.GroupID, scope.ID, s.now())
		if err != nil {
			return RegisterOutput{}, err
		}
		if err := tx.InsertGroup(ctx, group); err != nil {
			return RegisterOutput{}, err
		}
	} else if err != nil {
		return RegisterOutput{}, err
	} else {
		if group.ScopeID != scope.ID {
			return RegisterOutput{}, domain.Fail(domain.CodeScopeMismatch)
		}
		if err := s.verifySponsor(ctx, tx, group, in); err != nil {
			return RegisterOutput{}, err
		}
	}
	device := domain.Device{
		ID:        in.DeviceID,
		GroupID:   group.ID,
		ScopeID:   scope.ID,
		PublicKey: append([]byte(nil), in.PublicKey...),
		Status:    domain.DeviceActive,
		CreatedAt: s.now(),
	}
	if err := tx.InsertDevice(ctx, device); err != nil {
		return RegisterOutput{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RegisterOutput{}, err
	}
	s.log.Info("device enrolled", "device_id", device.ID, "group_id", group.ID, "key_epoch", group.KeyEpoch)
	return RegisterOutput{DeviceID: device.ID, GroupID: group.ID, KeyEpoch: group.KeyEpoch}, nil
}

func (s *Service) verifySponsor(ctx context.Context, tx domain.Tx, group domain.Group, in RegisterInput) error {
	if !domain.ValidID(in.SponsorDeviceID) || len(in.SponsorSignature) == 0 {
		return domain.Fail(domain.CodeUnauthorized)
	}
	sponsor, err := tx.Device(ctx, in.SponsorDeviceID)
	if err != nil {
		if domain.IsCode(err, domain.CodeNotFound) {
			return domain.Fail(domain.CodeUnauthorized)
		}
		return err
	}
	if sponsor.GroupID != group.ID || sponsor.ScopeID != group.ScopeID || sponsor.Status != domain.DeviceActive {
		return domain.Fail(domain.CodeUnauthorized)
	}
	message := domain.EnrollMessage(group.ID, in.DeviceID, in.PublicKeyB64)
	return domain.VerifySignature(sponsor.PublicKey, message, in.SponsorSignature)
}

// RevokeInput identifies the actor by signature and the target by id.
type RevokeInput struct {
	Authorization string
	Auth          DeviceProof
	TargetID      string
}

// RevokeOutput reports the epoch future writes must use.
type RevokeOutput struct {
	DeviceID string `json:"device_id"`
	KeyEpoch uint64 `json:"key_epoch"`
	Status   string `json:"status"`
}

// Revoke marks a device revoked and rotates the group key epoch once.
// Repeating the revoke does not rotate the epoch again.
func (s *Service) Revoke(ctx context.Context, in RevokeInput) (RevokeOutput, error) {
	scope, err := s.authorize(ctx, in.Authorization)
	if err != nil {
		return RevokeOutput{}, err
	}
	tx, err := s.begin(ctx)
	if err != nil {
		return RevokeOutput{}, err
	}
	defer tx.Rollback(ctx)
	actor, err := s.authenticate(ctx, tx, scope, in.Auth)
	if err != nil {
		return RevokeOutput{}, err
	}
	target, err := tx.Device(ctx, in.TargetID)
	if err != nil {
		return RevokeOutput{}, err
	}
	if target.GroupID != actor.GroupID || target.ScopeID != scope.ID {
		return RevokeOutput{}, domain.Fail(domain.CodeScopeMismatch)
	}
	group, err := tx.Group(ctx, target.GroupID)
	if err != nil {
		return RevokeOutput{}, err
	}
	if target.Status == domain.DeviceRevoked {
		if err := tx.Commit(ctx); err != nil {
			return RevokeOutput{}, err
		}
		return RevokeOutput{DeviceID: target.ID, KeyEpoch: group.KeyEpoch, Status: string(domain.DeviceRevoked)}, nil
	}
	now := s.now()
	if err := tx.RevokeDevice(ctx, target.ID, now); err != nil {
		return RevokeOutput{}, err
	}
	next := group.KeyEpoch + 1
	if err := tx.SetEpoch(ctx, group.ID, next); err != nil {
		return RevokeOutput{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return RevokeOutput{}, err
	}
	s.log.Info("device revoked", "device_id", target.ID, "group_id", group.ID, "key_epoch", next)
	return RevokeOutput{DeviceID: target.ID, KeyEpoch: next, Status: string(domain.DeviceRevoked)}, nil
}
