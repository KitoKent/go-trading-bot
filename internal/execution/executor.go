package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	models "trading-bot/internal/domain/entity"
	"trading-bot/internal/exchange"
)

// TradeLog пишет сделки в JSON-файл (одна строка = одна сделка).
// Это проще и надёжнее, чем база данных.
type TradeLog struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

func NewTradeLog(path string) (*TradeLog, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &TradeLog{file: f, enc: json.NewEncoder(f)}, nil
}

func (l *TradeLog) Write(symbol, side string, price, qty float64, reason, orderID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enc.Encode(models.TradeRecord{
		Time:     time.Now(),
		Symbol:   symbol,
		Side:     side,
		Price:    price,
		Quantity: qty,
		Reason:   reason,
		OrderID:  orderID,
	})
}

func (l *TradeLog) Close() error {
	return l.file.Close()
}

type Notifier interface {
	Send(ctx context.Context, msg string) error
}

type Executor struct {
	exchange exchange.Client
	tradeLog *TradeLog
	notifier Notifier
	log      *slog.Logger
}

func New(cli exchange.Client, log *TradeLog, n Notifier) *Executor {
	return &Executor{
		exchange: cli,
		tradeLog: log,
		notifier: n,
		log:      slog.With("module", "executor"),
	}
}

func (e *Executor) Buy(ctx context.Context, sig *models.Signal) {
	if sig == nil || sig.Type != models.Buy {
		return
	}
	e.execute(ctx, sig, e.exchange.Buy)
}

func (e *Executor) Sell(ctx context.Context, sig *models.Signal) {
	if sig == nil || sig.Type != models.Sell {
		return
	}
	e.execute(ctx, sig, e.exchange.Sell)
}

func (e *Executor) execute(ctx context.Context, sig *models.Signal, fn func(context.Context, string, float64) (*models.Order, error)) {
	qty := sig.Quantity
	if qty <= 0 {
		qty = 0.001
	}

	order, err := fn(ctx, sig.Symbol, qty)
	if err != nil {
		e.log.Error("trade failed", "symbol", sig.Symbol, "side", sig.Type, "error", err)
		e.notify(ctx, fmt.Sprintf("Ошибка: %s %s — %s", sig.Symbol, sig.Type, err))
		return
	}

	e.tradeLog.Write(sig.Symbol, string(sig.Type), order.Price, order.FilledQty, sig.Reason, order.ID)

	e.log.Info("trade", "symbol", sig.Symbol, "side", sig.Type, "qty", order.FilledQty, "price", order.Price)
	e.notify(ctx, fmt.Sprintf("Сделка: %s %s | qty=%.4f price=%.2f", sig.Symbol, sig.Type, order.FilledQty, order.Price))
}

func (e *Executor) notify(ctx context.Context, msg string) {
	if e.notifier != nil {
		e.notifier.Send(ctx, msg)
	}
}

func (e *Executor) Close() {
	if e.tradeLog != nil {
		e.tradeLog.Close()
	}
}
