package models

import "time"

type Candle struct {
	Symbol    string    `json:"symbol"`
	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

type OrderSide string

const (
	BuyOrder  OrderSide = "buy"
	SellOrder OrderSide = "sell"
)

type OrderType string

const (
	MarketOrder OrderType = "market"
	LimitOrder  OrderType = "limit"
)

type Order struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Side      OrderSide `json:"side"`
	Type      OrderType `json:"type"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	FilledQty float64   `json:"filled_qty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type SignalType string

const (
	Buy  SignalType = "buy"
	Sell SignalType = "sell"
	None SignalType = "none"
)

type Signal struct {
	Symbol   string     `json:"symbol"`
	Type     SignalType `json:"type"`
	Price    float64    `json:"price"`
	Quantity float64    `json:"quantity"`
	Reason   string     `json:"reason"`
}

type TradeRecord struct {
	Time     time.Time `json:"time"`
	Symbol   string    `json:"symbol"`
	Side     string    `json:"side"`
	Price    float64   `json:"price"`
	Quantity float64   `json:"quantity"`
	Reason   string    `json:"reason"`
	OrderID  string    `json:"order_id,omitempty"`
}
