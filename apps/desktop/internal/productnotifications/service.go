package productnotifications

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Inbox interface {
	Inbox(context.Context, Session, string) (Page, error)
	Receipt(context.Context, Session, string, string, string) (Receipt, error)
}
type Record struct {
	Account        string     `json:"account"`
	MessageID      string     `json:"messageId"`
	NativeID       string     `json:"nativeId"`
	Buttons        []Button   `json:"buttons"`
	Hosts          []string   `json:"hosts"`
	ExpiresAt      *time.Time `json:"expiresAt"`
	ReservedAt     time.Time  `json:"reservedAt"`
	SubmittedAt    *time.Time `json:"submittedAt"`
	ReceiptPending bool       `json:"receiptPending"`
	Dismissed      bool       `json:"dismissed"`
}
type Ledger struct {
	Schema  int               `json:"schema"`
	Records map[string]Record `json:"records"`
}
type Repository interface {
	Load() (Ledger, error)
	Save(Ledger) error
}
type Transport interface {
	Show(context.Context, Message, string) error
	Remove(string)
}
type Status struct {
	AccountKey string
	Error      error
	Uncertain  bool
}
type Service struct {
	mu                                    sync.Mutex
	inbox                                 Inbox
	store                                 Repository
	transport                             Transport
	open                                  func(string) error
	changed                               func(Status)
	ledger                                Ledger
	session                               Session
	enabled, allowed, safe, stopped, busy bool
	generation                            uint64
	ctx                                   context.Context
	cancel                                context.CancelFunc
	next                                  time.Time
	cursor                                string
	status                                Status
}

func New(inbox Inbox, store Repository, transport Transport, open func(string) error, changed func(Status)) (*Service, error) {
	if inbox == nil || store == nil || transport == nil || open == nil {
		return nil, ErrPayload
	}
	ledger, err := store.Load()
	if err != nil || ValidateLedger(ledger) != nil {
		return nil, ErrStore
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{inbox: inbox, store: store, transport: transport, open: open, changed: changed, ledger: ledger, ctx: ctx, cancel: cancel}, nil
}
func ValidateLedger(l Ledger) error {
	if l.Schema != 1 || l.Records == nil || len(l.Records) > MaxRecords {
		return ErrStore
	}
	for key, r := range l.Records {
		if len(r.Account) != 64 || !ValidMessageID(r.MessageID) || r.NativeID != key || NativeID(r.Account, r.MessageID) != key || r.ReservedAt.IsZero() || len(r.Buttons) > 2 || len(r.Hosts) > 128 {
			return ErrStore
		}
		m := Message{ID: r.MessageID, ProductSlug: Product, Title: "ledger", Body: "ledger", Buttons: r.Buttons, PublishedAt: r.ReservedAt, ExpiresAt: r.ExpiresAt}
		// Expiry is independently checked on activation; old, expired records remain
		// readable for pruning without becoming valid activations.
		m.ExpiresAt = nil
		if ValidateMessage(m, r.Hosts, r.ReservedAt) != nil {
			return ErrStore
		}
		for _, h := range r.Hosts {
			if !ValidHost(h) {
				return ErrStore
			}
		}
	}
	return nil
}
func (s *Service) Configure(session Session, enabled, allowed, safe bool) {
	s.mu.Lock()
	changed := s.session != session || s.enabled != enabled || s.allowed != allowed
	var remove []string
	if changed {
		previous := s.session.Key()
		if previous != session.Key() {
			s.status = Status{}
		}
		s.cancel()
		s.generation++
		s.ctx, s.cancel = context.WithCancel(context.Background())
		s.next = time.Time{}
		s.cursor = ""
		s.busy = false
		clearPrevious := s.session.Key() != session.Key() || !enabled || !allowed
		for id, r := range s.ledger.Records {
			if clearPrevious && r.Account == previous && !r.Dismissed {
				remove = append(remove, id)
			}
		}
	}
	s.session, s.enabled, s.allowed, s.safe = session, enabled, allowed, safe
	s.mu.Unlock()
	for _, id := range remove {
		s.transport.Remove(id)
	}
}
func (s *Service) Stop() { s.mu.Lock(); s.stopped = true; s.cancel(); s.generation++; s.mu.Unlock() }
func (s *Service) notifyFor(gen uint64, key string, err error) {
	s.mu.Lock()
	if s.generation != gen || s.session.Key() != key {
		s.mu.Unlock()
		return
	}
	s.status.AccountKey, s.status.Error = key, err
	status, fn := s.status, s.changed
	s.mu.Unlock()
	if fn != nil {
		fn(status)
	}
}
func (s *Service) liveLocked(gen uint64, key string) bool {
	return !s.stopped && s.generation == gen && s.enabled && s.allowed && s.session.Valid() && s.session.Key() == key
}
func (s *Service) saveLocked() error {
	if err := s.store.Save(s.ledger); err != nil {
		return ErrStore
	}
	return nil
}

// Poll reads the newest page and at most one catchup page per minute. Bootstrap runs it off the UI thread;
// constructors, input events and disabled/offline profiles perform no requests.
func (s *Service) Poll(parent context.Context, now time.Time) error {
	s.mu.Lock()
	if s.busy || !s.liveLocked(s.generation, s.session.Key()) || !s.safe || now.Before(s.next) {
		s.mu.Unlock()
		return ErrDeferred
	}
	gen, key, session, cursor := s.generation, s.session.Key(), s.session, s.cursor
	s.busy = true
	s.next = now.Add(time.Minute)
	ctx, cancel := context.WithTimeout(s.ctx, 20*time.Second)
	stop := context.AfterFunc(parent, cancel)
	s.mu.Unlock()
	defer cancel()
	defer stop()
	defer func() {
		s.mu.Lock()
		if s.generation == gen {
			s.busy = false
		}
		s.mu.Unlock()
	}()
	// Always inspect the newest page. Independently advance one catchup page
	// so retained history cannot starve announcements published while catching up.
	cursors := []string{""}
	if cursor != "" {
		cursors = append(cursors, cursor)
	}
	firstNext, next := "", ""
	for index, pageCursor := range cursors {
		page, err := s.inbox.Inbox(ctx, session, pageCursor)
		if err == nil {
			err = ValidatePage(page, now)
		}
		if err != nil {
			if ctx.Err() == nil {
				s.notifyFor(gen, key, err)
			}
			return err
		}
		s.mu.Lock()
		live := s.liveLocked(gen, key)
		s.mu.Unlock()
		if !live {
			return ErrStale
		}
		if !page.Enabled {
			return nil
		}
		for _, m := range page.Items {
			if m.ExpiresAt != nil && !m.ExpiresAt.After(now) {
				continue
			}
			if err := s.submit(ctx, gen, key, session, m, page.AllowedActionHosts, now); err != nil {
				if !errors.Is(err, ErrDeferred) && !errors.Is(err, ErrStale) && !errors.Is(err, context.Canceled) {
					s.notifyFor(gen, key, err)
				}
				return err
			}
		}
		next = ""
		if page.NextCursor != nil {
			next = *page.NextCursor
		}
		if index == 0 {
			firstNext = next
		}
	}
	if next == "" && len(cursors) > 1 {
		next = firstNext
	}
	s.mu.Lock()
	if s.liveLocked(gen, key) {
		s.cursor = next
	}
	s.mu.Unlock()
	s.notifyFor(gen, key, nil)
	return nil
}
func (s *Service) submit(ctx context.Context, gen uint64, key string, session Session, m Message, hosts []string, now time.Time) error {
	id := NativeID(key, m.ID)
	s.mu.Lock()
	if !s.liveLocked(gen, key) || ctx.Err() != nil {
		s.mu.Unlock()
		return ErrStale
	}
	if m.Receipt.Handled() {
		if r, ok := s.ledger.Records[id]; ok && r.ReceiptPending {
			r.ReceiptPending = false
			s.ledger.Records[id] = r
			if err := s.saveLocked(); err != nil {
				s.mu.Unlock()
				return err
			}
		}
		s.mu.Unlock()
		return nil
	}
	r, exists := s.ledger.Records[id]
	if exists {
		if r.SubmittedAt == nil {
			s.status.Uncertain = true
			s.mu.Unlock()
			return nil
		}
		pending := r.ReceiptPending && !r.Dismissed
		s.mu.Unlock()
		if pending {
			return s.received(ctx, gen, key, session, id, m.ID)
		}
		return nil
	}
	if !s.safe {
		s.mu.Unlock()
		return ErrDeferred
	}
	// Expired records need no action lookup. At the bound, only acknowledged
	// submissions may be pruned: server receipts independently prevent replay.
	for recordID, record := range s.ledger.Records {
		if record.ExpiresAt != nil && record.ExpiresAt.Before(now.Add(-24*time.Hour)) {
			delete(s.ledger.Records, recordID)
		}
	}
	var pruned []string
	if len(s.ledger.Records) >= MaxRecords {
		oldestID := ""
		var oldest time.Time
		for recordID, record := range s.ledger.Records {
			if record.SubmittedAt != nil && !record.ReceiptPending && (oldestID == "" || record.ReservedAt.Before(oldest)) {
				oldestID = recordID
				oldest = record.ReservedAt
			}
		}
		if oldestID == "" {
			s.mu.Unlock()
			return ErrStore
		}
		delete(s.ledger.Records, oldestID)
		pruned = append(pruned, oldestID)
	}
	r = Record{Account: key, MessageID: m.ID, NativeID: id, Buttons: append([]Button(nil), m.Buttons...), Hosts: append([]string(nil), hosts...), ExpiresAt: m.ExpiresAt, ReservedAt: now}
	s.ledger.Records[id] = r
	if err := s.saveLocked(); err != nil {
		delete(s.ledger.Records, id)
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()
	for _, oldID := range pruned {
		s.transport.Remove(oldID)
	}
	err := s.transport.Show(ctx, m, id)
	s.mu.Lock()
	if !s.liveLocked(gen, key) || ctx.Err() != nil {
		if s.generation == gen {
			s.status.Uncertain = true
		}
		s.mu.Unlock()
		s.transport.Remove(id)
		return ErrStale
	}
	if err != nil {
		delete(s.ledger.Records, id)
		saveErr := s.saveLocked()
		s.mu.Unlock()
		if saveErr != nil {
			return saveErr
		}
		return err
	}
	r = s.ledger.Records[id]
	r.SubmittedAt = &now
	r.ReceiptPending = true
	s.ledger.Records[id] = r
	if err := s.saveLocked(); err != nil {
		s.status.Uncertain = true
		s.mu.Unlock()
		return err
	}
	s.mu.Unlock()
	return s.received(ctx, gen, key, session, id, m.ID)
}
func (s *Service) received(ctx context.Context, gen uint64, key string, session Session, nativeID, messageID string) error {
	receipt, err := s.inbox.Receipt(ctx, session, messageID, "received", "")
	if err != nil {
		return err
	}
	if receipt.ReceivedAt == nil {
		return ErrPayload
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.liveLocked(gen, key) || ctx.Err() != nil {
		return ErrStale
	}
	r, ok := s.ledger.Records[nativeID]
	if !ok {
		return ErrStale
	}
	r.ReceiptPending = false
	s.ledger.Records[nativeID] = r
	return s.saveLocked()
}

// Activate resolves only a persisted current-account record. The backend must
// authorize the receipt against its current recipient/config/expiry state before
// any URL is opened. An activation argument can never supply a URL or command.
func (s *Service) Activate(parent context.Context, nativeID, action string, now time.Time) error {
	s.mu.Lock()
	gen, key, session := s.generation, s.session.Key(), s.session
	r, known := s.ledger.Records[nativeID]
	if !s.liveLocked(gen, key) || !known || r.Account != key || r.Dismissed || r.ExpiresAt != nil && !r.ExpiresAt.After(now) {
		s.mu.Unlock()
		return ErrStale
	}
	event, buttonID, target := "read", "", ""
	if action != "" {
		if !strings.HasPrefix(action, "button-") {
			s.mu.Unlock()
			return ErrPayload
		}
		index, err := strconv.Atoi(strings.TrimPrefix(action, "button-"))
		if err != nil || index < 0 || index >= len(r.Buttons) || action != "button-"+strconv.Itoa(index) {
			s.mu.Unlock()
			return ErrPayload
		}
		b := r.Buttons[index]
		buttonID = b.ID
		switch b.Action {
		case "dismiss":
			event = "dismissed"
			buttonID = ""
		case "open_url":
			event = "action"
			if ValidateURL(b.URL, r.Hosts) != nil {
				s.mu.Unlock()
				return ErrPayload
			}
			target = b.URL
		default:
			s.mu.Unlock()
			return ErrPayload
		}
	}
	ctx, cancel := context.WithTimeout(s.ctx, 20*time.Second)
	stop := context.AfterFunc(parent, cancel)
	s.mu.Unlock()
	defer cancel()
	defer stop()
	receipt, err := s.inbox.Receipt(ctx, session, r.MessageID, event, buttonID)
	if err != nil {
		s.notifyFor(gen, key, err)
		return err
	}
	valid := event == "read" && receipt.ReadAt != nil || event == "dismissed" && receipt.DismissedAt != nil
	if event == "action" {
		for _, id := range receipt.ActionIDs {
			if id == buttonID {
				valid = true
			}
		}
	}
	if !valid {
		return ErrPayload
	}
	s.mu.Lock()
	if !s.liveLocked(gen, key) || ctx.Err() != nil {
		s.mu.Unlock()
		return ErrStale
	}
	r, stillKnown := s.ledger.Records[nativeID]
	if !stillKnown || r.Dismissed {
		s.mu.Unlock()
		return ErrStale
	}
	r.Dismissed = true
	r.ReceiptPending = false
	s.ledger.Records[nativeID] = r
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		s.notifyFor(gen, key, err)
		return err
	}
	// Open is an OS browser adapter that never dispatches to the UI thread.
	// Its final generation check and launch are serialized with sign-out.
	if target != "" {
		if err := s.open(target); err != nil {
			r.Dismissed = false
			s.ledger.Records[nativeID] = r
			saveErr := s.saveLocked()
			s.mu.Unlock()
			if saveErr != nil {
				s.notifyFor(gen, key, saveErr)
				return saveErr
			}
			s.notifyFor(gen, key, ErrOpen)
			return ErrOpen
		}
	}
	s.mu.Unlock()
	s.transport.Remove(nativeID)
	s.notifyFor(gen, key, nil)
	return nil
}
