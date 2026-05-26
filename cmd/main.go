package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"trading-bot/internal/backtest"
	"trading-bot/internal/bot"
	"trading-bot/internal/config"
	models "trading-bot/internal/domain/entity"
	"trading-bot/internal/exchange"
	"trading-bot/internal/strategy"
)

func main() {
	env := flag.String("env", ".env", "path to .env file")
	cfgPath := flag.String("config", "config.yaml", "path to config.yaml")
	mode := flag.String("mode", "live", "live | backtest")
	flag.Parse()

	cfg, err := config.Load(*env, *cfgPath)
	if err != nil {
		slog.Error("config", "error", err)
		os.Exit(1)
	}

	setupLog(cfg.LogLevel)
	slog.Info("starting", "mode", *mode, "exchange", cfg.Exchange, "strategy", cfg.Strategy)

	switch *mode {
	case "backtest":
		runBacktest(cfg)
	default:
		runLive(cfg)
	}
}

func runLive(cfg *config.Config) {
	b := bot.New(cfg)
	if err := b.Run(context.Background()); err != nil {
		slog.Error("bot", "error", err)
		os.Exit(1)
	}
}

func runBacktest(cfg *config.Config) {
	exch, err := exchange.New(cfg.Exchange, cfg.Binance.APIKey, cfg.Binance.SecretKey, cfg.Risk.Testnet)
	if err != nil {
		slog.Error("exchange", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	var allCandles []models.Candle
	for _, symbol := range cfg.Symbols {
		candles, err := exch.GetCandles(ctx, symbol, cfg.Interval, 500)
		if err != nil {
			slog.Error("get candles", "symbol", symbol, "error", err)
			os.Exit(1)
		}
		allCandles = append(allCandles, candles...)
	}

	slog.Info("backtest loaded", "candles", len(allCandles))

	strat, err := strategy.New(cfg.Strategy)
	if err != nil {
		slog.Error("strategy", "error", err)
		os.Exit(1)
	}

	engine := backtest.New(strat)
	if err := engine.Run(ctx, allCandles); err != nil {
		slog.Error("backtest", "error", err)
		os.Exit(1)
	}

	fmt.Println(engine.Report())
}

func setupLog(level string) {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: l})))
}
