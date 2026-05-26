package strategy

import (
	"context"
	"fmt"

	models "trading-bot/internal/domain/entity"
	"trading-bot/internal/strategy/mystrategy"
)

type Strategy interface {
	Name() string
	Init(ctx context.Context, symbols []string) error
	OnCandle(ctx context.Context, candle models.Candle) (*models.Signal, error)
	Stop(ctx context.Context) error
}

func New(name string) (Strategy, error) {
	switch name {
	case "mystrategy":
		return mystrategy.New(), nil
	// Добавьте свою стратегию здесь:
	// case "sma_cross":
	//     return smacross.New(...), nil
	default:
		return nil, fmt.Errorf("unknown strategy: %s", name)
	}
}
