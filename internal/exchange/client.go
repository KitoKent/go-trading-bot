package exchange

import (
	"context"

	models "trading-bot/internal/domain/entity"
)

type Client interface {
	Name() string
	Ping(ctx context.Context) error
	GetCandles(ctx context.Context, symbol, interval string, limit int) ([]models.Candle, error)
	GetTicker(ctx context.Context, symbol string) (float64, error)
	Buy(ctx context.Context, symbol string, quantity float64) (*models.Order, error)
	Sell(ctx context.Context, symbol string, quantity float64) (*models.Order, error)
	SubscribeCandles(ctx context.Context, symbol, interval string, ch chan<- models.Candle) error
}
