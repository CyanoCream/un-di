package domain

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"undangan/kernel/apperror"
)

const day = 24 * time.Hour

var t0 = time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)

func plan() *Plan {
	return &Plan{ID: "p1", Name: "Basic", Price: 100_000, DurationDays: 30, GraceDays: 7, MaxInvitations: 1, MaxGuests: 100}
}

func errCode(err error) string {
	var ae *apperror.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func TestExtendPeriod(t *testing.T) {
	p := plan()
	tests := []struct {
		name               string
		current            *Subscription
		days               int
		wantStart, wantEnd time.Time
		wantGraceEnd       time.Time
	}{
		{"no subscription", nil, 0, t0, t0.Add(30 * day), t0.Add(37 * day)},
		{"custom days", nil, 10, t0, t0.Add(10 * day), t0.Add(17 * day)},
		{
			"active, ends in future → extend from ends_at",
			&Subscription{Status: SubscriptionActive, StartsAt: t0.Add(-20 * day), EndsAt: t0.Add(10 * day)},
			0, t0.Add(-20 * day), t0.Add(40 * day), t0.Add(47 * day),
		},
		{
			"grace, ended → extend from now",
			&Subscription{Status: SubscriptionGrace, StartsAt: t0.Add(-33 * day), EndsAt: t0.Add(-3 * day)},
			0, t0.Add(-33 * day), t0.Add(30 * day), t0.Add(37 * day),
		},
		{
			"expired → new period",
			&Subscription{Status: SubscriptionExpired, StartsAt: t0.Add(-90 * day), EndsAt: t0.Add(-60 * day)},
			0, t0, t0.Add(30 * day), t0.Add(37 * day),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, e, g := ExtendPeriod(t0, tt.current, p, tt.days)
			if !s.Equal(tt.wantStart) || !e.Equal(tt.wantEnd) || !g.Equal(tt.wantGraceEnd) {
				t.Fatalf("got %v %v %v, want %v %v %v", s, e, g, tt.wantStart, tt.wantEnd, tt.wantGraceEnd)
			}
		})
	}
}

func TestApplyGrant(t *testing.T) {
	p := plan()
	p.ID, p.MaxGuests = "p2", 500
	cur := &Subscription{ID: "s1", UserID: "u1", PlanID: "p1", Status: SubscriptionGrace, StartsAt: t0.Add(-40 * day),
		EndsAt: t0.Add(-2 * day), MaxGuests: 100, MaxInvitations: 1}
	got := ApplyGrant(t0, cur, "u1", p, 0)
	if got.ID != "s1" || got.Status != SubscriptionActive || got.PlanID != "p2" || got.MaxGuests != 500 {
		t.Fatalf("unexpected %+v", got)
	}
	if cur.Status != SubscriptionGrace {
		t.Fatal("ApplyGrant must not mutate current")
	}
	fresh := ApplyGrant(t0, &Subscription{ID: "old", Status: SubscriptionExpired}, "u1", p, 0)
	if fresh.ID != "" || fresh.UserID != "u1" || !fresh.StartsAt.Equal(t0) {
		t.Fatalf("expected new subscription, got %+v", fresh)
	}
}

func TestEffectiveStatus(t *testing.T) {
	s := &Subscription{Status: SubscriptionActive, EndsAt: t0, GraceEndsAt: t0.Add(7 * day)}
	if got := s.EffectiveStatus(t0.Add(-time.Hour)); got != SubscriptionActive {
		t.Fatal(got)
	}
	if got := s.EffectiveStatus(t0.Add(time.Hour)); got != SubscriptionGrace {
		t.Fatal(got)
	}
	if got := s.EffectiveStatus(t0.Add(8 * day)); got != SubscriptionExpired {
		t.Fatal(got)
	}
}

func TestReminderKind(t *testing.T) {
	cases := map[time.Duration]string{
		10 * day:       "",
		7 * day:        NotifyReminder7d,
		4 * day:        NotifyReminder7d,
		3 * day:        NotifyReminder3d,
		36 * time.Hour: NotifyReminder3d,
		24 * time.Hour: NotifyReminder1d,
		time.Minute:    NotifyReminder1d,
		0:              "",
		-time.Hour:     "",
	}
	for left, want := range cases {
		got, ok := ReminderKind(t0, t0.Add(left))
		if got != want || ok != (want != "") {
			t.Errorf("left %v: got %q %v, want %q", left, got, ok, want)
		}
	}
}

func TestPickUniqueCode(t *testing.T) {
	code, err := PickUniqueCode(100_000, []int64{100_010, 100_011}, 10)
	if err != nil || code != 12 {
		t.Fatalf("got %d %v", code, err)
	}
	// wrap around 499 → 1
	code, err = PickUniqueCode(0, []int64{499}, 499)
	if err != nil || code != 1 {
		t.Fatalf("got %d %v", code, err)
	}
	var all []int64
	for i := MinUniqueCode; i <= MaxUniqueCode; i++ {
		all = append(all, OrderAmount(5000, i))
	}
	if _, err := PickUniqueCode(5000, all, 1); errCode(err) != "unique_code_exhausted" {
		t.Fatalf("expected exhausted, got %v", err)
	}
	for start := -3; start < 600; start += 97 {
		c, err := PickUniqueCode(1, nil, start)
		if err != nil || c < MinUniqueCode || c > MaxUniqueCode {
			t.Fatalf("start %d → %d %v", start, c, err)
		}
	}
}

func TestNewOrderCode(t *testing.T) {
	// 20:00 UTC = 03:00 WIB keesokan harinya.
	got := NewOrderCode(time.Date(2026, 9, 12, 20, 0, 0, 0, time.UTC), "7KQ2")
	if got != "ORD-20260913-7KQ2" {
		t.Fatal(got)
	}
	if !regexp.MustCompile(`^ORD-\d{8}-[A-Z0-9]{4}$`).MatchString(got) {
		t.Fatal("format")
	}
}

func TestNewOrder(t *testing.T) {
	o := NewOrder("u1", plan(), 123, t0)
	if o.Amount != 100_123 || o.Status != OrderAwaitingPayment || !o.ExpiresAt.Equal(t0.Add(24*time.Hour)) {
		t.Fatalf("%+v", o)
	}
}

func TestOrderTransitions(t *testing.T) {
	newOrder := func(status string) *Order {
		return &Order{ID: "o1", Status: status, ExpiresAt: t0.Add(time.Hour)}
	}

	// upload proof
	o := newOrder(OrderAwaitingPayment)
	if err := o.AttachProof("proofs/a.jpg", t0); err != nil || o.Status != OrderAwaitingConfirmation || o.ProofUploadedAt == nil {
		t.Fatalf("attach: %v %+v", err, o)
	}
	if err := o.AttachProof("proofs/b.jpg", t0); errCode(err) != "invalid_status" {
		t.Fatalf("re-upload while awaiting_confirmation: %v", err)
	}
	if err := newOrder(OrderAwaitingPayment).AttachProof("x", t0.Add(2*time.Hour)); errCode(err) != "order_expired" {
		t.Fatalf("overdue upload: %v", err)
	}
	for _, st := range []string{OrderPaid, OrderExpired} {
		if err := newOrder(st).AttachProof("x", t0); err == nil {
			t.Fatalf("upload on %s must fail", st)
		}
	}
	rej := newOrder(OrderRejected)
	rej.RejectReason = "buram"
	rej.ExpiresAt = t0.Add(-48 * time.Hour) // rejected boleh upload ulang walau lewat batas
	if err := rej.AttachProof("x", t0); err != nil || rej.RejectReason != "" || rej.Status != OrderAwaitingConfirmation {
		t.Fatalf("rejected re-upload: %v %+v", err, rej)
	}

	// approve
	for st, ok := range map[string]bool{OrderAwaitingPayment: true, OrderAwaitingConfirmation: true, OrderPaid: false, OrderRejected: false, OrderExpired: false} {
		o := newOrder(st)
		err := o.Approve("admin", "s1", t0)
		if (err == nil) != ok {
			t.Fatalf("approve %s: %v", st, err)
		}
		if ok && (o.Status != OrderPaid || *o.SubscriptionID != "s1" || *o.ReviewedBy != "admin") {
			t.Fatalf("approve result %+v", o)
		}
	}

	// reject
	if err := newOrder(OrderAwaitingConfirmation).Reject("admin", "   ", t0); errCode(err) != "validation" {
		t.Fatalf("empty reason: %v", err)
	}
	if err := newOrder(OrderAwaitingConfirmation).Reject("admin", strings.Repeat("x", 501), t0); errCode(err) != "validation" {
		t.Fatalf("long reason: %v", err)
	}
	if err := newOrder(OrderAwaitingPayment).Reject("admin", "salah", t0); errCode(err) != "invalid_status" {
		t.Fatalf("reject awaiting_payment: %v", err)
	}
	o = newOrder(OrderAwaitingConfirmation)
	if err := o.Reject("admin", "  nominal tidak sesuai ", t0); err != nil || o.Status != OrderRejected || o.RejectReason != "nominal tidak sesuai" {
		t.Fatalf("reject: %v %+v", err, o)
	}

	// expire
	if err := newOrder(OrderAwaitingPayment).Expire(t0); err == nil {
		t.Fatal("expire before expires_at must fail")
	}
	o = newOrder(OrderAwaitingPayment)
	if err := o.Expire(t0.Add(time.Hour)); err != nil || o.Status != OrderExpired {
		t.Fatalf("expire: %v", err)
	}
}

func TestOrderJSON(t *testing.T) {
	o := Order{ID: "o1", Code: "ORD-1", Status: OrderAwaitingPayment}
	b, _ := json.Marshal(o)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	if m["proof_url"] != nil {
		t.Fatalf("proof_url should be null: %s", b)
	}
	for _, k := range []string{"id", "code", "user_id", "user_name", "user_email", "plan_id", "plan_name", "price", "unique_code",
		"amount", "status", "proof_url", "proof_uploaded_at", "reject_reason", "reviewed_by_name", "reviewed_at", "expires_at", "created_at"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing key %s", k)
		}
	}
	if len(m) != 18 {
		t.Errorf("unexpected extra keys: %s", b)
	}
	o.ProofPath = "proofs/x.jpg"
	b, _ = json.Marshal(&o)
	if !strings.Contains(string(b), `"proof_url":"/api/v1/orders/o1/proof"`) || strings.Contains(string(b), "proofs/x.jpg") {
		t.Fatalf("proof_url: %s", b)
	}
}

func TestPlanNormalize(t *testing.T) {
	p := Plan{Name: " A ", Price: -1, DurationDays: 0, GraceDays: -1, MaxInvitations: 0, MaxGuests: 0}
	err := p.Normalize()
	var ae *apperror.Error
	if !errors.As(err, &ae) {
		t.Fatal("expected validation error")
	}
	for _, f := range []string{"name", "price", "duration_days", "grace_days", "max_invitations", "max_guests"} {
		if _, ok := ae.Fields[f]; !ok {
			t.Errorf("missing field error %s", f)
		}
	}
	ok := NewPlanDefaults()
	ok.Name = "  Paket Hemat "
	if err := ok.Normalize(); err != nil || ok.Name != "Paket Hemat" {
		t.Fatal(err, ok.Name)
	}
	for _, d := range DefaultPlans() {
		if err := d.Normalize(); err != nil {
			t.Fatal(err)
		}
	}
}
