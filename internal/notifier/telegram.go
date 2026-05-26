package notifier

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Telegram struct {
	token  string
	chatID string
	http   *http.Client
	log    *slog.Logger
}

func NewTelegram(token, chatID string) *Telegram {
	return &Telegram{
		token:  token,
		chatID: chatID,
		http:   &http.Client{Timeout: 10 * time.Second},
		log:    slog.With("notifier", "telegram"),
	}
}

func (t *Telegram) Send(ctx context.Context, msg string) error {
	if t.token == "" || t.chatID == "" {
		return nil
	}
	api := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.token)
	data := url.Values{"chat_id": {t.chatID}, "text": {msg}, "parse_mode": {"HTML"}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, api, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram %d: %s", resp.StatusCode, body)
	}
	return nil
}
