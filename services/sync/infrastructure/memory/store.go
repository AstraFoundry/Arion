package memory

import (
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"slices"
	"sync"
	"time"

	"arion.local/sync/application"
	"arion.local/sync/domain"
)

// Store is an in-memory repository for tests. It implements the same ports as
// PostgreSQL and is not wired into the server process.
type Store struct {
	mu    sync.Mutex
	state *state
}

type state struct {
	groups       map[string]domain.Group
	devices      map[string]domain.Device
	envelopes    map[int64]domain.Envelope
	receipts     map[string]domain.Receipt
	fingerprints map[string]struct{}
	acks         map[int64]map[string]struct{}
	next         int64
	links        map[string]string
	tenants      map[string]application.Tenant
	controls     map[string]control
	jtis         map[string]time.Time
}

type control struct {
	frozen  bool
	purged  bool
	session *time.Time
}

// Counts is test-visible storage size.
type Counts struct {
	Tenants   int
	Links     int
	Events    int
	Envelopes int
	Groups    int
	Devices   int
}

// New returns an empty store.
func New() *Store {
	return &Store{state: newState()}
}

func newState() *state {
	return &state{
		groups:       map[string]domain.Group{},
		devices:      map[string]domain.Device{},
		envelopes:    map[int64]domain.Envelope{},
		receipts:     map[string]domain.Receipt{},
		fingerprints: map[string]struct{}{},
		acks:         map[int64]map[string]struct{}{},
		next:         1,
		links:        map[string]string{},
		tenants:      map[string]application.Tenant{},
		controls:     map[string]control{},
		jtis:         map[string]time.Time{},
	}
}

// Counts reports row counts. It must not be called while a transaction from
// the same goroutine is open.
func (s *Store) Counts() Counts {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Counts{
		Tenants:   len(s.state.tenants),
		Links:     len(s.state.links),
		Events:    len(s.state.jtis),
		Envelopes: len(s.state.envelopes),
		Groups:    len(s.state.groups),
		Devices:   len(s.state.devices),
	}
}

func (s *Store) Begin(context.Context) (domain.Tx, error) { return s.begin() }

func (s *Store) BeginAccount(context.Context) (application.AccountTx, error) { return s.begin() }

func (s *Store) begin() (*memTx, error) {
	s.mu.Lock()
	return &memTx{db: s, state: cloneState(s.state)}, nil
}

func (s *Store) Lookup(_ context.Context, issuer, subject string) (application.Access, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return lookup(s.state, issuer, subject), nil
}

func (s *Store) EnsureTenant(_ context.Context, issuer, subject string, quota int64, now time.Time) (application.Tenant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ensureTenant(s.state, issuer, subject, quota, now)
}

type memTx struct {
	db    *Store
	state *state
	done  bool
}

func (t *memTx) Commit(context.Context) error {
	if t.done {
		return nil
	}
	t.db.state = t.state
	t.done = true
	t.db.mu.Unlock()
	return nil
}

func (t *memTx) Rollback(context.Context) error {
	if t.done {
		return nil
	}
	t.done = true
	t.db.mu.Unlock()
	return nil
}

func (t *memTx) Group(_ context.Context, id string) (domain.Group, error) {
	group, ok := t.state.groups[id]
	if !ok {
		return domain.Group{}, domain.Fail(domain.CodeNotFound)
	}
	return group, nil
}

func (t *memTx) InsertGroup(_ context.Context, group domain.Group) error {
	if _, ok := t.state.groups[group.ID]; ok {
		return domain.Fail(domain.CodeConflict)
	}
	t.state.groups[group.ID] = group
	return nil
}

func (t *memTx) SetEpoch(_ context.Context, groupID string, epoch uint64) error {
	group, ok := t.state.groups[groupID]
	if !ok {
		return domain.Fail(domain.CodeNotFound)
	}
	group.KeyEpoch = epoch
	t.state.groups[groupID] = group
	return nil
}

func (t *memTx) Device(_ context.Context, id string) (domain.Device, error) {
	device, ok := t.state.devices[id]
	if !ok {
		return domain.Device{}, domain.Fail(domain.CodeNotFound)
	}
	device.PublicKey = append([]byte(nil), device.PublicKey...)
	return device, nil
}

func (t *memTx) InsertDevice(_ context.Context, device domain.Device) error {
	if _, ok := t.state.devices[device.ID]; ok {
		return domain.Fail(domain.CodeConflict)
	}
	device.PublicKey = append([]byte(nil), device.PublicKey...)
	t.state.devices[device.ID] = device
	return nil
}

func (t *memTx) Devices(_ context.Context, groupID string) ([]domain.Device, error) {
	out := []domain.Device{}
	for _, device := range t.state.devices {
		if device.GroupID == groupID {
			device.PublicKey = append([]byte(nil), device.PublicKey...)
			out = append(out, device)
		}
	}
	slices.SortFunc(out, func(a, b domain.Device) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return out, nil
}

func (t *memTx) RevokeDevice(_ context.Context, deviceID string, at time.Time) error {
	device, ok := t.state.devices[deviceID]
	if !ok {
		return domain.Fail(domain.CodeNotFound)
	}
	device.Status = domain.DeviceRevoked
	stamp := at
	device.RevokedAt = &stamp
	t.state.devices[deviceID] = device
	return nil
}

func (t *memTx) Envelopes(_ context.Context, groupID, recordID string) ([]domain.Envelope, error) {
	out := []domain.Envelope{}
	for _, env := range t.state.envelopes {
		if env.GroupID == groupID && env.RecordID == recordID {
			out = append(out, copyEnvelope(env))
		}
	}
	slices.SortFunc(out, func(a, b domain.Envelope) int { return cmp.Compare(a.ServerRevision, b.ServerRevision) })
	return out, nil
}

func (t *memTx) Envelope(_ context.Context, groupID string, revision int64) (domain.Envelope, error) {
	env, ok := t.state.envelopes[revision]
	if !ok || env.GroupID != groupID {
		return domain.Envelope{}, domain.Fail(domain.CodeNotFound)
	}
	return copyEnvelope(env), nil
}

func (t *memTx) InsertEnvelope(_ context.Context, env domain.Envelope) error {
	if _, ok := t.state.envelopes[env.ServerRevision]; ok {
		return domain.Fail(domain.CodeConflict)
	}
	t.state.envelopes[env.ServerRevision] = copyEnvelope(env)
	return nil
}

func (t *memTx) SetSuppressed(_ context.Context, revision int64, suppressed bool) error {
	env, ok := t.state.envelopes[revision]
	if !ok {
		return domain.Fail(domain.CodeNotFound)
	}
	env.Suppressed = suppressed
	t.state.envelopes[revision] = env
	return nil
}

func (t *memTx) DeleteEnvelope(_ context.Context, revision int64) error {
	delete(t.state.envelopes, revision)
	return nil
}

func (t *memTx) ListAfter(_ context.Context, groupID string, after int64, limit int) ([]domain.Envelope, error) {
	out := []domain.Envelope{}
	for _, env := range t.state.envelopes {
		if env.GroupID == groupID && env.ServerRevision > after {
			out = append(out, copyEnvelope(env))
		}
	}
	slices.SortFunc(out, func(a, b domain.Envelope) int { return cmp.Compare(a.ServerRevision, b.ServerRevision) })
	if limit >= 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (t *memTx) CreatedAt(_ context.Context, groupID string, revision int64) (time.Time, bool, error) {
	env, ok := t.state.envelopes[revision]
	if !ok || env.GroupID != groupID {
		return time.Time{}, false, nil
	}
	return env.CreatedAt, true, nil
}

func (t *memTx) MaxRevision(_ context.Context, groupID string) (int64, error) {
	var max int64
	for _, env := range t.state.envelopes {
		if env.GroupID == groupID && env.ServerRevision > max {
			max = env.ServerRevision
		}
	}
	return max, nil
}

func (t *memTx) NextRevision(context.Context) (int64, error) {
	rev := t.state.next
	t.state.next++
	return rev, nil
}

func (t *memTx) Receipt(_ context.Context, deviceID, operationID string) (domain.Receipt, bool, error) {
	receipt, ok := t.state.receipts[deviceID+"\n"+operationID]
	if !ok {
		return domain.Receipt{}, false, nil
	}
	receipt.Fingerprint = append([]byte(nil), receipt.Fingerprint...)
	return receipt, true, nil
}

func (t *memTx) InsertReceipt(_ context.Context, receipt domain.Receipt) error {
	receipt.Fingerprint = append([]byte(nil), receipt.Fingerprint...)
	t.state.receipts[receipt.DeviceID+"\n"+receipt.OperationID] = receipt
	return nil
}

func (t *memTx) FingerprintExists(_ context.Context, fingerprint []byte) (bool, error) {
	_, ok := t.state.fingerprints[string(fingerprint)]
	return ok, nil
}

func (t *memTx) InsertFingerprint(_ context.Context, fingerprint []byte, _, _ string) error {
	t.state.fingerprints[string(fingerprint)] = struct{}{}
	return nil
}

func (t *memTx) Ack(_ context.Context, revision int64, deviceID string) error {
	set := t.state.acks[revision]
	if set == nil {
		set = map[string]struct{}{}
		t.state.acks[revision] = set
	}
	set[deviceID] = struct{}{}
	return nil
}

func (t *memTx) Acks(_ context.Context, revision int64) ([]string, error) {
	out := make([]string, 0, len(t.state.acks[revision]))
	for id := range t.state.acks[revision] {
		out = append(out, id)
	}
	slices.Sort(out)
	return out, nil
}

func (t *memTx) ScopeBytes(_ context.Context, scopeID string) (int64, error) {
	var total int64
	for _, env := range t.state.envelopes {
		if env.ScopeID == scopeID {
			total += int64(len(env.Ciphertext))
		}
	}
	return total, nil
}

func (t *memTx) ClaimJTI(_ context.Context, jti string, now time.Time) (bool, error) {
	if _, ok := t.state.jtis[jti]; ok {
		return false, nil
	}
	t.state.jtis[jti] = now
	return true, nil
}

func (t *memTx) SetFrozen(_ context.Context, issuer, subject string, frozen bool, _ time.Time) error {
	key := accountKey(issuer, subject)
	ctrl := t.state.controls[key]
	if !ctrl.purged {
		ctrl.frozen = frozen
	}
	t.state.controls[key] = ctrl
	if id, ok := t.state.links[key]; ok {
		tenant := t.state.tenants[id]
		if frozen {
			tenant.State = application.TenantFrozen
		} else if !ctrl.purged {
			tenant.State = application.TenantActive
		}
		t.state.tenants[id] = tenant
	}
	return nil
}

func (t *memTx) RevokeSessions(_ context.Context, issuer, subject string, at time.Time) error {
	key := accountKey(issuer, subject)
	ctrl := t.state.controls[key]
	if ctrl.session == nil || at.After(*ctrl.session) {
		stamp := at
		ctrl.session = &stamp
	}
	t.state.controls[key] = ctrl
	return nil
}

func (t *memTx) Purge(_ context.Context, issuer, subject string) error {
	key := accountKey(issuer, subject)
	if id, ok := t.state.links[key]; ok {
		purgeScope(t.state, id)
		delete(t.state.links, key)
		delete(t.state.tenants, id)
	}
	ctrl := t.state.controls[key]
	ctrl.purged = true
	ctrl.frozen = true
	t.state.controls[key] = ctrl
	return nil
}

func purgeScope(st *state, scopeID string) {
	deviceIDs := map[string]struct{}{}
	for id, device := range st.devices {
		if device.ScopeID == scopeID {
			deviceIDs[id] = struct{}{}
			delete(st.devices, id)
		}
	}
	for id, group := range st.groups {
		if group.ScopeID == scopeID {
			delete(st.groups, id)
		}
	}
	for rev, env := range st.envelopes {
		if env.ScopeID == scopeID {
			delete(st.envelopes, rev)
			delete(st.acks, rev)
		}
	}
	for key, receipt := range st.receipts {
		if _, ok := deviceIDs[receipt.DeviceID]; ok {
			delete(st.receipts, key)
			delete(st.fingerprints, string(receipt.Fingerprint))
		}
	}
}

func lookup(st *state, issuer, subject string) application.Access {
	key := accountKey(issuer, subject)
	ctrl := st.controls[key]
	access := application.Access{Frozen: ctrl.frozen, Purged: ctrl.purged}
	if ctrl.session != nil {
		access.HasSessionRevocation = true
		access.SessionRevokedAt = *ctrl.session
	}
	if id, ok := st.links[key]; ok {
		access.Linked = true
		access.Tenant = st.tenants[id]
		if access.Tenant.State == application.TenantFrozen {
			access.Frozen = true
		}
	}
	return access
}

func ensureTenant(st *state, issuer, subject string, quota int64, now time.Time) (application.Tenant, error) {
	access := lookup(st, issuer, subject)
	if access.Purged {
		return application.Tenant{}, domain.Fail(domain.CodeUnauthorized)
	}
	if access.Frozen {
		return application.Tenant{}, application.Fail(application.CodeTenantFrozen)
	}
	if access.Linked {
		return access.Tenant, nil
	}
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return application.Tenant{}, err
	}
	tenant := application.Tenant{ID: hex.EncodeToString(buf[:]), Quota: quota, State: application.TenantActive}
	st.tenants[tenant.ID] = tenant
	st.links[accountKey(issuer, subject)] = tenant.ID
	return tenant, nil
}

func accountKey(issuer, subject string) string { return issuer + "\n" + subject }

func cloneState(in *state) *state {
	out := newState()
	for id, group := range in.groups {
		out.groups[id] = group
	}
	for id, device := range in.devices {
		device.PublicKey = append([]byte(nil), device.PublicKey...)
		if device.RevokedAt != nil {
			stamp := *device.RevokedAt
			device.RevokedAt = &stamp
		}
		out.devices[id] = device
	}
	for rev, env := range in.envelopes {
		out.envelopes[rev] = copyEnvelope(env)
	}
	for key, receipt := range in.receipts {
		receipt.Fingerprint = append([]byte(nil), receipt.Fingerprint...)
		out.receipts[key] = receipt
	}
	for key := range in.fingerprints {
		out.fingerprints[key] = struct{}{}
	}
	for rev, set := range in.acks {
		out.acks[rev] = mapsClone(set)
	}
	out.next = in.next
	for key, id := range in.links {
		out.links[key] = id
	}
	for id, tenant := range in.tenants {
		out.tenants[id] = tenant
	}
	for key, ctrl := range in.controls {
		if ctrl.session != nil {
			stamp := *ctrl.session
			ctrl.session = &stamp
		}
		out.controls[key] = ctrl
	}
	for jti, at := range in.jtis {
		out.jtis[jti] = at
	}
	return out
}

func mapsClone(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for key := range in {
		out[key] = struct{}{}
	}
	return out
}

func copyEnvelope(env domain.Envelope) domain.Envelope {
	env.Nonce = append([]byte(nil), env.Nonce...)
	env.Ciphertext = append([]byte(nil), env.Ciphertext...)
	env.Tag = append([]byte(nil), env.Tag...)
	env.AAD = append([]byte(nil), env.AAD...)
	env.Parents = append([]int64(nil), env.Parents...)
	return env
}
