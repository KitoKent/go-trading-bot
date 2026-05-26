package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Exchange string   `yaml:"exchange"`
	Symbols  []string `yaml:"symbols"`
	Interval string   `yaml:"interval"`
	Strategy string   `yaml:"strategy"`

	TradeLog string `yaml:"trade_log"`

	Risk RiskConfig `yaml:"risk"`

	Binance struct {
		APIKey    string
		SecretKey string
	}

	Telegram struct {
		BotToken string
		ChatID   string
	}

	LogLevel string `yaml:"log_level"`
}

type RiskConfig struct {
	MaxPositions int     `yaml:"max_positions"`
	RiskPercent  float64 `yaml:"risk_percent"`
	Testnet      bool    `yaml:"testnet"`
}

func Load(envPath, yamlPath string) (*Config, error) {
	// 1. Загружаем .env (секреты)
	_ = godotenv.Load(envPath)

	// 2. Загружаем config.yaml (настройки)
	cfg := &Config{}
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("config yaml: %w", err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	// 3. Секреты из .env (перезаписываем, если есть)
	if v := os.Getenv("BINANCE_API_KEY"); v != "" {
		cfg.Binance.APIKey = v
	}
	if v := os.Getenv("BINANCE_SECRET_KEY"); v != "" {
		cfg.Binance.SecretKey = v
	}
	if v := os.Getenv("TELEGRAM_BOT_TOKEN"); v != "" {
		cfg.Telegram.BotToken = v
	}
	if v := os.Getenv("TELEGRAM_CHAT_ID"); v != "" {
		cfg.Telegram.ChatID = v
	}
	// Переменные из .env могут перезаписать YAML-поля
	if v := os.Getenv("BINANCE_TESTNET"); v != "" {
		cfg.Risk.Testnet, _ = strconv.ParseBool(v)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if len(c.Symbols) == 0 {
		return fmt.Errorf("symbols: at least one required")
	}
	if c.Exchange == "" {
		c.Exchange = "binance"
	}
	if c.Interval == "" {
		c.Interval = "1m"
	}
	if c.Strategy == "" {
		c.Strategy = "mystrategy"
	}
	if c.TradeLog == "" {
		c.TradeLog = "data/trades.jsonl"
	}
	if c.Risk.MaxPositions <= 0 {
		c.Risk.MaxPositions = 3
	}
	if c.Risk.RiskPercent <= 0 {
		c.Risk.RiskPercent = 2.0
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	return nil
}
