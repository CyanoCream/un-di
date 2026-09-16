package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"undangan/kernel/httpx"
	"undangan/kernel/notify"
	"undangan/services/notify/service"
	"undangan/services/notify/telegram"
)

type fakeSvc struct {
	secret  string
	updates chan telegram.Update
}

func (f *fakeSvc) Notify(context.Context, notify.Notification) {}
func (f *fakeSvc) HandleUpdate(_ context.Context, u telegram.Update) {
	select {
	case f.updates <- u:
	default:
	}
}
func (f *fakeSvc) Start(context.Context)              {}
func (f *fakeSvc) SetupWebhook(context.Context) error { return nil }
func (f *fakeSvc) WebhookSecret() string              { return f.secret }
func (f *fakeSvc) Allowed(int64) bool                 { return true }

var _ service.Service = (*fakeSvc)(nil)

func newServer(secret string) (*httptest.Server, *fakeSvc) {
	svc := &fakeSvc{secret: secret, updates: make(chan telegram.Update, 1)}
	mux := http.NewServeMux()
	New(svc).Register(httpx.NewRouter(mux))
	return httptest.NewServer(mux), svc
}

func post(t *testing.T, url, secret, body string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+"/api/v1/telegram/webhook", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if secret != "" {
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	return res.StatusCode
}

func TestWebhookButuhSecretYangBenar(t *testing.T) {
	srv, svc := newServer("s3cret")
	defer srv.Close()

	if got := post(t, srv.URL, "", `{"update_id":1}`); got != http.StatusUnauthorized {
		t.Errorf("tanpa secret = %d", got)
	}
	if got := post(t, srv.URL, "salah", `{"update_id":1}`); got != http.StatusUnauthorized {
		t.Errorf("secret salah = %d", got)
	}
	if len(svc.updates) != 0 {
		t.Fatal("update tidak boleh diproses tanpa secret yang benar")
	}

	if got := post(t, srv.URL, "s3cret", `{"update_id":7,"message":{"text":"/pending","chat":{"id":111}}}`); got != http.StatusOK {
		t.Fatalf("secret benar = %d", got)
	}
	select {
	case u := <-svc.updates:
		if u.UpdateID != 7 || u.Message == nil || u.Message.Text != "/pending" {
			t.Errorf("update = %+v", u)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("update tidak diproses")
	}
}

func TestWebhookBodyRusakTidakDiulang(t *testing.T) {
	srv, _ := newServer("s3cret")
	defer srv.Close()
	if got := post(t, srv.URL, "s3cret", `{bukan json`); got != http.StatusBadRequest {
		t.Errorf("body rusak = %d", got)
	}
}

// Secret kosong = webhook mati total (mode polling), bukan terbuka untuk siapa saja.
func TestWebhookTanpaSecretDitolak(t *testing.T) {
	srv, _ := newServer("")
	defer srv.Close()
	if got := post(t, srv.URL, "", `{"update_id":1}`); got != http.StatusUnauthorized {
		t.Errorf("secret kosong = %d", got)
	}
}
