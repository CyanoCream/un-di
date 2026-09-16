package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
	"undangan/kernel/notify"

	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/kernel/authctx"
	"undangan/kernel/database"
	"undangan/kernel/httpx"
	"undangan/kernel/security"
	"undangan/kernel/storage"
	"undangan/services/billing/controller"
	"undangan/services/billing/domain"
	"undangan/services/billing/repository"
	"undangan/services/billing/service"
)

// Jalankan dengan BILLING_TEST_DATABASE_URL=postgres://... (database yang sudah dimigrasi).

type stubInvitations struct {
	mu                  sync.Mutex
	suspended, restored []string
	purged              int
}

func (s *stubInvitations) SuspendForUser(_ context.Context, userID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.suspended = append(s.suspended, userID)
	return 1, nil
}

func (s *stubInvitations) RestoreForUser(_ context.Context, userID string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.restored = append(s.restored, userID)
	return 0, nil
}

func (s *stubInvitations) PurgePersonalData(context.Context, time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.purged++
	return 0, nil
}

func (s *stubInvitations) count(list *[]string, userID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, u := range *list {
		if u == userID {
			n++
		}
	}
	return n
}

type env struct {
	pool      *pgxpool.Pool
	inv       *stubInvitations
	plans     service.PlanService
	orders    service.OrderService
	subs      service.SubscriptionService
	settings  service.SettingsService
	lifecycle *service.Lifecycle
	quota     *service.InvitationQuotaAdapter
	handler   http.Handler
}

func setup(t *testing.T) *env {
	url := os.Getenv("BILLING_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("BILLING_TEST_DATABASE_URL tidak di-set")
	}
	ctx := context.Background()
	pool, err := database.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	tx := database.NewTxManager(pool)
	audit := pgAudit{pool: pool}
	planRepo, subRepo, orderRepo := repository.NewPlans(pool), repository.NewSubscriptions(pool), repository.NewOrders(pool)
	locker := repository.NewLocker(pool)
	files := storage.NewLocal(t.TempDir(), t.TempDir())
	e := &env{pool: pool, inv: &stubInvitations{}}
	e.plans = service.NewPlanService(planRepo, locker, audit, tx)
	e.subs = service.NewSubscriptionService(subRepo, planRepo, e.inv, locker, audit, tx)
	e.orders = service.NewOrderService(orderRepo, planRepo, e.subs, files, locker, audit, notify.Nop{}, "", tx)
	e.settings = service.NewSettingsService(repository.NewSettings(pool), audit)
	e.lifecycle = service.NewLifecycle(orderRepo, subRepo, repository.NewNotifications(pool), e.inv, locker, tx, log)
	e.quota = service.NewInvitationQuotaAdapter(subRepo, planRepo)

	mux := http.NewServeMux()
	controller.New(e.plans, e.orders, e.subs, e.settings).Register(httpx.NewRouter(mux))
	// Pengganti middleware auth: identitas dari header uji.
	e.handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := r.Header.Get("X-Test-User"); id != "" {
			r = r.WithContext(authctx.With(r.Context(), authctx.Principal{UserID: id, Role: r.Header.Get("X-Test-Role")}))
		}
		mux.ServeHTTP(w, r)
	})
	return e
}

func (e *env) user(t *testing.T, role string) authctx.Principal {
	t.Helper()
	email := fmt.Sprintf("billing-%s@test.local", security.RandomCode(10))
	var id string
	if err := e.pool.QueryRow(context.Background(), `INSERT INTO users (email, name, password_hash, role)
		VALUES ($1, $2, 'x', $3::user_role) RETURNING id`, email, "Uji "+role, role).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return authctx.Principal{UserID: id, Role: role, Email: email}
}

func (e *env) do(t *testing.T, p authctx.Principal, method, path string, body io.Reader, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if p.UserID != "" {
		req.Header.Set("X-Test-User", p.UserID)
		req.Header.Set("X-Test-Role", p.Role)
	}
	rec := httptest.NewRecorder()
	e.handler.ServeHTTP(rec, req)
	return rec
}

func (e *env) json(t *testing.T, p authctx.Principal, method, path string, in any, wantStatus int, out any) {
	t.Helper()
	var body io.Reader
	if in != nil {
		b, _ := json.Marshal(in)
		body = bytes.NewReader(b)
	}
	rec := e.do(t, p, method, path, body, "application/json")
	if rec.Code != wantStatus {
		t.Fatalf("%s %s: status %d, want %d: %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("%s %s: %v: %s", method, path, err, rec.Body.String())
		}
	}
}

var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x02\x00\x00\x00")

func TestIntegrationOrderApproveFlow(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	if err := e.plans.EnsureDefaultPlans(ctx); err != nil {
		t.Fatal(err)
	}
	if err := e.plans.EnsureDefaultPlans(ctx); err != nil {
		t.Fatal(err)
	}
	admin := e.user(t, authctx.RoleSuperAdmin)
	cust := e.user(t, authctx.RoleCustomer)
	other := e.user(t, authctx.RoleCustomer)

	// Admin buat paket, customer melihat paket aktif.
	var plan domain.Plan
	e.json(t, admin, "POST", "/api/v1/admin/plans", map[string]any{"name": "Paket Uji " + security.RandomCode(4), "price": 150000, "max_guests": 250, "sort_order": 99}, 201, &plan)
	var patched domain.Plan
	e.json(t, admin, "PATCH", "/api/v1/admin/plans/"+plan.ID, map[string]any{"description": "untuk uji"}, 200, &patched)
	if patched.Description != "untuk uji" || patched.Price != 150000 || patched.MaxGuests != 250 {
		t.Fatalf("patch: %+v", patched)
	}
	e.json(t, admin, "PATCH", "/api/v1/admin/plans/not-a-uuid", map[string]any{"name": "Nama"}, 404, nil)
	var plans httpx.List[map[string]any]
	e.json(t, cust, "GET", "/api/v1/plans", nil, 200, &plans)
	if len(plans.Items) < 3 {
		t.Fatalf("plans: %+v", plans)
	}
	e.json(t, cust, "GET", "/api/v1/admin/plans", nil, 403, nil)

	// Order.
	var order map[string]any
	e.json(t, cust, "POST", "/api/v1/me/orders", map[string]any{"plan_id": plan.ID}, 201, &order)
	orderID := order["id"].(string)
	if order["status"] != "awaiting_payment" || order["proof_url"] != nil || order["user_email"] != cust.Email ||
		order["plan_name"] != plan.Name || int64(order["amount"].(float64)) != 150000+int64(order["unique_code"].(float64)) {
		t.Fatalf("order: %+v", order)
	}
	var pending struct {
		Error struct {
			Code   string            `json:"code"`
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	e.json(t, cust, "POST", "/api/v1/me/orders", map[string]any{"plan_id": plan.ID}, 409, &pending)
	if pending.Error.Code != "order_pending" || pending.Error.Fields["order_id"] != orderID {
		t.Fatalf("pending: %+v", pending)
	}
	var otherOrder domain.Order
	e.json(t, other, "POST", "/api/v1/me/orders", map[string]any{"plan_id": plan.ID}, 201, &otherOrder)
	if otherOrder.Amount == int64(order["amount"].(float64)) {
		t.Fatal("amount must be unique among open orders")
	}
	e.json(t, other, "GET", "/api/v1/me/orders/"+orderID, nil, 404, nil)
	e.json(t, admin, "GET", "/api/v1/admin/orders/"+orderID, nil, 200, nil)

	var mine httpx.Paginated[domain.Order]
	e.json(t, cust, "GET", "/api/v1/me/orders", nil, 200, &mine)
	if mine.Total != 1 || mine.Items[0].ID != orderID {
		t.Fatalf("my orders: %+v", mine)
	}

	// Upload bukti (multipart).
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "bukti.png")
	_, _ = fw.Write(pngBytes)
	_ = mw.Close()
	rec := e.do(t, cust, "POST", "/api/v1/me/orders/"+orderID+"/proof", &buf, mw.FormDataContentType())
	if rec.Code != 200 {
		t.Fatalf("upload: %d %s", rec.Code, rec.Body.String())
	}
	var uploaded map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &uploaded)
	if uploaded["status"] != "awaiting_confirmation" || uploaded["proof_url"] != "/api/v1/orders/"+orderID+"/proof" {
		t.Fatalf("uploaded: %+v", uploaded)
	}

	// Download bukti.
	rec = e.do(t, cust, "GET", "/api/v1/orders/"+orderID+"/proof", nil, "")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || rec.Header().Get("Cache-Control") != "private, no-store" ||
		!bytes.Equal(rec.Body.Bytes(), pngBytes) {
		t.Fatalf("proof: %d %v", rec.Code, rec.Header())
	}
	if rec := e.do(t, other, "GET", "/api/v1/orders/"+orderID+"/proof", nil, ""); rec.Code != 404 {
		t.Fatalf("proof other: %d", rec.Code)
	}
	if rec := e.do(t, admin, "GET", "/api/v1/orders/"+orderID+"/proof", nil, ""); rec.Code != 200 {
		t.Fatalf("proof admin: %d", rec.Code)
	}

	// Reject → upload ulang → approve.
	e.json(t, admin, "POST", "/api/v1/admin/orders/"+orderID+"/reject", map[string]any{"reason": ""}, 422, nil)
	var rejected domain.Order
	e.json(t, admin, "POST", "/api/v1/admin/orders/"+orderID+"/reject", map[string]any{"reason": "Bukti buram"}, 200, &rejected)
	if rejected.Status != "rejected" || rejected.RejectReason != "Bukti buram" || rejected.ReviewedByName == nil {
		t.Fatalf("rejected: %+v", rejected)
	}
	if _, err := e.orders.UploadProof(ctx, cust, orderID, pngBytes); err != nil {
		t.Fatal(err)
	}
	var approved map[string]any
	e.json(t, admin, "POST", "/api/v1/admin/orders/"+orderID+"/approve", nil, 200, &approved)
	if approved["status"] != "paid" || approved["reviewed_by_name"] != "Uji super_admin" || approved["reject_reason"] != "" {
		t.Fatalf("approved: %+v", approved)
	}
	e.json(t, admin, "POST", "/api/v1/admin/orders/"+orderID+"/approve", nil, 409, nil)
	if e.inv.count(&e.inv.restored, cust.UserID) != 1 {
		t.Fatal("RestoreForUser not called")
	}
	var auditCount int
	_ = e.pool.QueryRow(ctx, `SELECT count(*) FROM audit_logs WHERE target = $1 AND action IN ('order.approve', 'order.reject')`, "order:"+orderID).Scan(&auditCount)
	if auditCount != 2 {
		t.Fatalf("audit rows: %d", auditCount)
	}

	// Langganan customer.
	var my struct {
		Current *map[string]any  `json:"current"`
		History []map[string]any `json:"history"`
	}
	e.json(t, cust, "GET", "/api/v1/me/subscription", nil, 200, &my)
	if my.Current == nil || (*my.Current)["status"] != "active" || (*my.Current)["plan_name"] != plan.Name ||
		int((*my.Current)["max_guests"].(float64)) != 250 || len(my.History) != 1 {
		t.Fatalf("me/subscription: %+v", my)
	}
	var none struct {
		Current *domain.Subscription  `json:"current"`
		History []domain.Subscription `json:"history"`
	}
	rec = e.do(t, other, "GET", "/api/v1/me/subscription", nil, "")
	if rec.Code != 200 || !bytes.Contains(rec.Body.Bytes(), []byte(`"current":null`)) || !bytes.Contains(rec.Body.Bytes(), []byte(`"history":[]`)) {
		t.Fatalf("empty subscription: %s", rec.Body.String())
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &none)

	// Grant manual admin memperpanjang langganan yang sama.
	firstEnd := (*my.Current)["ends_at"].(string)
	var granted domain.Subscription
	e.json(t, admin, "POST", "/api/v1/admin/users/"+cust.UserID+"/subscriptions", map[string]any{"plan_id": plan.ID, "days": 10}, 201, &granted)
	end0, _ := time.Parse(time.RFC3339Nano, firstEnd)
	if granted.ID != (*my.Current)["id"] || !granted.EndsAt.Equal(end0.Add(10*24*time.Hour)) {
		t.Fatalf("grant extend: %+v (was %s)", granted, firstEnd)
	}
	e.json(t, admin, "POST", "/api/v1/admin/users/00000000-0000-0000-0000-000000000000/subscriptions", map[string]any{"plan_id": plan.ID}, 404, nil)
	e.json(t, admin, "POST", "/api/v1/admin/users/"+cust.UserID+"/subscriptions", map[string]any{"plan_id": "bad"}, 422, nil)

	var subsPage httpx.Paginated[domain.Subscription]
	e.json(t, admin, "GET", "/api/v1/admin/subscriptions?status=active&q="+cust.Email, nil, 200, &subsPage)
	if subsPage.Total != 1 || subsPage.Items[0].UserEmail != cust.Email {
		t.Fatalf("admin subscriptions: %+v", subsPage)
	}
	var ordersPage httpx.Paginated[domain.Order]
	e.json(t, admin, "GET", "/api/v1/admin/orders?status=awaiting_payment&q="+other.Email, nil, 200, &ordersPage)
	if ordersPage.Total != 1 || ordersPage.Items[0].ID != otherOrder.ID {
		t.Fatalf("admin orders: %+v", ordersPage)
	}

	// Quota adapter.
	status, maxInv, maxGuests, _, ok, err := e.quota.LimitsForUser(ctx, cust.UserID)
	if err != nil || !ok || status != "active" || maxInv != 1 || maxGuests != 250 {
		t.Fatalf("limits: %s %d %d %v %v", status, maxInv, maxGuests, ok, err)
	}
	if _, _, _, _, ok, err := e.quota.LimitsForUser(ctx, other.UserID); ok || err != nil {
		t.Fatal("other has no subscription")
	}
	if n, err := e.quota.DefaultGuestLimit(ctx); err != nil || n < 1 || n > 100 {
		t.Fatalf("default guest limit: %d %v", n, err)
	}

	// Pengaturan pembayaran.
	var ps domain.PaymentSettings
	e.json(t, admin, "PUT", "/api/v1/admin/settings/payment", map[string]any{"qris_image": "/uploads/q.png", "merchant_name": "Undangan", "instructions": "Scan", "admin_whatsapp": "0812 3456 7890"}, 200, &ps)
	if ps.AdminWhatsApp != "6281234567890" {
		t.Fatalf("settings: %+v", ps)
	}
	e.json(t, admin, "PUT", "/api/v1/admin/settings/payment", map[string]any{"admin_whatsapp": "abc"}, 422, nil)
	e.json(t, cust, "PUT", "/api/v1/admin/settings/payment", map[string]any{}, 403, nil)
	e.json(t, cust, "GET", "/api/v1/payment-settings", nil, 200, &ps)
	if ps.MerchantName != "Undangan" || ps.QRISImage != "/uploads/q.png" {
		t.Fatalf("settings read: %+v", ps)
	}
	e.json(t, authctx.Principal{}, "GET", "/api/v1/payment-settings", nil, 401, nil)
}

func TestIntegrationLifecycle(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	if err := e.plans.EnsureDefaultPlans(ctx); err != nil {
		t.Fatal(err)
	}
	all, err := e.plans.ListAll(ctx)
	if err != nil || len(all) == 0 {
		t.Fatal(err)
	}
	plan := all[0]

	grant := func(u authctx.Principal) *domain.Subscription {
		s, err := e.subs.Grant(ctx, u.UserID, plan.ID, 0)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	setPeriod := func(id, status string, endsIn, graceIn time.Duration) {
		if _, err := e.pool.Exec(ctx, `UPDATE subscriptions SET status = $2::subscription_status, ends_at = now() + $3::interval,
			grace_ends_at = now() + $4::interval WHERE id = $1`, id, status,
			fmt.Sprintf("%d seconds", int(endsIn.Seconds())), fmt.Sprintf("%d seconds", int(graceIn.Seconds()))); err != nil {
			t.Fatal(err)
		}
	}
	statusOf := func(id string) string {
		var s string
		_ = e.pool.QueryRow(ctx, `SELECT status::text FROM subscriptions WHERE id = $1`, id).Scan(&s)
		return s
	}
	kinds := func(id string) []string {
		rows, _ := e.pool.Query(ctx, `SELECT kind FROM notifications WHERE subscription_id = $1 ORDER BY kind`, id)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var k string
			_ = rows.Scan(&k)
			out = append(out, k)
		}
		return out
	}

	uRem := e.user(t, authctx.RoleCustomer)
	uGrace := e.user(t, authctx.RoleCustomer)
	uExp := e.user(t, authctx.RoleCustomer)
	uOrder := e.user(t, authctx.RoleCustomer)
	day := 24 * time.Hour

	sRem := grant(uRem)
	setPeriod(sRem.ID, "active", 2*day+time.Hour, 9*day)
	sGrace := grant(uGrace)
	setPeriod(sGrace.ID, "active", -time.Hour, 6*day)
	sExp := grant(uExp)
	setPeriod(sExp.ID, "grace", -8*day, -time.Hour)

	o, err := e.orders.Create(ctx, uOrder, plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.pool.Exec(ctx, `UPDATE orders SET expires_at = now() - interval '1 minute' WHERE id = $1`, o.ID); err != nil {
		t.Fatal(err)
	}

	for range 2 {
		if err := e.lifecycle.RunOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got := kinds(sRem.ID); len(got) != 1 || got[0] != domain.NotifyReminder3d {
		t.Fatalf("reminder: %v", got)
	}
	if statusOf(sGrace.ID) != "grace" || fmt.Sprint(kinds(sGrace.ID)) != "[grace]" {
		t.Fatalf("grace: %s %v", statusOf(sGrace.ID), kinds(sGrace.ID))
	}
	if statusOf(sExp.ID) != "expired" || fmt.Sprint(kinds(sExp.ID)) != "[expired]" || e.inv.count(&e.inv.suspended, uExp.UserID) != 1 {
		t.Fatalf("expired: %s %v %v", statusOf(sExp.ID), kinds(sExp.ID), e.inv.suspended)
	}
	if e.inv.count(&e.inv.suspended, uGrace.UserID) != 0 {
		t.Fatal("grace user must not be suspended")
	}
	got, err := e.orders.Get(ctx, uOrder, o.ID)
	if err != nil || got.Status != domain.OrderExpired {
		t.Fatalf("order expiry: %v %+v", err, got)
	}
	if e.inv.purged < 2 {
		t.Fatal("purge not called")
	}

	// Perpanjang user yang expired → langganan baru + restore; user grace → aktif lagi, pengingat periode baru bisa terkirim.
	sNew := grant(uExp)
	if sNew.ID == sExp.ID || sNew.Status != "active" || e.inv.count(&e.inv.restored, uExp.UserID) != 2 {
		t.Fatalf("re-grant expired: %+v", sNew)
	}
	sBack := grant(uGrace)
	if sBack.ID != sGrace.ID || sBack.Status != "active" || sBack.EndsAt.Before(time.Now().Add(29*day)) {
		t.Fatalf("re-grant grace: %+v", sBack)
	}
	setPeriod(sRem.ID, "active", 5*day, 12*day) // periode lain → ref berbeda
	if err := e.lifecycle.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if got := kinds(sRem.ID); fmt.Sprint(got) != "[reminder_3d reminder_7d]" {
		t.Fatalf("reminder new period: %v", got)
	}

	// Advisory lock: bila dipegang koneksi lain, RunOnce dilewati tanpa error.
	release, ok, err := repository.NewLocker(e.pool).TryLock(ctx, "billing:lifecycle")
	if err != nil || !ok {
		t.Fatalf("trylock: %v %v", ok, err)
	}
	o2, err := e.orders.Create(ctx, e.user(t, authctx.RoleCustomer), plan.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = e.pool.Exec(ctx, `UPDATE orders SET expires_at = now() - interval '1 minute' WHERE id = $1`, o2.ID)
	if err := e.lifecycle.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var st string
	_ = e.pool.QueryRow(ctx, `SELECT status::text FROM orders WHERE id = $1`, o2.ID).Scan(&st)
	if st != "awaiting_payment" {
		t.Fatalf("lifecycle ran despite lock: %s", st)
	}
	release()
	if err := e.lifecycle.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	_ = e.pool.QueryRow(ctx, `SELECT status::text FROM orders WHERE id = $1`, o2.ID).Scan(&st)
	if st != "expired" {
		t.Fatalf("after release: %s", st)
	}
}

// pgAudit = recorder audit minimal untuk test (menulis ke audit_logs) tanpa bergantung ke service audit.
type pgAudit struct{ pool *pgxpool.Pool }

func (a pgAudit) Record(ctx context.Context, actorID, action, target string, meta map[string]any) {
	var actor *string
	if actorID != "" {
		actor = &actorID
	}
	_, _ = a.pool.Exec(ctx, `INSERT INTO audit_logs (actor_id, action, target, meta) VALUES ($1, $2, $3, $4)`, actor, action, target, meta)
}
