package main

import (
	_ "github.com/joho/godotenv/autoload" // load env vars from .env file
	"github.com/kelseyhightower/envconfig"
)

// config represents application configuration.
type config struct {
	// Development mode enables dev-only features
	Development bool `envconfig:"DEVELOPMENT" default:"false"`

	// secret Telegram token
	BotToken string `envconfig:"BOT_TOKEN" required:"true"`

	// Port to listen on
	Port int `envconfig:"PORT" default:"8080"`

	// PostgreSQL server
	PGHost     string `envconfig:"POSTGRES_HOST" required:"true"`
	PGPort     uint16 `envconfig:"POSTGRES_PORT" required:"true"`
	PGDatabase string `envconfig:"POSTGRES_DATABASE" required:"true"`
	PGParams   string `envconfig:"POSTGRES_PARAMS"`
	PGUsername string `envconfig:"POSTGRES_USERNAME" required:"true"`
	PGPassword string `envconfig:"POSTGRES_PASSWORD" required:"true"`

	// Redis
	RedisHost    string `envconfig:"REDIS_HOST" required:"true"`
	RedisPort    uint16 `envconfig:"REDIS_PORT" required:"true"`
	RedisTimeKey int    `envconfig:"REDIS_TIME_KEY_EXIST" required:"true"`
}

func readConfig() (*config, error) {
	cfg := &config{}
	err := envconfig.Process("", cfg)
	return cfg, err
}
