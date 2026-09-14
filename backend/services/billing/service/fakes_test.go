package service

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"undangan/services/billing/domain"
	"undangan/kernel/apperror"
)

// ---- infrastruktur palsu ----

type fakeTx struct{}

func (fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error { return fn(ctx) }

type fakeLocker struct{ held bool }

func (*fakeLocker) LockTx(context.Context, string) error { return nil }
func (l *fakeLocker) TryLock(context.Context, string) (func(), bool, error) {
	if l.held {
		return nil, false, nil
	}
	l.held = true
	return func() { l.held = false }, true, nil
}

type auditEntry struct{ actor, action, target string }

type fakeAudit struct{ entries []auditEntry }

func (a *fakeAudit) Record(_ context.Context, actorID, action, target string, _ map[string]any) {
	a.entries = append(a.entries, auditEntry{actorID, action, target})
}

func (a *fakeAudit) has(action string) bool {
	for _, e := range a.entries {
		if e.action == action {
			return true
		}
	}
	return false
}

type fakeInvitations struct{ suspended, restored []string }

func (f *fakeInvitations) SuspendForUser(_ context.Context, userID string) (int, error) {
	f.suspended = append(f.suspended, userID)
	return 1, nil
}
func (f *fakeInvitations) RestoreForUser(_ context.Context, userID string) (int, error) {
	f.restored = append(f.restored, userID)
	return 1, nil
}
func (f *fakeInvitations) PurgePersonalData(context.Context, time.Time) (int, error) { return 0, nil }

type nopReadSeekCloser struct{ *strings.Reader }

func (nopReadSeekCloser) Close() error { return nil }

type fakeStorage struct{ files map[string][]byte }

func (s *fakeStorage) SavePublic(context.Context, []byte, string) (string, string, error) {
	return "", "", fmt.Errorf("not used")
}
func (s *fakeStorage) SavePrivate(_ context.Context, folder string, data []byte, ext string) (string, error) {
	key := fmt.Sprintf("%s/%d%s", folder, len(s.files)+1, ext)
	s.files[key] = data
	return key, nil
}
func (s *fakeStorage) OpenPrivate(_ context.Context, key string) (io.ReadSeekCloser, time.Time, error) {
	d, ok := s.files[key]
	if !ok {
		return nil, time.Time{}, fs.ErrNotExist
	}
	return nopReadSeekCloser{strings.NewReader(string(d))}, time.Time{}, nil
}
func (s *fakeStorage) PublicFS() fs.FS { return nil }

// ---- repository palsu ----

type memStore struct {
	mu     sync.Mutex
	seq    int
	plans  map[string]domain.Plan
	subs   map[string]domain.Subscription
	orders map[string]domain.Order
	notifs map[string]domain.Notification
}

func newMemStore() *memStore {
	return &memStore{plans: map[string]domain.Plan{}, subs: map[string]domain.Subscription{},
		orders: map[string]domain.Order{}, notifs: map[string]domain.Notification{}}
}

func (m *memStore) id(prefix string) string {
	m.seq++
	return fmt.Sprintf("%s-%d", prefix, m.seq)
}

type memPlans struct{ *memStore }

func (r memPlans) Create(_ context.Context, p *domain.Plan) error {
	p.ID = r.id("plan")
	r.plans[p.ID] = *p
	return nil
}
func (r memPlans) Update(_ context.Context, p *domain.Plan) error {
	if _, ok := r.plans[p.ID]; !ok {
		return apperror.ErrNotFound
	}
	r.plans[p.ID] = *p
	return nil
}
func (r memPlans) FindByID(_ context.Context, id string) (*domain.Plan, error) {
	p, ok := r.plans[id]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return &p, nil
}
func (r memPlans) List(_ context.Context, activeOnly bool) ([]domain.Plan, error) {
	var out []domain.Plan
	for _, p := range r.plans {
		if !activeOnly || p.IsActive {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Price < out[j].Price
	})
	return out, nil
}
func (r memPlans) Count(context.Context) (int, error) { return len(r.plans), nil }
func (r memPlans) MinActiveMaxGuests(context.Context) (int, bool, error) {
	n, ok := 0, false
	for _, p := range r.plans {
		if p.IsActive && (!ok || p.MaxGuests < n) {
			n, ok = p.MaxGuests, true
		}
	}
	return n, ok, nil
}

type memSubs struct{ *memStore }

func (r memSubs) Create(_ context.Context, s *domain.Subscription) error {
	s.ID = r.id("sub")
	r.subs[s.ID] = *s
	return nil
}
func (r memSubs) Update(_ context.Context, s *domain.Subscription) error {
	if _, ok := r.subs[s.ID]; !ok {
		return apperror.ErrNotFound
	}
	r.subs[s.ID] = *s
	return nil
}
func (r memSubs) FindByID(_ context.Context, id string) (*domain.Subscription, error) {
	s, ok := r.subs[id]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return &s, nil
}
func (r memSubs) FindCurrentForUser(_ context.Context, userID string) (*domain.Subscription, error) {
	var best *domain.Subscription
	for _, s := range r.subs {
		if s.UserID == userID && s.IsLive() && (best == nil || s.EndsAt.After(best.EndsAt)) {
			s := s
			best = &s
		}
	}
	if best == nil {
		return nil, apperror.ErrNotFound
	}
	return best, nil
}
func (r memSubs) ListByUser(_ context.Context, userID string) ([]domain.Subscription, error) {
	var out []domain.Subscription
	for _, s := range r.subs {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}
func (r memSubs) List(_ context.Context, f domain.SubscriptionFilter) ([]domain.Subscription, int, error) {
	var out []domain.Subscription
	for _, s := range r.subs {
		if f.Status == "" || s.Status == f.Status {
			out = append(out, s)
		}
	}
	return out, len(out), nil
}
func (r memSubs) HasLive(_ context.Context, userID string) (bool, error) {
	for _, s := range r.subs {
		if s.UserID == userID && s.IsLive() {
			return true, nil
		}
	}
	return false, nil
}
func (r memSubs) ListByStatusEndingBefore(_ context.Context, status string, t time.Time) ([]domain.Subscription, error) {
	var out []domain.Subscription
	for _, s := range r.subs {
		end := s.EndsAt
		if status == domain.SubscriptionGrace {
			end = s.GraceEndsAt
		}
		if s.Status == status && end.Before(t) {
			out = append(out, s)
		}
	}
	return out, nil
}
func (r memSubs) TransitionStatus(_ context.Context, id, from, to string) (bool, error) {
	s, ok := r.subs[id]
	if !ok || s.Status != from {
		return false, nil
	}
	s.Status = to
	r.subs[id] = s
	return true, nil
}

type memOrders struct{ *memStore }

func (r memOrders) Create(_ context.Context, o *domain.Order) error {
	for _, x := range r.orders {
		if x.Code == o.Code {
			return domain.ErrOrderCodeTaken
		}
	}
	o.ID = r.id("order")
	r.orders[o.ID] = *o
	return nil
}
func (r memOrders) Update(_ context.Context, o *domain.Order) error {
	if _, ok := r.orders[o.ID]; !ok {
		return apperror.ErrNotFound
	}
	r.orders[o.ID] = *o
	return nil
}
func (r memOrders) FindByID(_ context.Context, id string) (*domain.Order, error) {
	o, ok := r.orders[id]
	if !ok {
		return nil, apperror.ErrNotFound
	}
	return &o, nil
}
func (r memOrders) FindByIDForUpdate(ctx context.Context, id string) (*domain.Order, error) {
	return r.FindByID(ctx, id)
}
func (r memOrders) List(_ context.Context, f domain.OrderFilter) ([]domain.Order, int, error) {
	var out []domain.Order
	for _, o := range r.orders {
		if (f.UserID == "" || o.UserID == f.UserID) && (f.Status == "" || o.Status == f.Status) {
			out = append(out, o)
		}
	}
	return out, len(out), nil
}
func (r memOrders) ListPendingByUser(_ context.Context, userID string) ([]domain.Order, error) {
	var out []domain.Order
	for _, o := range r.orders {
		if o.UserID == userID && o.IsPending() {
			out = append(out, o)
		}
	}
	return out, nil
}
func (r memOrders) TakenAmounts(_ context.Context, lo, hi int64, now time.Time) ([]int64, error) {
	var out []int64
	for _, o := range r.orders {
		open := o.Status == domain.OrderAwaitingConfirmation || (o.Status == domain.OrderAwaitingPayment && o.ExpiresAt.After(now))
		if open && o.Amount >= lo && o.Amount <= hi {
			out = append(out, o.Amount)
		}
	}
	return out, nil
}
func (r memOrders) ExpireOverdue(_ context.Context, now time.Time) (int, error) {
	n := 0
	for id, o := range r.orders {
		if o.Status == domain.OrderAwaitingPayment && o.ExpiresAt.Before(now) {
			o.Status = domain.OrderExpired
			r.orders[id] = o
			n++
		}
	}
	return n, nil
}

type memNotifs struct{ *memStore }

func (r memNotifs) Insert(_ context.Context, n domain.Notification) (bool, error) {
	key := n.SubscriptionID + "|" + n.Kind + "|" + n.Channel + "|" + n.Ref
	if _, ok := r.notifs[key]; ok {
		return false, nil
	}
	r.notifs[key] = n
	return true, nil
}

func (r memNotifs) kinds(subID string) []string {
	var out []string
	for _, n := range r.notifs {
		if n.SubscriptionID == subID {
			out = append(out, n.Kind)
		}
	}
	sort.Strings(out)
	return out
}

// ---- harness ----

type harness struct {
	store       *memStore
	clock       time.Time
	audit       *fakeAudit
	invitations *fakeInvitations
	files       *fakeStorage
	locker      *fakeLocker
	plans       PlanService
	subs        *subscriptionService
	orders      *orderService
	lifecycle   *Lifecycle
}

func newHarness() *harness {
	h := &harness{
		store:       newMemStore(),
		clock:       time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC),
		audit:       &fakeAudit{},
		invitations: &fakeInvitations{},
		files:       &fakeStorage{files: map[string][]byte{}},
		locker:      &fakeLocker{},
	}
	now := func() time.Time { return h.clock }
	plans, subs, orders, notifs := memPlans{h.store}, memSubs{h.store}, memOrders{h.store}, memNotifs{h.store}
	h.plans = NewPlanService(plans, h.locker, h.audit, fakeTx{})
	h.subs = NewSubscriptionService(subs, plans, h.invitations, h.locker, h.audit, fakeTx{}).(*subscriptionService)
	h.subs.now = now
	h.orders = NewOrderService(orders, plans, h.subs, h.files, h.locker, h.audit, fakeTx{}).(*orderService)
	h.orders.now = now
	h.lifecycle = NewLifecycle(orders, subs, notifs, h.invitations, h.locker, fakeTx{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	h.lifecycle.now = now
	return h
}
