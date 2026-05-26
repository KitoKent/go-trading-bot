package exchange

import (
	"fmt"

	"trading-bot/internal/exchange/binance"
)

func New(name, apiKey, secretKey string, testnet bool) (Client, error) {
	switch name {
	case "binance":
		return binance.New(apiKey, secretKey, testnet), nil
	// Добавьте новую биржу здесь:
	// case "bybit":
	//     return bybit.New(...), nil
	default:
		return nil, fmt.Errorf("unknown exchange: %s", name)
	}
}
