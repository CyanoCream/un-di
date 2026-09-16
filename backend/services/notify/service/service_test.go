package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"undangan/kernel/authctx"
	"undangan/kernel/notify"
	"undangan/services/notify/domain"
	"undangan/services/notify/telegram"
)

// ---- Bot API tiruan ----

type call struct {
	method string
	body   map[string]any
}

type fakeAPI struct {
	*httptest.Server
	mu    sync.Mutex
	calls []call
}

func newFakeAPI(t *testing.T) *fakeAPI {
	t.Helper()
	f := &fakeAPI{}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		f.mu.Lock()
		f.calls = append(f.calls, call{method: parts[len(parts)-1], body: payload})
		f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":99}}`))
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeAPI) of(method string) []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []call
	for _, c := range f.calls {
		if c.method == method {
			out = append(out, c)
		}
	}
	return out
}

func (f *fakeAPI) waitFor(t *testing.T, method string, n int) []call {
	t.Helper()
	for range 100 {
		if got := f.of(method); len(got) >= n {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timeout menunggu %d panggilan %s (didapat %d)", n, method, len(f.of(method)))
	return nil
}

// ---- fake ports ----

type fakeOrders struct {
	approved, rejected []string
	reason, actorName  string
	err                error
	pending            []domain.PendingOrder
}

func (f *fakeOrders) Approve(_ context.Context, actor authctx.Principal, id string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.approved = append(f.approved, id)
	f.actorName = actor.Name
	return "ORD-1", nil
}

func (f *fakeOrders) Reject(_ context.Context, actor authctx.Principal, id, reason string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.rejected, f.reason, f.actorName = append(f.rejected, id), reason, actor.Name
	return "ORD-2", nil
}

func (f *fakeOrders) Pending(context.Context) ([]domain.PendingOrder, error) { return f.pending, nil }

type fakeAdmin struct{ err error }

func (f fakeAdmin) SystemAdmin(context.Context) (authctx.Principal, error) {
	if f.err != nil {
		return authctx.Principal{}, f.err
	}
	return authctx.Principal{UserID: "admin-1", Role: authctx.RoleSuperAdmin, Name: "Super Admin"}, nil
}

func newSvc(t *testing.T, api *fakeAPI, orders *fakeOrders, admin domain.AdminResolver) Service {
	t.Helper()
	return New(Config{
		BotToken: "T0KEN", ChatIDs: []int64{111, 222}, WebhookSecret: "s3cret", APIBaseURL: api.URL,
		AdminURL: "https://admin.undangin.id", AppName: "Undangin",
	}, orders, admin, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func callback(data string, chatID int64) telegram.Update {
	return telegram.Update{CallbackQuery: &telegram.CallbackQuery{
		ID: "cb1", Data: data, From: &telegram.User{ID: 5, Username: "anang"},
		Message: &telegram.IncomingMsg{MessageID: 42, Chat: &telegram.Chat{ID: chatID}},
	}}
}

// ---- test ----

func TestNotifyKirimKeSemuaChatDenganTombol(t *testing.T) {
	api := newFakeAPI(t)
	svc := newSvc(t, api, &fakeOrders{}, fakeAdmin{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)

	svc.Notify(ctx, notify.Notification{
		Kind: notify.KindOrderProof, Title: "Bukti transfer", Lines: []string{"Order: ORD-1", "Nominal: Rp100.390"},
		Link: "https://admin.undangin.id/orders?id=x", Actions: []notify.Action{{Label: "✅ Setujui", Data: "order:approve:x"}},
	})

	sends := api.waitFor(t, "sendMessage", 2)
	chats := map[float64]bool{}
	for _, c := range sends {
		chats[c.body["chat_id"].(float64)] = true
		text := c.body["text"].(string)
		if !strings.Contains(text, "Bukti transfer") || !strings.Contains(text, "Rp100.390") {
			t.Errorf("isi pesan: %s", text)
		}
		if c.body["reply_markup"] == nil {
			t.Error("tombol tidak terkirim")
		}
	}
	if !chats[111] || !chats[222] {
		t.Errorf("chat tujuan = %v", chats)
	}
}

func TestNotifyEscapeHTML(t *testing.T) {
	api := newFakeAPI(t)
	svc := newSvc(t, api, &fakeOrders{}, fakeAdmin{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.Start(ctx)

	svc.Notify(ctx, notify.Notification{Kind: notify.KindUserRegister, Title: "Customer baru", Lines: []string{"Nama: <b>bold</b> & co"}})
	text := api.waitFor(t, "sendMessage", 1)[0].body["text"].(string)
	if strings.Contains(text, "<b>bold</b>") || !strings.Contains(text, "&lt;b&gt;bold&lt;/b&gt; &amp; co") {
		t.Errorf("teks user tidak di-escape: %s", text)
	}
}

func TestCallbackSetujui(t *testing.T) {
	api := newFakeAPI(t)
	orders := &fakeOrders{}
	svc := newSvc(t, api, orders, fakeAdmin{})

	svc.HandleUpdate(context.Background(), callback("order:approve:order-9", 111))

	if len(orders.approved) != 1 || orders.approved[0] != "order-9" {
		t.Fatalf("approve = %v", orders.approved)
	}
	if !strings.Contains(orders.actorName, "@anang") {
		t.Errorf("pelaku aksi harus tercatat: %q", orders.actorName)
	}
	if len(api.of("answerCallbackQuery")) == 0 || len(api.of("editMessageReplyMarkup")) == 0 {
		t.Error("callback harus dijawab & tombol dihapus")
	}
}

func TestCallbackDariChatAsingDitolak(t *testing.T) {
	api := newFakeAPI(t)
	orders := &fakeOrders{}
	svc := newSvc(t, api, orders, fakeAdmin{})

	svc.HandleUpdate(context.Background(), callback("order:approve:order-9", 999)) // chat tidak terdaftar

	if len(orders.approved) != 0 {
		t.Fatal("order tidak boleh disetujui dari chat asing")
	}
	if len(api.of("sendMessage")) != 0 {
		t.Error("jangan balas ke chat asing")
	}
}

func TestCallbackGagalDilaporkan(t *testing.T) {
	api := newFakeAPI(t)
	orders := &fakeOrders{err: errors.New("status order tidak sesuai")}
	svc := newSvc(t, api, orders, fakeAdmin{})

	svc.HandleUpdate(context.Background(), callback("order:approve:order-9", 111))

	msgs := api.of("sendMessage")
	if len(msgs) != 1 || !strings.Contains(msgs[0].body["text"].(string), "status order tidak sesuai") {
		t.Errorf("pesan kegagalan = %+v", msgs)
	}
}

func TestAlurTolakMintaAlasan(t *testing.T) {
	api := newFakeAPI(t)
	orders := &fakeOrders{}
	svc := newSvc(t, api, orders, fakeAdmin{})
	ctx := context.Background()

	svc.HandleUpdate(ctx, callback("order:reject:order-7", 111))
	prompt := api.of("sendMessage")
	if len(prompt) != 1 {
		t.Fatalf("prompt = %+v", prompt)
	}
	promptText := prompt[0].body["text"].(string)
	if !strings.Contains(promptText, "#id:order-7") || prompt[0].body["reply_markup"] == nil {
		t.Fatalf("prompt harus force reply & memuat id: %s", promptText)
	}
	if len(orders.rejected) != 0 {
		t.Fatal("belum boleh menolak sebelum ada alasan")
	}

	// Alasan terlalu pendek → belum ditolak.
	svc.HandleUpdate(ctx, telegram.Update{Message: &telegram.IncomingMsg{
		Chat: &telegram.Chat{ID: 111}, From: &telegram.User{Username: "anang"}, Text: "ok",
		ReplyTo: &telegram.IncomingMsg{Text: promptText},
	}})
	if len(orders.rejected) != 0 {
		t.Fatal("alasan pendek harus ditolak")
	}

	svc.HandleUpdate(ctx, telegram.Update{Message: &telegram.IncomingMsg{
		Chat: &telegram.Chat{ID: 111}, From: &telegram.User{Username: "anang"}, Text: "nominal tidak sesuai mutasi",
		ReplyTo: &telegram.IncomingMsg{Text: promptText},
	}})
	if len(orders.rejected) != 1 || orders.rejected[0] != "order-7" || orders.reason != "nominal tidak sesuai mutasi" {
		t.Fatalf("reject = %v alasan=%q", orders.rejected, orders.reason)
	}
}

func TestPerintahStartDanPending(t *testing.T) {
	api := newFakeAPI(t)
	orders := &fakeOrders{pending: []domain.PendingOrder{
		{ID: "o1", Code: "ORD-1", Customer: "Raka", PlanName: "Basic", Amount: 100390, HasProof: true, CreatedAgo: "5 menit lalu"},
	}}
	svc := newSvc(t, api, orders, fakeAdmin{})
	ctx := context.Background()

	// /start dari chat asing tetap dibalas chat id-nya (untuk konfigurasi), dengan catatan belum terdaftar.
	svc.HandleUpdate(ctx, telegram.Update{Message: &telegram.IncomingMsg{Chat: &telegram.Chat{ID: 999}, Text: "/start"}})
	first := api.of("sendMessage")[0].body["text"].(string)
	if !strings.Contains(first, "999") || !strings.Contains(first, "belum terdaftar") {
		t.Errorf("/start = %s", first)
	}

	svc.HandleUpdate(ctx, telegram.Update{Message: &telegram.IncomingMsg{Chat: &telegram.Chat{ID: 111}, Text: "/pending"}})
	last := api.of("sendMessage")
	text := last[len(last)-1].body["text"].(string)
	if !strings.Contains(text, "ORD-1") || !strings.Contains(text, "Rp100.390") || last[len(last)-1].body["reply_markup"] == nil {
		t.Errorf("/pending = %s", text)
	}
}

func TestRupiah(t *testing.T) {
	for in, want := range map[int64]string{0: "Rp0", 999: "Rp999", 100390: "Rp100.390", 1750000: "Rp1.750.000"} {
		if got := Rupiah(in); got != want {
			t.Errorf("Rupiah(%d) = %s, want %s", in, got, want)
		}
	}
}

func TestParseCallback(t *testing.T) {
	if c, ok := domain.ParseCallback("order:approve:abc"); !ok || c.Action != "approve" || c.ID != "abc" {
		t.Errorf("valid = %+v %v", c, ok)
	}
	for _, bad := range []string{"", "order:approve", "order:hapus:abc", "invoice:approve:abc", "order::abc"} {
		if _, ok := domain.ParseCallback(bad); ok {
			t.Errorf("%q seharusnya ditolak", bad)
		}
	}
}
