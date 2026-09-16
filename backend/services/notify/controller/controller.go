// Package controller (notify) = endpoint webhook Telegram.
package controller

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"undangan/kernel/httpx"
	"undangan/services/notify/service"
	"undangan/services/notify/telegram"
)

type Controller struct{ svc service.Service }

func New(svc service.Service) *Controller { return &Controller{svc: svc} }

// WebhookPath = path webhook (dikecualikan dari CSRF guard saat wiring).
const WebhookPath = "/api/v1/telegram/webhook"

func (c *Controller) Register(rt *httpx.Router) {
	// Publik: dipanggil server Telegram. Keamanan = header secret yang hanya diketahui Telegram & aplikasi.
	rt.Public("POST "+WebhookPath, c.webhook)
}

func (c *Controller) webhook(w http.ResponseWriter, r *http.Request) error {
	if secret := c.svc.WebhookSecret(); secret == "" || r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != secret {
		w.WriteHeader(http.StatusUnauthorized)
		return nil
	}
	var u telegram.Update
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&u); err != nil {
		w.WriteHeader(http.StatusBadRequest) // update rusak: jangan minta Telegram mengirim ulang
		return nil
	}
	// Balas cepat; pemrosesan (panggil API Telegram, DB) tidak boleh menahan request Telegram.
	go c.svc.HandleUpdate(context.WithoutCancel(r.Context()), u)
	w.WriteHeader(http.StatusOK)
	return nil
}
