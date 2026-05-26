package backtest

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	models "trading-bot/internal/domain/entity"
	"trading-bot/internal/strategy"
)

type Engine struct {
	strategy strategy.Strategy
	log      *slog.Logger

	TotalSignals int
	Buys         int
	Sells        int
	TotalPnL     float64 // Теперь считаем реальную прибыль/убыток
	TotalTrades  int

	trades []TradeResult
}

type TradeResult struct {
	Time   time.Time         `json:"time"`
	Symbol string            `json:"symbol"`
	Type   models.SignalType `json:"type"`
	Price  float64           `json:"price"`
	Reason string            `json:"reason"`
	PnL    float64           `json:"pnl,omitempty"` // для будущего улучшения
}

func New(s strategy.Strategy) *Engine {
	return &Engine{
		strategy: s,
		log:      slog.With("module", "backtest"),
		trades:   make([]TradeResult, 0),
	}
}

func (e *Engine) Run(ctx context.Context, candles []models.Candle) error {
	e.log.Info("backtest started", "candles", len(candles))

	if err := e.strategy.Init(ctx, uniqueSymbols(candles)); err != nil {
		return err
	}

	// Простая имитация позиции (для начала)
	var position *models.Signal // сейчас держим только одну позицию

	for _, candle := range candles {
		sig, err := e.strategy.OnCandle(ctx, candle)
		if err != nil {
			continue
		}
		if sig == nil || sig.Type == models.None {
			continue
		}

		e.TotalSignals++

		if sig.Type == models.Buy {
			e.Buys++
			if position == nil { // открываем позицию
				position = sig
				e.log.Debug("position opened", "symbol", sig.Symbol, "price", sig.Price)
			}
		} else if sig.Type == models.Sell && position != nil {
			e.Sells++
			// Считаем PnL
			pnl := (sig.Price - position.Price) / position.Price * 100 // в процентах
			e.TotalPnL += pnl
			e.TotalTrades++

			e.log.Debug("position closed", "symbol", sig.Symbol, "entry", position.Price, "exit", sig.Price, "pnl%", pnl)

			position = nil // закрываем позицию
		}

		e.trades = append(e.trades, TradeResult{
			Time:   candle.CloseTime,
			Symbol: sig.Symbol,
			Type:   sig.Type,
			Price:  sig.Price,
			Reason: sig.Reason,
		})
	}

	e.log.Info("backtest completed", "signals", e.TotalSignals, "trades", e.TotalTrades)
	return nil
}

func (e *Engine) Report() string {
	var b strings.Builder
	b.WriteString("=== Backtest Report ===\n")
	b.WriteString(fmt.Sprintf("Total Signals:     %d\n", e.TotalSignals))
	b.WriteString(fmt.Sprintf("Buy Signals:       %d\n", e.Buys))
	b.WriteString(fmt.Sprintf("Sell Signals:      %d\n", e.Sells))
	b.WriteString(fmt.Sprintf("Closed Trades:     %d\n", e.TotalTrades))
	b.WriteString(fmt.Sprintf("Total PnL:         %.2f%%\n", e.TotalPnL))

	if e.TotalTrades > 0 {
		avgPnL := e.TotalPnL / float64(e.TotalTrades)
		b.WriteString(fmt.Sprintf("Average PnL:       %.2f%%\n", avgPnL))
	}

	return b.String()
}

func uniqueSymbols(candles []models.Candle) []string {
	seen := make(map[string]bool)
	var out []string
	for _, c := range candles {
		if !seen[c.Symbol] {
			seen[c.Symbol] = true
			out = append(out, c.Symbol)
		}
	}
	return out
}
