package productnotifications

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

var testNow = time.Unix(1_800_000_000, 0).UTC()

func testMessage() Message {
	return Message{ID: "12345678-1234-1234-1234-123456789abc", ProductSlug: Product, Title: "Fixture title", Body: "Fixture body", PublishedAt: testNow.Add(-time.Hour), Buttons: []Button{{ID: "visit", Label: "View", Action: "open_url", URL: "https://example.com/news"}, {ID: "dismiss", Label: "Dismiss", Action: "dismiss"}}}
}
func testSession(subject string) Session {
	return Session{Issuer: "https://issuer.example", Subject: subject, Token: "fixture-token"}
}

type memoryRepo struct {
	mu   sync.Mutex
	l    Ledger
	fail bool
}

func (r *memoryRepo) Load() (Ledger, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.l.Records == nil {
		return Ledger{Schema: 1, Records: map[string]Record{}}, nil
	}
	raw, _ := json.Marshal(r.l)
	var l Ledger
	_ = json.Unmarshal(raw, &l)
	return l, nil
}
func (r *memoryRepo) Save(l Ledger) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail {
		return ErrStore
	}
	raw, _ := json.Marshal(l)
	return json.Unmarshal(raw, &r.l)
}

type fakeInbox struct {
	mu             sync.Mutex
	pages          []Page
	cursors        []string
	receipts       []string
	failEvent      string
	blocked        chan struct{}
	started        chan struct{}
	receiptBlocked chan struct{}
	receiptStarted chan struct{}
}

func (f *fakeInbox) Inbox(ctx context.Context, s Session, cursor string) (Page, error) {
	f.mu.Lock()
	f.cursors = append(f.cursors, cursor)
	index := len(f.cursors) - 1
	blocked, started := f.blocked, f.started
	pages := f.pages
	f.mu.Unlock()
	if blocked != nil {
		if started != nil {
			close(started)
		}
		select {
		case <-ctx.Done():
			return Page{}, ctx.Err()
		case <-blocked:
		}
	}
	if len(pages) == 0 {
		return Page{Enabled: true, Items: []Message{testMessage()}, AllowedActionHosts: []string{"example.com"}}, nil
	}
	return pages[index%len(pages)], nil
}
func (f *fakeInbox) Receipt(ctx context.Context, s Session, id, event, action string) (Receipt, error) {
	f.mu.Lock()
	if event == "action" && f.receiptBlocked != nil {
		blocked, started := f.receiptBlocked, f.receiptStarted
		f.mu.Unlock()
		if started != nil {
			close(started)
		}
		<-blocked
		f.mu.Lock()
	}
	defer f.mu.Unlock()
	if f.failEvent == event {
		return Receipt{}, ErrHTTP
	}
	f.receipts = append(f.receipts, event+":"+action)
	r := Receipt{}
	switch event {
	case "received":
		r.ReceivedAt = &testNow
	case "read":
		r.ReadAt = &testNow
	case "dismissed":
		r.DismissedAt = &testNow
	case "action":
		r.ActionIDs = []string{action}
	}
	return r, nil
}

type fakeTransport struct {
	mu             sync.Mutex
	shown, removed []string
	err            error
}

func (f *fakeTransport) Show(ctx context.Context, m Message, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shown = append(f.shown, id)
	return f.err
}
func (f *fakeTransport) Remove(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, id)
}
func setup(t *testing.T) (*Service, *memoryRepo, *fakeInbox, *fakeTransport, *[]string) {
	t.Helper()
	repo, inbox, native := &memoryRepo{}, &fakeInbox{}, &fakeTransport{}
	opened := []string{}
	s, err := New(inbox, repo, native, func(raw string) error { opened = append(opened, raw); return nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Stop)
	return s, repo, inbox, native, &opened
}
func TestSubmissionDedupeColdActionAndAccountIsolation(t *testing.T) {
	s, repo, inbox, native, opened := setup(t)
	session := testSession("a")
	s.Configure(session, true, true, true)
	if err := s.Poll(context.Background(), testNow); err != nil {
		t.Fatal(err)
	}
	if len(native.shown) != 1 || len(inbox.receipts) != 1 || inbox.receipts[0] != "received:" {
		t.Fatal("submission did not produce exactly a submission receipt")
	}
	id := NativeID(session.Key(), testMessage().ID)
	restored, err := New(inbox, repo, native, func(raw string) error { *opened = append(*opened, raw); return nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()
	restored.Configure(session, true, true, true)
	if err := restored.Poll(context.Background(), testNow.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(native.shown) != 1 {
		t.Fatal("restart duplicated submission")
	}
	restored.Configure(testSession("b"), true, true, true)
	if err := restored.Activate(context.Background(), id, "button-0", testNow); !errors.Is(err, ErrStale) {
		t.Fatal(err)
	}
	if len(*opened) != 0 {
		t.Fatal("previous account URL opened")
	}
	restored.Configure(session, true, true, true)
	if err := restored.Activate(context.Background(), id, "button-0", testNow); err != nil {
		t.Fatal(err)
	}
	if len(*opened) != 1 || inbox.receipts[len(inbox.receipts)-1] != "action:visit" {
		t.Fatal("action was not authorized before opening")
	}
	if err := restored.Activate(context.Background(), id, "button-0", testNow); !errors.Is(err, ErrStale) {
		t.Fatal("repeated activation opened twice")
	}
}
func TestDisabledPermissionBusyAndIntervalNeverPoll(t *testing.T) {
	s, _, inbox, native, _ := setup(t)
	for _, flags := range [][3]bool{{false, true, true}, {true, false, true}, {true, true, false}} {
		s.Configure(testSession("a"), flags[0], flags[1], flags[2])
		if err := s.Poll(context.Background(), testNow); !errors.Is(err, ErrDeferred) {
			t.Fatal(err)
		}
	}
	if len(inbox.cursors) != 0 || len(native.shown) != 0 {
		t.Fatal("disabled or busy configuration used network")
	}
	s.Configure(testSession("a"), true, true, true)
	if err := s.Poll(context.Background(), testNow); err != nil {
		t.Fatal(err)
	}
	if err := s.Poll(context.Background(), testNow.Add(time.Second)); !errors.Is(err, ErrDeferred) {
		t.Fatal("polled more than once per minute")
	}
}
func TestSignOutCancelsInflightInbox(t *testing.T) {
	s, _, inbox, native, _ := setup(t)
	inbox.blocked = make(chan struct{})
	inbox.started = make(chan struct{})
	s.Configure(testSession("a"), true, true, true)
	done := make(chan error, 1)
	go func() { done <- s.Poll(context.Background(), testNow) }()
	<-inbox.started
	s.Configure(Session{}, false, false, false)
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if len(native.shown) != 0 {
		t.Fatal("old account submitted after signout")
	}
}
func TestActionMustPassCurrentBackendValidation(t *testing.T) {
	s, _, inbox, _, opened := setup(t)
	session := testSession("a")
	s.Configure(session, true, true, true)
	if err := s.Poll(context.Background(), testNow); err != nil {
		t.Fatal(err)
	}
	id := NativeID(session.Key(), testMessage().ID)
	for _, action := range []string{"https://example.com", "button--1", "button-01", "button-2", "command"} {
		if err := s.Activate(context.Background(), id, action, testNow); !errors.Is(err, ErrPayload) {
			t.Fatalf("accepted %q", action)
		}
	}
	inbox.failEvent = "action"
	if err := s.Activate(context.Background(), id, "button-0", testNow); !errors.Is(err, ErrHTTP) {
		t.Fatal(err)
	}
	if len(*opened) != 0 {
		t.Fatal("opened on revoked/offline backend failure")
	}
	inbox.failEvent = ""
	if err := s.Activate(context.Background(), id, "button-1", testNow); err != nil {
		t.Fatal(err)
	}
	if inbox.receipts[len(inbox.receipts)-1] != "dismissed:" || len(*opened) != 0 {
		t.Fatal("dismiss opened URL")
	}
}
func TestPersistenceFailurePreventsNativeSubmission(t *testing.T) {
	s, repo, _, native, _ := setup(t)
	repo.fail = true
	s.Configure(testSession("a"), true, true, true)
	if err := s.Poll(context.Background(), testNow); !errors.Is(err, ErrStore) {
		t.Fatal(err)
	}
	if len(native.shown) != 0 {
		t.Fatal("submitted without durable cold metadata")
	}
}
func TestReceiptRetryDoesNotResubmit(t *testing.T) {
	s, _, inbox, native, _ := setup(t)
	inbox.failEvent = "received"
	s.Configure(testSession("a"), true, true, true)
	if err := s.Poll(context.Background(), testNow); !errors.Is(err, ErrHTTP) {
		t.Fatal(err)
	}
	inbox.failEvent = ""
	if err := s.Poll(context.Background(), testNow.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(native.shown) != 1 {
		t.Fatal("receipt retry duplicated notification")
	}
}
func TestPaginationReturnsToNewestPage(t *testing.T) {
	s, _, inbox, native, _ := setup(t)
	cursor := "page-two"
	first := testMessage()
	first.Receipt.ReceivedAt = &testNow
	newest := testMessage()
	newest.ID = "abcdefab-1234-1234-1234-123456789abc"
	head := Page{Enabled: true, Items: []Message{first}, AllowedActionHosts: []string{"example.com"}, NextCursor: &cursor}
	inbox.pages = []Page{head, head, {Enabled: true, Items: []Message{}, AllowedActionHosts: []string{"example.com"}}, {Enabled: true, Items: []Message{newest}, AllowedActionHosts: []string{"example.com"}}, {Enabled: true, Items: []Message{}, AllowedActionHosts: []string{"example.com"}}}
	s.Configure(testSession("a"), true, true, true)
	for i := 0; i < 3; i++ {
		if err := s.Poll(context.Background(), testNow.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if len(inbox.cursors) != 5 || inbox.cursors[0] != "" || inbox.cursors[1] != "" || inbox.cursors[2] != cursor || inbox.cursors[3] != "" || inbox.cursors[4] != cursor || len(native.shown) != 1 {
		t.Fatalf("cursors=%q shows=%d", inbox.cursors, len(native.shown))
	}
}

func TestUncertainReservationIsNeverClaimedSubmitted(t *testing.T) {
	s, repo, inbox, native, _ := setup(t)
	session := testSession("a")
	m := testMessage()
	id := NativeID(session.Key(), m.ID)
	repo.l = Ledger{Schema: 1, Records: map[string]Record{id: {Account: session.Key(), NativeID: id, MessageID: m.ID, Buttons: m.Buttons, Hosts: []string{"example.com"}, ReservedAt: testNow}}}
	restored, err := New(inbox, repo, native, func(string) error { return nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Stop()
	restored.Configure(session, true, true, true)
	if err := restored.Poll(context.Background(), testNow); err != nil {
		t.Fatal(err)
	}
	if len(native.shown) != 0 || len(inbox.receipts) != 0 || !restored.status.Uncertain {
		t.Fatal("uncertain reservation was repeated or acknowledged")
	}
	s.Stop()
}

func TestPruningAcknowledgedRecordDoesNotRepeatServerReceivedMessage(t *testing.T) {
	s, _, inbox, native, _ := setup(t)
	session := testSession("a")
	s.Configure(session, true, true, true)
	oldMessage := testMessage()
	oldMessage.ID = "00000000-0000-0000-0000-000000000000"
	oldID := NativeID(session.Key(), oldMessage.ID)
	for i := 0; i < MaxRecords; i++ {
		messageID := fmt.Sprintf("%08x-0000-0000-0000-000000000000", i)
		id := NativeID(session.Key(), messageID)
		when := testNow.Add(-time.Duration(MaxRecords-i) * time.Hour)
		s.ledger.Records[id] = Record{Account: session.Key(), MessageID: messageID, NativeID: id, ReservedAt: when, SubmittedAt: &when, Hosts: []string{}, Buttons: []Button{}}
	}
	if err := s.Poll(context.Background(), testNow); err != nil {
		t.Fatal(err)
	}
	if len(s.ledger.Records) != MaxRecords || len(native.removed) != 1 || native.removed[0] != oldID {
		t.Fatal("did not prune one oldest acknowledged record")
	}
	oldMessage.Receipt.ReceivedAt = &testNow
	inbox.pages = []Page{{Enabled: true, Items: []Message{oldMessage}, AllowedActionHosts: []string{"example.com"}}}
	if err := s.Poll(context.Background(), testNow.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(native.shown) != 1 {
		t.Fatal("pruned notification was submitted again despite server receipt")
	}
}

func TestButtonURLValidationRejectsCommandsAndPrivateDestinations(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "file:///tmp/executable", "http://example.com", "https://user:password@example.com", "https://example.com:444/path", "https://127.0.0.1", "https://service.local", "https://intranet.internal", "https://example.com\\@untrusted.invalid", "https://other.example"} {
		if ValidateURL(raw, []string{"example.com"}) == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, raw := range []string{"https://example.com/path?a=1&b=2", "https://EXAMPLE.COM:443/"} {
		if err := ValidateURL(raw, []string{"example.com"}); err != nil {
			t.Errorf("rejected allowed destination %q", raw)
		}
	}
}

func TestBusyBeginsDuringFetchDefersSubmissionWithFakeClock(t *testing.T) {
	s, _, inbox, native, _ := setup(t)
	inbox.blocked = make(chan struct{})
	inbox.started = make(chan struct{})
	session := testSession("a")
	s.Configure(session, true, true, true)
	done := make(chan error, 1)
	go func() { done <- s.Poll(context.Background(), testNow) }()
	<-inbox.started
	s.Configure(session, true, true, false)
	close(inbox.blocked)
	if err := <-done; !errors.Is(err, ErrDeferred) {
		t.Fatal(err)
	}
	if len(native.shown) != 0 || len(inbox.receipts) != 0 {
		t.Fatal("active input was disturbed by late fetch")
	}
	inbox.blocked = nil
	inbox.started = nil
	s.Configure(session, true, true, true)
	if err := s.Poll(context.Background(), testNow.Add(time.Second)); !errors.Is(err, ErrDeferred) {
		t.Fatal("input completion triggered an immediate network request")
	}
	if err := s.Poll(context.Background(), testNow.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(native.shown) != 1 {
		t.Fatal("idle interval did not submit")
	}
}
func TestAccountChangesDuringAuthorizedActionNeverOpenOldURL(t *testing.T) {
	s, _, inbox, _, opened := setup(t)
	session := testSession("a")
	s.Configure(session, true, true, true)
	if err := s.Poll(context.Background(), testNow); err != nil {
		t.Fatal(err)
	}
	inbox.receiptBlocked = make(chan struct{})
	inbox.receiptStarted = make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- s.Activate(context.Background(), NativeID(session.Key(), testMessage().ID), "button-0", testNow)
	}()
	<-inbox.receiptStarted
	s.Configure(testSession("b"), true, true, true)
	close(inbox.receiptBlocked)
	if err := <-done; !errors.Is(err, ErrStale) {
		t.Fatal(err)
	}
	if len(*opened) != 0 {
		t.Fatal("old action opened after account generation changed")
	}
}
func TestDeniedNativeSubmissionNeverClaimsReceipt(t *testing.T) {
	s, _, inbox, native, _ := setup(t)
	native.err = ErrAuthorization
	s.Configure(testSession("a"), true, true, true)
	if err := s.Poll(context.Background(), testNow); !errors.Is(err, ErrAuthorization) {
		t.Fatal(err)
	}
	if len(inbox.receipts) != 0 || len(s.ledger.Records) != 0 {
		t.Fatal("native denial was acknowledged or permanently suppressed")
	}
}
