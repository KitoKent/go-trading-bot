package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"
	models "trading-bot/internal/domain/entity"

	"github.com/gorilla/websocket"
)

func wsSubscribe(ctx context.Context, baseURL, stream string, ch chan<- models.Candle) error {
	u := fmt.Sprintf("%s/%s", baseURL, stream)
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return fmt.Errorf("ws: %w", err)
	}

	go func() {
		defer conn.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, msg, err := conn.ReadMessage()
				if err != nil {
					slog.Warn("ws disconnect, reconnecting...", "error", err)
					wsReconnect(ctx, baseURL, stream, ch)
					return
				}
				c, err := parseKline(msg)
				if err != nil {
					continue
				}
				select {
				case ch <- *c:
				default:
				}
			}
		}
	}()
	return nil
}

func wsReconnect(ctx context.Context, baseURL, stream string, ch chan<- models.Candle) {
	time.Sleep(3 * time.Second)
	u := fmt.Sprintf("%s/%s", baseURL, stream)
	conn, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, msg, err := conn.ReadMessage()
			if err != nil {
				return
			}
			c, err := parseKline(msg)
			if err != nil {
				continue
			}
			select {
			case ch <- *c:
			default:
			}
		}
	}
}

type wsKline struct {
	Symbol string `json:"s"`
	Kline  struct {
		Start  int64  `json:"t"`
		End    int64  `json:"T"`
		Open   string `json:"o"`
		High   string `json:"h"`
		Low    string `json:"l"`
		Close  string `json:"c"`
		Volume string `json:"v"`
	} `json:"k"`
}

func parseKline(msg []byte) (*models.Candle, error) {
	var k wsKline
	if err := json.Unmarshal(msg, &k); err != nil {
		return nil, err
	}
	open, _ := strconv.ParseFloat(k.Kline.Open, 64)
	high, _ := strconv.ParseFloat(k.Kline.High, 64)
	low, _ := strconv.ParseFloat(k.Kline.Low, 64)
	close_, _ := strconv.ParseFloat(k.Kline.Close, 64)
	vol, _ := strconv.ParseFloat(k.Kline.Volume, 64)

	return &models.Candle{
		Symbol:    k.Symbol,
		OpenTime:  time.UnixMilli(k.Kline.Start),
		CloseTime: time.UnixMilli(k.Kline.End),
		Open:      open, High: high, Low: low, Close: close_, Volume: vol,
	}, nil
}
