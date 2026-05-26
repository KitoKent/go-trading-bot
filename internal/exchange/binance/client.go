package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	models "trading-bot/internal/domain/entity"
)

type Client struct {
	apiKey, secretKey string
	base, ws          string
	http              *http.Client
	log               *slog.Logger
}

func New(apiKey, secretKey string, testnet bool) *Client {
	base := "https://api.binance.com"
	ws := "wss://stream.binance.com:9443/ws"
	if testnet {
		base = "https://testnet.binance.vision"
		ws = "wss://testnet.binance.vision/ws"
	}
	return &Client{
		apiKey:    apiKey,
		secretKey: secretKey,
		base:      base,
		ws:        ws,
		http:      &http.Client{Timeout: 15 * time.Second},
		log:       slog.With("exchange", "binance"),
	}
}

func (c *Client) Name() string { return "binance" }

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.get(ctx, "/api/v3/ping", nil)
	return err
}

func (c *Client) GetCandles(ctx context.Context, symbol, interval string, limit int) ([]models.Candle, error) {
	p := url.Values{}
	p.Set("symbol", strings.ToUpper(symbol))
	p.Set("interval", interval)
	if limit > 0 {
		p.Set("limit", strconv.Itoa(limit))
	}

	body, err := c.get(ctx, "/api/v3/klines", p)
	if err != nil {
		return nil, fmt.Errorf("candles: %w", err)
	}

	var raw [][]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("candles decode: %w", err)
	}

	out := make([]models.Candle, 0, len(raw))
	for _, k := range raw {
		out = append(out, models.Candle{
			Symbol:    symbol,
			OpenTime:  time.UnixMilli(int64(unsafeFloat(k[0]))),
			Open:      unsafeFloat(k[1]),
			High:      unsafeFloat(k[2]),
			Low:       unsafeFloat(k[3]),
			Close:     unsafeFloat(k[4]),
			Volume:    unsafeFloat(k[5]),
			CloseTime: time.UnixMilli(int64(unsafeFloat(k[6]))),
		})
	}
	return out, nil
}

func (c *Client) GetTicker(ctx context.Context, symbol string) (float64, error) {
	body, err := c.get(ctx, "/api/v3/ticker/price", url.Values{"symbol": {strings.ToUpper(symbol)}})
	if err != nil {
		return 0, err
	}
	var r struct {
		Price string `json:"price"`
	}
	json.Unmarshal(body, &r)
	return strconv.ParseFloat(r.Price, 64)
}

func (c *Client) Buy(ctx context.Context, symbol string, quantity float64) (*models.Order, error) {
	return c.order(ctx, symbol, "BUY", quantity)
}

func (c *Client) Sell(ctx context.Context, symbol string, quantity float64) (*models.Order, error) {
	return c.order(ctx, symbol, "SELL", quantity)
}

func (c *Client) order(ctx context.Context, symbol, side string, quantity float64) (*models.Order, error) {
	p := url.Values{}
	p.Set("symbol", strings.ToUpper(symbol))
	p.Set("side", side)
	p.Set("type", "MARKET")
	p.Set("quantity", strconv.FormatFloat(quantity, 'f', -1, 64))
	p.Set("newOrderRespType", "FULL")

	body, err := c.signAndPost(ctx, "/api/v3/order", p)
	if err != nil {
		return nil, fmt.Errorf("order: %w", err)
	}

	var r struct {
		OrderID      int64  `json:"orderId"`
		Status       string `json:"status"`
		Price        string `json:"price"`
		OrigQty      string `json:"origQty"`
		ExecQty      string `json:"executedQty"`
		CumQuote     string `json:"cummulativeQuoteQty"`
		TransactTime int64  `json:"transactTime"`
	}
	json.Unmarshal(body, &r)

	price, _ := strconv.ParseFloat(r.Price, 64)
	qty, _ := strconv.ParseFloat(r.OrigQty, 64)
	filled, _ := strconv.ParseFloat(r.ExecQty, 64)

	if r.Status == "FILLED" && price == 0 {
		cum, _ := strconv.ParseFloat(r.CumQuote, 64)
		if filled > 0 {
			price = cum / filled
		}
	}

	return &models.Order{
		ID:        fmt.Sprintf("%d", r.OrderID),
		Symbol:    symbol,
		Side:      parseSide(side),
		Type:      models.MarketOrder,
		Price:     price,
		Quantity:  qty,
		FilledQty: filled,
		Status:    strings.ToLower(r.Status),
		CreatedAt: time.UnixMilli(r.TransactTime),
	}, nil
}

func (c *Client) SubscribeCandles(ctx context.Context, symbol, interval string, ch chan<- models.Candle) error {
	stream := fmt.Sprintf("%s@kline_%s", strings.ToLower(symbol), interval)
	return wsSubscribe(ctx, c.ws, stream, ch)
}

func (c *Client) get(ctx context.Context, path string, p url.Values) ([]byte, error) {
	u, _ := url.Parse(c.base + path)
	u.RawQuery = p.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	return c.do(req)
}

func (c *Client) signAndPost(ctx context.Context, path string, p url.Values) ([]byte, error) {
	u, _ := url.Parse(c.base + path)
	p.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
	p.Set("signature", sign(c.secretKey, p.Encode()))
	u.RawQuery = p.Encode()

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), nil)
	req.Header.Set("X-MBX-APIKEY", c.apiKey)
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("binance %s: %s", req.URL.Path, string(body))
	}
	return body, nil
}

func sign(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func parseSide(s string) models.OrderSide {
	if s == "BUY" {
		return models.BuyOrder
	}
	return models.SellOrder
}

func unsafeFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}
