// Package telegram = client Bot API minimal (tanpa dependency pihak ketiga).
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const DefaultAPI = "https://api.telegram.org"

type Client struct {
	token string
	api   string
	http  *http.Client
}

func New(token, api string) *Client {
	if api == "" {
		api = DefaultAPI
	}
	return &Client{token: token, api: strings.TrimRight(api, "/"), http: &http.Client{Timeout: 30 * time.Second}}
}

// InlineButton & InlineKeyboard = tombol di bawah pesan.
type InlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

type Message struct {
	ChatID      any              `json:"chat_id"`
	Text        string           `json:"text"`
	ParseMode   string           `json:"parse_mode,omitempty"`
	ReplyMarkup *ReplyMarkup     `json:"reply_markup,omitempty"`
	LinkPreview *linkPreviewOpts `json:"link_preview_options,omitempty"`
}

type ReplyMarkup struct {
	InlineKeyboard [][]InlineButton `json:"inline_keyboard,omitempty"`
	ForceReply     bool             `json:"force_reply,omitempty"`
	Selective      bool             `json:"selective,omitempty"`
}

type linkPreviewOpts struct {
	IsDisabled bool `json:"is_disabled"`
}

func (c *Client) call(ctx context.Context, method string, payload, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.api+"/bot"+c.token+"/"+method, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))

	var envelope struct {
		OK          bool            `json:"ok"`
		Description string          `json:"description"`
		Result      json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return fmt.Errorf("telegram %s: respons tidak valid (%d)", method, res.StatusCode)
	}
	if !envelope.OK {
		return fmt.Errorf("telegram %s: %s", method, envelope.Description)
	}
	if out != nil && len(envelope.Result) > 0 {
		return json.Unmarshal(envelope.Result, out)
	}
	return nil
}

func (c *Client) SendMessage(ctx context.Context, m Message) (messageID int64, err error) {
	if m.ParseMode == "" {
		m.ParseMode = "HTML"
	}
	m.LinkPreview = &linkPreviewOpts{IsDisabled: true}
	var out struct {
		MessageID int64 `json:"message_id"`
	}
	err = c.call(ctx, "sendMessage", m, &out)
	return out.MessageID, err
}

func (c *Client) AnswerCallback(ctx context.Context, callbackID, text string) error {
	return c.call(ctx, "answerCallbackQuery", map[string]any{"callback_query_id": callbackID, "text": text}, nil)
}

// EditReplyMarkup dipakai untuk menghapus tombol setelah order diproses.
func (c *Client) EditReplyMarkup(ctx context.Context, chatID any, messageID int64, markup *ReplyMarkup) error {
	return c.call(ctx, "editMessageReplyMarkup", map[string]any{"chat_id": chatID, "message_id": messageID, "reply_markup": markup}, nil)
}

func (c *Client) SetWebhook(ctx context.Context, url, secret string) error {
	return c.call(ctx, "setWebhook", map[string]any{
		"url": url, "secret_token": secret, "allowed_updates": []string{"message", "callback_query"}, "drop_pending_updates": false,
	}, nil)
}

func (c *Client) DeleteWebhook(ctx context.Context) error {
	return c.call(ctx, "deleteWebhook", map[string]any{"drop_pending_updates": false}, nil)
}

func (c *Client) GetMe(ctx context.Context) (username string, err error) {
	var out struct {
		Username string `json:"username"`
	}
	err = c.call(ctx, "getMe", map[string]any{}, &out)
	return out.Username, err
}

// GetUpdates = long polling (dipakai saat dev, tanpa URL publik).
func (c *Client) GetUpdates(ctx context.Context, offset int64, timeoutSec int) ([]Update, error) {
	var out []Update
	err := c.call(ctx, "getUpdates", map[string]any{
		"offset": offset, "timeout": timeoutSec, "allowed_updates": []string{"message", "callback_query"},
	}, &out)
	return out, err
}

// ---- tipe update ----

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *IncomingMsg   `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type IncomingMsg struct {
	MessageID int64        `json:"message_id"`
	From      *User        `json:"from"`
	Chat      *Chat        `json:"chat"`
	Text      string       `json:"text"`
	ReplyTo   *IncomingMsg `json:"reply_to_message"`
}

type CallbackQuery struct {
	ID      string       `json:"id"`
	From    *User        `json:"from"`
	Data    string       `json:"data"`
	Message *IncomingMsg `json:"message"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID int64 `json:"id"`
}

func (u *User) Display() string {
	if u == nil {
		return "seseorang"
	}
	if u.Username != "" {
		return "@" + u.Username
	}
	return u.FirstName
}
