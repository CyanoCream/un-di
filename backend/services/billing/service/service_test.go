package service

import (
	"context"
	"errors"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"undangan/kernel/notify"

	"undangan/kernel/apperror"
	"undangan/kernel/authctx"
	"undangan/services/billing/domain"
)

const day = 24 * time.Hour

var (
	admin    = authctx.Principal{UserID: "admin-1", Role: authctx.RoleSuperAdmin}
	customer = authctx.Principal{UserID: "user-1", Role: authctx.RoleCustomer}
	other    = authctx.Principal{UserID: "user-2", Role: authctx.RoleCustomer}
	pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00")
)

func code(err error) string {
	var ae *apperror.Error
	if errors.As(err, &ae) {
		return ae.Code
	}
	return ""
}

func ptr[T any](v T) *T { return &v }

func mustPlan(t *testing.T, h *harness, in PlanInput) *domain.Plan {
	t.Helper()
	p, err := h.plans.Create(context.Background(), admin, in)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func basicPlan(t *testing.T, h *harness) *domain.Plan {
	return mustPlan(t, h, PlanInput{Name: ptr("Basic"), Price: ptr(int64(100_000)), DurationDays: ptr(30), GraceDays: ptr(7)})
}

func TestPlanService(t *testing.T) {
	ctx := context.Background()
	h := newHarness()
	if _, err := h.plans.Create(ctx, admin, PlanInput{Name: ptr("X"), Price: ptr(int64(-5)), DurationDays: ptr(0)}); code(err) != "validation" {
		t.Fatalf("expected validation, got %v", err)
	}
	p := basicPlan(t, h)
	if !p.IsActive || p.MaxGuests != 100 || p.MaxInvitations != 1 {
		t.Fatalf("defaults not applied: %+v", p)
	}
	upd, err := h.plans.Update(ctx, admin, p.ID, PlanInput{IsActive: ptr(false), MaxGuests: ptr(250)})
	if err != nil || upd.IsActive || upd.MaxGuests != 250 || upd.Name != "Basic" {
		t.Fatalf("partial update: %v %+v", err, upd)
	}
	if _, err := h.plans.Update(ctx, admin, p.ID, PlanInput{MaxGuests: ptr(0)}); code(err) != "validation" {
		t.Fatalf("expected validation, got %v", err)
	}
	if _, err := h.plans.Update(ctx, admin, "nope", PlanInput{}); code(err) != "not_found" {
		t.Fatalf("expected not_found, got %v", err)
	}
	active, _ := h.plans.ListActive(ctx)
	if len(active) != 0 {
		t.Fatalf("inactive plan listed: %+v", active)
	}
	if !h.audit.has("plan.create") || !h.audit.has("plan.update") {
		t.Fatal("audit missing")
	}

	// EnsureDefaultPlans hanya bila kosong.
	h2 := newHarness()
	for range 2 {
		if err := h2.plans.EnsureDefaultPlans(ctx); err != nil {
			t.Fatal(err)
		}
	}
	all, _ := h2.plans.ListAll(ctx)
	if len(all) != 2 || all[0].Name != "Paket Basic" || all[1].Name != "Paket Premium" || !all[1].AllowCustomDomain || all[1].MaxGuests != 500 {
		t.Fatalf("defaults: %+v", all)
	}
}

func TestGrantNewAndExtend(t *testing.T) {
	ctx := context.Background()
	h := newHarness()
	p := basicPlan(t, h)
	premium := mustPlan(t, h, PlanInput{Name: ptr("Premium"), Price: ptr(int64(175_000)), MaxGuests: ptr(500), GraceDays: ptr(3)})

	s1, err := h.subs.Grant(ctx, "user-1", p.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if s1.Status != domain.SubscriptionActive || !s1.StartsAt.Equal(h.clock) || !s1.EndsAt.Equal(h.clock.Add(30*day)) ||
		!s1.GraceEndsAt.Equal(h.clock.Add(37*day)) || s1.MaxGuests != 100 {
		t.Fatalf("new subscription: %+v", s1)
	}
	if !reflect.DeepEqual(h.invitations.restored, []string{"user-1"}) {
		t.Fatalf("restore not called: %v", h.invitations.restored)
	}

	// 10 hari kemudian, perpanjang dengan paket premium 15 hari → dari ends_at lama.
	h.clock = h.clock.Add(10 * day)
	s2, err := h.subs.GrantByAdmin(ctx, admin, "user-1", premium.ID, 15)
	if err != nil {
		t.Fatal(err)
	}
	if s2.ID != s1.ID || !s2.EndsAt.Equal(s1.EndsAt.Add(15*day)) || !s2.GraceEndsAt.Equal(s2.EndsAt.Add(3*day)) ||
		s2.PlanID != premium.ID || s2.MaxGuests != 500 || !s2.StartsAt.Equal(s1.StartsAt) {
		t.Fatalf("extend active: %+v", s2)
	}
	if !h.audit.has("subscription.grant") {
		t.Fatal("audit subscription.grant missing")
	}

	// Masuk grace (ends_at lewat) → perpanjang dari sekarang.
	h.clock = s2.EndsAt.Add(2 * day)
	st := h.store.subs[s2.ID]
	st.Status = domain.SubscriptionGrace
	h.store.subs[s2.ID] = st
	s3, err := h.subs.Grant(ctx, "user-1", p.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if s3.ID != s1.ID || s3.Status != domain.SubscriptionActive || !s3.EndsAt.Equal(h.clock.Add(30*day)) || s3.MaxGuests != 100 {
		t.Fatalf("extend grace: %+v", s3)
	}

	// Expired → langganan baru.
	st = h.store.subs[s3.ID]
	st.Status = domain.SubscriptionExpired
	h.store.subs[s3.ID] = st
	s4, err := h.subs.Grant(ctx, "user-1", p.ID, 0)
	if err != nil || s4.ID == s1.ID || !s4.StartsAt.Equal(h.clock) {
		t.Fatalf("new after expired: %v %+v", err, s4)
	}

	mine, err := h.subs.ForUser(ctx, "user-1")
	if err != nil || mine.Current == nil || mine.Current.ID != s4.ID || len(mine.History) != 2 {
		t.Fatalf("ForUser: %v %+v", err, mine)
	}
	none, _ := h.subs.ForUser(ctx, "user-9")
	if none.Current != nil || none.History == nil {
		t.Fatalf("empty ForUser: %+v", none)
	}

	// Validasi.
	if _, err := h.subs.Grant(ctx, "user-1", "missing", 0); code(err) != "validation" {
		t.Fatalf("missing plan: %v", err)
	}
	if _, err := h.subs.Grant(ctx, "user-1", p.ID, -1); code(err) != "validation" {
		t.Fatalf("negative days: %v", err)
	}

	// Adapter user module: nil untyped bila tidak ada langganan.
	ua := NewUserAdapter(h.subs)
	if v, err := ua.CurrentForUser(ctx, "user-9"); v != nil || err != nil {
		t.Fatalf("adapter nil: %#v %v", v, err)
	}
	if v, _ := ua.CurrentForUser(ctx, "user-1"); v.(*domain.Subscription).ID != s4.ID {
		t.Fatal("adapter current")
	}
}

func TestOrderFlow(t *testing.T) {
	ctx := context.Background()
	h := newHarness()
	p := basicPlan(t, h)
	inactive := mustPlan(t, h, PlanInput{Name: ptr("Lama"), IsActive: ptr(false)})

	if _, err := h.orders.Create(ctx, customer, inactive.ID); code(err) != "validation" {
		t.Fatalf("inactive plan: %v", err)
	}
	o, err := h.orders.Create(ctx, customer, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != domain.OrderAwaitingPayment || o.UniqueCode < 1 || o.UniqueCode > 499 || o.Amount != p.Price+int64(o.UniqueCode) ||
		!o.ExpiresAt.Equal(h.clock.Add(24*time.Hour)) || len(o.Code) != len("ORD-20260913-7KQ2") {
		t.Fatalf("created: %+v", o)
	}

	// Order kedua ditolak: order_pending + fields.order_id.
	_, err = h.orders.Create(ctx, customer, p.ID)
	var ae *apperror.Error
	if !errors.As(err, &ae) || ae.Code != "order_pending" || ae.Status != 409 || ae.Fields["order_id"] != o.ID {
		t.Fatalf("pending: %#v", err)
	}

	// Customer lain: nominal tidak boleh sama dengan order terbuka.
	o2, err := h.orders.Create(ctx, other, p.ID)
	if err != nil || o2.Amount == o.Amount {
		t.Fatalf("unique amount: %v %+v", err, o2)
	}

	// Kepemilikan.
	if _, err := h.orders.Get(ctx, other, o.ID); code(err) != "not_found" {
		t.Fatalf("other customer must get 404: %v", err)
	}
	if _, err := h.orders.Get(ctx, admin, o.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.orders.UploadProof(ctx, other, o.ID, pngBytes); code(err) != "not_found" {
		t.Fatalf("upload by other: %v", err)
	}

	// Reject hanya saat awaiting_confirmation.
	if _, err := h.orders.Reject(ctx, admin, o.ID, "salah"); code(err) != "invalid_status" {
		t.Fatalf("reject awaiting_payment: %v", err)
	}
	if _, err := h.orders.UploadProof(ctx, customer, o.ID, []byte("bukan gambar")); code(err) != "bad_request" {
		t.Fatalf("non image: %v", err)
	}
	o, err = h.orders.UploadProof(ctx, customer, o.ID, pngBytes)
	if err != nil || o.Status != domain.OrderAwaitingConfirmation || o.ProofUploadedAt == nil || *o.ProofURL() != "/api/v1/orders/"+o.ID+"/proof" {
		t.Fatalf("upload: %v %+v", err, o)
	}
	if _, err := h.orders.Reject(ctx, admin, o.ID, "  "); code(err) != "validation" {
		t.Fatalf("empty reason: %v", err)
	}
	o, err = h.orders.Reject(ctx, admin, o.ID, "Nominal tidak sesuai")
	if err != nil || o.Status != domain.OrderRejected || o.RejectReason != "Nominal tidak sesuai" || !h.audit.has("order.reject") {
		t.Fatalf("reject: %v %+v", err, o)
	}

	// Rejected → upload ulang → approve.
	o, err = h.orders.UploadProof(ctx, customer, o.ID, pngBytes)
	if err != nil || o.Status != domain.OrderAwaitingConfirmation || o.RejectReason != "" {
		t.Fatalf("re-upload: %v %+v", err, o)
	}

	// Bukti hanya untuk pemilik & admin.
	if _, err := h.orders.OpenProof(ctx, other, o.ID); code(err) != "not_found" {
		t.Fatalf("proof by other: %v", err)
	}
	pf, err := h.orders.OpenProof(ctx, customer, o.ID)
	if err != nil || pf.ContentType != "image/png" {
		t.Fatalf("proof: %v %+v", err, pf)
	}
	body, _ := io.ReadAll(pf.Body)
	if string(body) != string(pngBytes) {
		t.Fatal("proof body")
	}
	if _, err := h.orders.OpenProof(ctx, admin, o2.ID); code(err) != "not_found" {
		t.Fatalf("no proof yet: %v", err)
	}

	o, err = h.orders.Approve(ctx, admin, o.ID)
	if err != nil || o.Status != domain.OrderPaid || o.SubscriptionID == nil || *o.ReviewedBy != admin.UserID || o.ReviewedAt == nil {
		t.Fatalf("approve: %v %+v", err, o)
	}
	sub := h.store.subs[*o.SubscriptionID]
	if sub.UserID != customer.UserID || sub.Status != domain.SubscriptionActive || sub.PlanID != p.ID {
		t.Fatalf("subscription: %+v", sub)
	}
	if !h.audit.has("order.approve") || len(h.invitations.restored) != 1 {
		t.Fatal("approve side effects missing")
	}
	if _, err := h.orders.Approve(ctx, admin, o.ID); code(err) != "invalid_status" {
		t.Fatalf("double approve: %v", err)
	}
	if _, err := h.orders.UploadProof(ctx, customer, o.ID, pngBytes); code(err) != "invalid_status" {
		t.Fatalf("upload on paid: %v", err)
	}

	// Setelah paid, customer boleh order lagi (perpanjangan) → approve langsung dari awaiting_payment.
	o3, err := h.orders.Create(ctx, customer, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	o3, err = h.orders.Approve(ctx, admin, o3.ID)
	if err != nil || *o3.SubscriptionID != sub.ID || !h.store.subs[sub.ID].EndsAt.Equal(sub.EndsAt.Add(30*day)) {
		t.Fatalf("extend via second order: %v %+v", err, h.store.subs[sub.ID])
	}

	// Order tertunda yang lewat batas tidak memblokir order baru (ditandai expired).
	h.clock = h.clock.Add(25 * time.Hour)
	if _, err := h.orders.UploadProof(ctx, other, o2.ID, pngBytes); code(err) != "order_expired" {
		t.Fatalf("overdue upload: %v", err)
	}
	o4, err := h.orders.Create(ctx, other, p.ID)
	if err != nil || h.store.orders[o2.ID].Status != domain.OrderExpired || o4.Status != domain.OrderAwaitingPayment {
		t.Fatalf("overdue pending: %v %s", err, h.store.orders[o2.ID].Status)
	}
}

func TestLifecycleRunOnce(t *testing.T) {
	ctx := context.Background()
	h := newHarness()
	p := basicPlan(t, h)
	now := h.clock
	s := h.store
	add := func(userID, status string, endsIn, graceIn time.Duration) string {
		id := s.id("sub")
		s.subs[id] = domain.Subscription{ID: id, UserID: userID, PlanID: p.ID, Status: status,
			StartsAt: now.Add(-30 * day), EndsAt: now.Add(endsIn), GraceEndsAt: now.Add(graceIn)}
		return id
	}
	rem7 := add("u-rem7", domain.SubscriptionActive, 5*day, 12*day)
	rem1 := add("u-rem1", domain.SubscriptionActive, 2*time.Hour, 7*day)
	far := add("u-far", domain.SubscriptionActive, 20*day, 27*day)
	toGrace := add("u-grace", domain.SubscriptionActive, -time.Hour, 6*day)
	toExpire := add("u-exp", domain.SubscriptionGrace, -8*day, -time.Hour)
	// user punya langganan lain yang masih hidup → tidak di-suspend
	toExpireLive := add("u-live", domain.SubscriptionGrace, -8*day, -time.Hour)
	add("u-live", domain.SubscriptionActive, 20*day, 27*day)

	s.orders["o-old"] = domain.Order{ID: "o-old", Status: domain.OrderAwaitingPayment, ExpiresAt: now.Add(-time.Minute)}
	s.orders["o-new"] = domain.Order{ID: "o-new", Status: domain.OrderAwaitingPayment, ExpiresAt: now.Add(time.Hour)}

	for range 2 { // idempotent
		if err := h.lifecycle.RunOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}
	n := memNotifs{s}
	checks := map[string][]string{
		rem7:         {domain.NotifyReminder7d},
		rem1:         {domain.NotifyReminder1d},
		far:          nil,
		toGrace:      {domain.NotifyGrace},
		toExpire:     {domain.NotifyExpired},
		toExpireLive: {domain.NotifyExpired},
	}
	for id, want := range checks {
		if got := n.kinds(id); !reflect.DeepEqual(got, want) {
			t.Errorf("notifications %s: got %v want %v", id, got, want)
		}
	}
	if s.subs[toGrace].Status != domain.SubscriptionGrace || s.subs[toExpire].Status != domain.SubscriptionExpired ||
		s.subs[toExpireLive].Status != domain.SubscriptionExpired || s.subs[rem7].Status != domain.SubscriptionActive {
		t.Fatal("status transitions")
	}
	if !reflect.DeepEqual(h.invitations.suspended, []string{"u-exp"}) {
		t.Fatalf("suspended: %v", h.invitations.suspended)
	}
	if s.orders["o-old"].Status != domain.OrderExpired || s.orders["o-new"].Status != domain.OrderAwaitingPayment {
		t.Fatal("order expiry")
	}

	// Setelah diperpanjang, pengingat periode baru terkirim lagi.
	st := s.subs[rem7]
	st.EndsAt = st.EndsAt.Add(30 * day)
	s.subs[rem7] = st
	h.clock = st.EndsAt.Add(-2 * day)
	if err := h.lifecycle.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if got := n.kinds(rem7); !reflect.DeepEqual(got, []string{domain.NotifyReminder3d, domain.NotifyReminder7d}) {
		t.Fatalf("reminder after extend: %v", got)
	}

	// Lock dipegang instance lain → dilewati.
	h.locker.held = true
	s.orders["o-new2"] = domain.Order{ID: "o-new2", Status: domain.OrderAwaitingPayment, ExpiresAt: h.clock.Add(-time.Hour)}
	if err := h.lifecycle.RunOnce(ctx); err != nil || s.orders["o-new2"].Status != domain.OrderAwaitingPayment {
		t.Fatal("lock not respected")
	}
}

func TestQuotaAdapter(t *testing.T) {
	ctx := context.Background()
	h := newHarness()
	a := NewInvitationQuotaAdapter(memSubs{h.store}, memPlans{h.store})
	a.now = func() time.Time { return h.clock }
	if n, _ := a.DefaultGuestLimit(ctx); n != 100 {
		t.Fatalf("fallback: %d", n)
	}
	mustPlan(t, h, PlanInput{Name: ptr("Paket A"), MaxGuests: ptr(300)})
	mustPlan(t, h, PlanInput{Name: ptr("Paket B"), MaxGuests: ptr(50), IsActive: ptr(false)})
	p := mustPlan(t, h, PlanInput{Name: ptr("Paket C"), MaxGuests: ptr(150), MaxInvitations: ptr(2)})
	if n, _ := a.DefaultGuestLimit(ctx); n != 150 {
		t.Fatalf("min active: %d", n)
	}
	if _, _, _, _, ok, err := a.LimitsForUser(ctx, "user-1"); ok || err != nil {
		t.Fatal("no subscription must be ok=false")
	}
	sub, _ := h.subs.Grant(ctx, "user-1", p.ID, 0)
	status, inv, guests, endsAt, ok, err := a.LimitsForUser(ctx, "user-1")
	if err != nil || !ok || status != "active" || inv != 2 || guests != 150 || !endsAt.Equal(sub.EndsAt) {
		t.Fatalf("limits: %s %d %d %v %v %v", status, inv, guests, endsAt, ok, err)
	}
	h.clock = sub.EndsAt.Add(day)
	if status, _, _, _, ok, _ := a.LimitsForUser(ctx, "user-1"); !ok || status != "grace" {
		t.Fatalf("effective grace: %s %v", status, ok)
	}
	h.clock = sub.GraceEndsAt.Add(time.Hour)
	if _, _, _, _, ok, _ := a.LimitsForUser(ctx, "user-1"); ok {
		t.Fatal("effective expired must be ok=false")
	}
}

func TestNormalizePaymentSettings(t *testing.T) {
	ok, err := NormalizePaymentSettings(domain.PaymentSettings{QRISImage: " /uploads/2026/09/q.png ", MerchantName: " Toko ", AdminWhatsApp: "0812-3456-7890"})
	if err != nil || ok.AdminWhatsApp != "6281234567890" || ok.MerchantName != "Toko" || ok.QRISImage != "/uploads/2026/09/q.png" {
		t.Fatalf("%v %+v", err, ok)
	}
	_, err = NormalizePaymentSettings(domain.PaymentSettings{AdminWhatsApp: "021-555", QRISImage: "javascript:alert(1)"})
	var ae *apperror.Error
	if !errors.As(err, &ae) || ae.Fields["admin_whatsapp"] == "" || ae.Fields["qris_image"] == "" {
		t.Fatalf("expected validation: %v", err)
	}
	if _, err := NormalizePaymentSettings(domain.PaymentSettings{AdminWhatsApp: "+62 21 5550 1234"}); code(err) != "validation" {
		t.Fatalf("landline must fail: %v", err)
	}
}

// Notifikasi (Telegram) dikirim pada peristiwa order, dan tombol Setujui/Tolak hanya ada saat bukti masuk.
func TestOrderNotifications(t *testing.T) {
	ctx := context.Background()
	h := newHarness()
	p := basicPlan(t, h)

	o, err := h.orders.Create(ctx, customer, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.orders.UploadProof(ctx, customer, o.ID, pngBytes); err != nil {
		t.Fatal(err)
	}
	if _, err := h.orders.Approve(ctx, admin, o.ID); err != nil {
		t.Fatal(err)
	}

	want := []string{notify.KindOrderCreated, notify.KindOrderProof, notify.KindOrderApproved}
	if got := h.notify.kinds(); !slices.Equal(got, want) {
		t.Fatalf("kinds = %v, want %v", got, want)
	}
	proof := h.notify.sent[1]
	if len(proof.Actions) != 2 || proof.Actions[0].Data != "order:approve:"+o.ID || proof.Actions[1].Data != "order:reject:"+o.ID {
		t.Errorf("tombol aksi = %+v", proof.Actions)
	}
	if len(h.notify.sent[0].Actions) != 0 || len(h.notify.sent[2].Actions) != 0 {
		t.Error("notifikasi selain bukti transfer tidak boleh punya tombol")
	}
	joined := strings.Join(proof.Lines, "|")
	if !strings.Contains(joined, o.Code) || !strings.Contains(joined, "Rp") {
		t.Errorf("isi notifikasi: %v", proof.Lines)
	}

	// Order kedua yang ditolak → notifikasi penolakan berisi alasan.
	o2, err := h.orders.Create(ctx, other, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.orders.UploadProof(ctx, other, o2.ID, pngBytes); err != nil {
		t.Fatal(err)
	}
	if _, err := h.orders.Reject(ctx, admin, o2.ID, "nominal tidak sesuai"); err != nil {
		t.Fatal(err)
	}
	last := h.notify.sent[len(h.notify.sent)-1]
	if last.Kind != notify.KindOrderRejected || !strings.Contains(strings.Join(last.Lines, "|"), "nominal tidak sesuai") {
		t.Errorf("notifikasi tolak = %+v", last)
	}
}
