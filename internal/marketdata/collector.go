package marketdata

import (
	"context"
	"log/slog"
	"sync"
	"time"

	models "trading-bot/internal/domain/entity"
	"trading-bot/internal/exchange"
)

type Collector struct {
	mu       sync.RWMutex
	client   exchange.Client
	buf      map[string][]models.Candle
	candleCh chan models.Candle
	log      *slog.Logger
}

func New(client exchange.Client) *Collector {
	return &Collector{
		client:   client,
		buf:      make(map[string][]models.Candle),
		candleCh: make(chan models.Candle, 200),
		log:      slog.With("module", "marketdata"),
	}
}

// C возвращает канал, в который приходят новые свечи в реальном времени.
func (c *Collector) C() <-chan models.Candle {
	return c.candleCh
}

// Start запускает сбор свечей для каждого символа.
// 1. Загружает последние 100 свечей (история).
// 2. Подписывается на WebSocket — новые свечи приходят в канал C().
// 3. Если WS недоступен — переключается на polling.
func (c *Collector) Start(ctx context.Context, symbols []string, interval string) error {
	for _, symbol := range symbols {
		candles, err := c.client.GetCandles(ctx, symbol, interval, 100)
		if err != nil {
			return err
		}
		for i := range candles {
			c.addCandle(candles[i])
		}

		wsCh := make(chan models.Candle, 100)
		if err := c.client.SubscribeCandles(ctx, symbol, interval, wsCh); err != nil {
			c.log.Warn("ws unavailable, using polling", "symbol", symbol)
			go c.pollLoop(ctx, symbol, interval)
			continue
		}
		go c.readWS(ctx, wsCh)
	}
	return nil
}

// readWS читает свечи из WebSocket-канала и отправляет в общий канал candleCh.
func (c *Collector) readWS(ctx context.Context, wsCh <-chan models.Candle) {
	for {
		select {
		case <-ctx.Done():
			return
		case candle := <-wsCh:
			c.addCandle(candle)
			select {
			case c.candleCh <- candle:
			default:
				// Если канал переполнен — пропускаем (не блокируем WS).
			}
		}
	}
}

// pollLoop опрашивает REST API раз в 30 секунд (fallback для WS).
func (c *Collector) pollLoop(ctx context.Context, symbol, interval string) {
	tk := time.NewTicker(30 * time.Second)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			candles, err := c.client.GetCandles(ctx, symbol, interval, 1)
			if err != nil {
				continue
			}
			for i := range candles {
				c.addCandle(candles[i])
				select {
				case c.candleCh <- candles[i]:
				default:
				}
			}
		}
	}
}

// addCandle сохраняет свечу в буфер (последние 500 шт на символ).
func (c *Collector) addCandle(candle models.Candle) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := candle.Symbol
	c.buf[s] = append(c.buf[s], candle)
	if len(c.buf[s]) > 500 {
		c.buf[s] = c.buf[s][len(c.buf[s])-500:]
	}
}

// Candles возвращает последние n свечей для символа для стратегий,
// которым нужна история (индикаторы, скользящие средние и т.д.).
func (c *Collector) Candles(symbol string, n int) []models.Candle {
	c.mu.RLock()
	defer c.mu.RUnlock()
	candles := c.buf[symbol]
	if len(candles) == 0 {
		return nil
	}
	if n <= 0 || n > len(candles) {
		n = len(candles)
	}
	out := make([]models.Candle, n)
	copy(out, candles[len(candles)-n:])
	return out
}
