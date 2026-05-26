package mystrategy

import (
	"context"
	"fmt"

	models "trading-bot/internal/domain/entity"
)

type Strategy struct {
	name    string
	lastBuy map[string]float64 // для простого выхода
}

func New() *Strategy {
	return &Strategy{
		name:    "mystrategy",
		lastBuy: make(map[string]float64),
	}
}

func (s *Strategy) Name() string { return s.name }

func (s *Strategy) Init(ctx context.Context, symbols []string) error {
	return nil
}

// Улучшенная стратегия: Buy + Sell
func (s *Strategy) OnCandle(ctx context.Context, candle models.Candle) (*models.Signal, error) {
	changePercent := (candle.Close - candle.Open) / candle.Open * 100

	// === BUY ===
	if changePercent > 0.8 && candle.Volume > 1.8 {
		s.lastBuy[candle.Symbol] = candle.Close
		return &models.Signal{
			Symbol:   candle.Symbol,
			Type:     models.Buy,
			Price:    candle.Close,
			Quantity: 0.01,
			Reason:   "bullish_impulse",
		}, nil
	}

	// === SELL (выход) ===
	if entryPrice, ok := s.lastBuy[candle.Symbol]; ok {
		profitPercent := (candle.Close - entryPrice) / entryPrice * 100

		// Выходим либо в плюс 1.5%, либо в минус 0.8%
		if profitPercent >= 1.5 || profitPercent <= -0.8 {
			delete(s.lastBuy, candle.Symbol) // закрываем позицию
			return &models.Signal{
				Symbol:   candle.Symbol,
				Type:     models.Sell,
				Price:    candle.Close,
				Quantity: 0.01,
				Reason:   fmt.Sprintf("exit_%.1f%%", profitPercent),
			}, nil
		}
	}

	return nil, nil
}

func (s *Strategy) Stop(ctx context.Context) error {
	return nil
}
