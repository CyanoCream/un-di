// Package service (notify) = pengirim notifikasi Telegram + handler perintah/tombol bot.
package service

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"undangan/kernel/notify"
	"undangan/services/notify/domain"
	"undangan/services/notify/telegram"
)

type Config struct {
	BotToken      string
	ChatIDs       []int64 // penerima notifikasi & satu-satunya yang boleh memakai bot
	WebhookSecret string
	WebhookURL    string // kosong = mode polling (dev)
	APIBaseURL    string // kosong = api.telegram.org (diisi saat test)
	AdminURL      string // mis. https://admin.undangin.id (untuk tautan di pesan)
	AppName       string
}

func (c Config) Enabled() bool { return c.BotToken != "" && len(c.ChatIDs) > 0 }

type Service interface {
	notify.Notifier
	// HandleUpdate memproses satu update dari webhook atau polling.
	HandleUpdate(ctx context.Context, u telegram.Update)
	// Start menjalankan worker pengirim + (bila mode polling) loop getUpdates. Berhenti saat ctx selesai.
	Start(ctx context.Context)
	// SetupWebhook mendaftarkan/menghapus webhook sesuai konfigurasi.
	SetupWebhook(ctx context.Context) error
	WebhookSecret() string
	Allowed(chatID int64) bool
}

type service struct {
	cfg    Config
	client *telegram.Client
	orders domain.OrderOps
	admin  domain.AdminResolver
	log    *slog.Logger
	queue  chan notify.Notification
}

func New(cfg Config, orders domain.OrderOps, admin domain.AdminResolver, log *slog.Logger) Service {
	return &service{
		cfg: cfg, client: telegram.New(cfg.BotToken, cfg.APIBaseURL), orders: orders, admin: admin, log: log,
		queue: make(chan notify.Notification, 100),
	}
}

func (s *service) WebhookSecret() string { return s.cfg.WebhookSecret }

func (s *service) Allowed(chatID int64) bool {
	for _, id := range s.cfg.ChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}

// Notify tidak pernah memblokir pemanggil: antrean penuh → notifikasi dilewati (alur utama tetap jalan).
func (s *service) Notify(_ context.Context, n notify.Notification) {
	select {
	case s.queue <- n:
	default:
		s.log.Warn("antrean notifikasi penuh, notifikasi dilewati", "kind", n.Kind)
	}
}

func (s *service) Start(ctx context.Context) {
	go s.sendLoop(ctx)
	if s.cfg.WebhookURL == "" {
		go s.pollLoop(ctx)
	}
}

func (s *service) sendLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case n := <-s.queue:
			sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
			text, markup := render(n)
			for _, chat := range s.cfg.ChatIDs {
				if _, err := s.client.SendMessage(sendCtx, telegram.Message{ChatID: chat, Text: text, ReplyMarkup: markup}); err != nil {
					s.log.Error("kirim notifikasi telegram", "chat_id", chat, "kind", n.Kind, "err", err)
				}
			}
			cancel()
		}
	}
}

func (s *service) pollLoop(ctx context.Context) {
	var offset int64
	s.log.Info("telegram: mode polling (tanpa webhook)")
	for {
		if ctx.Err() != nil {
			return
		}
		updates, err := s.client.GetUpdates(ctx, offset, 30)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.log.Error("telegram getUpdates", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
			}
			continue
		}
		for _, u := range updates {
			offset = u.UpdateID + 1
			s.HandleUpdate(ctx, u)
		}
	}
}

func (s *service) SetupWebhook(ctx context.Context) error {
	if s.cfg.WebhookURL == "" {
		return s.client.DeleteWebhook(ctx)
	}
	return s.client.SetWebhook(ctx, s.cfg.WebhookURL, s.cfg.WebhookSecret)
}

// ---- format pesan ----

func render(n notify.Notification) (string, *telegram.ReplyMarkup) {
	var b strings.Builder
	fmt.Fprintf(&b, "%s <b>%s</b>", emoji(n.Kind), html.EscapeString(n.Title))
	for _, l := range n.Lines {
		b.WriteString("\n" + html.EscapeString(l))
	}
	if n.Link != "" {
		fmt.Fprintf(&b, "\n\n<a href=\"%s\">Buka di portal admin</a>", html.EscapeString(n.Link))
	}
	var markup *telegram.ReplyMarkup
	if len(n.Actions) > 0 {
		row := make([]telegram.InlineButton, 0, len(n.Actions))
		for _, a := range n.Actions {
			row = append(row, telegram.InlineButton{Text: a.Label, CallbackData: a.Data})
		}
		markup = &telegram.ReplyMarkup{InlineKeyboard: [][]telegram.InlineButton{row}}
	}
	return b.String(), markup
}

func emoji(kind string) string {
	switch kind {
	case notify.KindOrderCreated:
		return "🧾"
	case notify.KindOrderProof:
		return "💳"
	case notify.KindOrderApproved:
		return "✅"
	case notify.KindOrderRejected:
		return "⛔"
	case notify.KindUserRegister:
		return "🙋"
	}
	return "🔔"
}

// ---- handler bot ----

func (s *service) HandleUpdate(ctx context.Context, u telegram.Update) {
	switch {
	case u.CallbackQuery != nil:
		s.handleCallback(ctx, u.CallbackQuery)
	case u.Message != nil:
		s.handleMessage(ctx, u.Message)
	}
}

func (s *service) reply(ctx context.Context, chatID int64, text string, markup *telegram.ReplyMarkup) {
	if _, err := s.client.SendMessage(ctx, telegram.Message{ChatID: chatID, Text: text, ReplyMarkup: markup}); err != nil {
		s.log.Error("balas telegram", "chat_id", chatID, "err", err)
	}
}

func (s *service) handleMessage(ctx context.Context, m *telegram.IncomingMsg) {
	if m.Chat == nil {
		return
	}
	text := strings.TrimSpace(m.Text)
	// /start & /id boleh dari siapa pun: dipakai untuk mengetahui chat id saat konfigurasi.
	if strings.HasPrefix(text, "/start") || strings.HasPrefix(text, "/id") {
		msg := fmt.Sprintf("Chat ID Anda: <code>%d</code>", m.Chat.ID)
		if !s.Allowed(m.Chat.ID) {
			msg += "\n\nChat ini belum terdaftar. Tambahkan id di TELEGRAM_CHAT_IDS lalu restart aplikasi."
		} else {
			msg += "\n\nPerintah: /pending — daftar order menunggu konfirmasi."
		}
		s.reply(ctx, m.Chat.ID, msg, nil)
		return
	}
	if !s.Allowed(m.Chat.ID) {
		return // abaikan chat lain tanpa balasan
	}
	if strings.HasPrefix(text, "/pending") {
		s.sendPending(ctx, m.Chat.ID)
		return
	}
	// Balasan alasan penolakan (reply ke pesan prompt bot).
	if m.ReplyTo != nil {
		if id := domain.OrderIDFromPrompt(m.ReplyTo.Text); id != "" {
			s.rejectOrder(ctx, m.Chat.ID, id, text, m.From)
		}
	}
}

func (s *service) sendPending(ctx context.Context, chatID int64) {
	items, err := s.orders.Pending(ctx)
	if err != nil {
		s.reply(ctx, chatID, "Gagal mengambil data order: "+html.EscapeString(err.Error()), nil)
		return
	}
	if len(items) == 0 {
		s.reply(ctx, chatID, "Tidak ada order yang menunggu konfirmasi. 👌", nil)
		return
	}
	for _, o := range items {
		lines := []string{
			"Customer: " + o.Customer,
			"Paket: " + o.PlanName,
			"Nominal: " + Rupiah(o.Amount),
			"Dibuat: " + o.CreatedAgo,
		}
		if !o.HasProof {
			lines = append(lines, "Bukti transfer: belum diunggah")
		}
		text, markup := render(notify.Notification{
			Kind: notify.KindOrderProof, Title: "Order " + o.Code, Lines: lines,
			Link:    s.orderLink(o.ID),
			Actions: orderActions(o.ID),
		})
		s.reply(ctx, chatID, text, markup)
	}
}

func (s *service) handleCallback(ctx context.Context, q *telegram.CallbackQuery) {
	if q.Message == nil || q.Message.Chat == nil || !s.Allowed(q.Message.Chat.ID) {
		_ = s.client.AnswerCallback(ctx, q.ID, "Tidak diizinkan")
		return
	}
	cb, ok := domain.ParseCallback(q.Data)
	if !ok {
		_ = s.client.AnswerCallback(ctx, q.ID, "Aksi tidak dikenal")
		return
	}
	chatID := q.Message.Chat.ID

	if cb.Action == "reject" {
		_ = s.client.AnswerCallback(ctx, q.ID, "Tulis alasan penolakan")
		s.reply(ctx, chatID, domain.RejectPrompt+"#id:"+cb.ID+"\n(balas pesan ini dengan alasannya)",
			&telegram.ReplyMarkup{ForceReply: true, Selective: true})
		return
	}

	actor, err := s.admin.SystemAdmin(ctx)
	if err != nil {
		_ = s.client.AnswerCallback(ctx, q.ID, "Admin sistem tidak ditemukan")
		return
	}
	actor.Name = "Telegram " + q.From.Display()
	code, err := s.orders.Approve(ctx, actor, cb.ID)
	if err != nil {
		_ = s.client.AnswerCallback(ctx, q.ID, trim(err.Error(), 190))
		s.reply(ctx, chatID, "⚠️ Gagal menyetujui order: "+html.EscapeString(err.Error()), nil)
		return
	}
	_ = s.client.AnswerCallback(ctx, q.ID, "Order disetujui")
	_ = s.client.EditReplyMarkup(ctx, chatID, q.Message.MessageID, nil) // tombol dihapus agar tidak diklik dua kali
	s.reply(ctx, chatID, "✅ Order <b>"+html.EscapeString(code)+"</b> disetujui oleh "+html.EscapeString(q.From.Display())+
		". Langganan customer sudah aktif/diperpanjang.", nil)
}

func (s *service) rejectOrder(ctx context.Context, chatID int64, orderID, reason string, from *telegram.User) {
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) < 3 {
		s.reply(ctx, chatID, "Alasan terlalu pendek. Balas lagi pesan sebelumnya dengan alasan yang jelas.", nil)
		return
	}
	actor, err := s.admin.SystemAdmin(ctx)
	if err != nil {
		s.reply(ctx, chatID, "Admin sistem tidak ditemukan.", nil)
		return
	}
	actor.Name = "Telegram " + from.Display()
	code, err := s.orders.Reject(ctx, actor, orderID, reason)
	if err != nil {
		s.reply(ctx, chatID, "⚠️ Gagal menolak order: "+html.EscapeString(err.Error()), nil)
		return
	}
	s.reply(ctx, chatID, "⛔ Order <b>"+html.EscapeString(code)+"</b> ditolak. Alasan: "+html.EscapeString(reason), nil)
}

func (s *service) orderLink(id string) string {
	if s.cfg.AdminURL == "" {
		return ""
	}
	return strings.TrimRight(s.cfg.AdminURL, "/") + "/orders?id=" + id
}

func orderActions(id string) []notify.Action {
	return []notify.Action{
		{Label: "✅ Setujui", Data: domain.OrderAction("approve", id)},
		{Label: "⛔ Tolak", Data: domain.OrderAction("reject", id)},
	}
}

// Rupiah: 100390 → "Rp100.390".
func Rupiah(n int64) string {
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	b.WriteString("Rp")
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return b.String()
}

func trim(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
