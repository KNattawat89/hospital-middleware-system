package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Port    string `mapstructure:"port"`
	Debug   bool   `mapstructure:"debug"`
	Migrate bool   `mapstructure:"migrate"`
}

type JwtConfig struct {
	Secret          string `mapstructure:"secret"`
	AccessTokenTTL  string `mapstructure:"access_token_ttl"`
	RefreshTokenTTL string `mapstructure:"refresh_token_ttl"`
}

type DBConfig struct {
	Dsn string `mapstructure:"dsn"`
}

type Config struct {
	App AppConfig `mapstructure:"app"`
	Dev bool      `mapstructure:"dev"`
	Db  DBConfig  `mapstructure:"db"`
	Jwt JwtConfig `mapstructure:"jwt"`
}

var (
	config   *Config
	configMu sync.RWMutex
)

// NewConfig loads variables from a `.env` file (if present) into the process
// environment, then builds the Config from it. In production, real
// environment variables are set directly and no `.env` file is required.
func NewConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	debug, err := parseBoolEnv("APP_DEBUG", false)
	if err != nil {
		return nil, err
	}

	migrate, err := parseBoolEnv("APP_MIGRATE", false)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		App: AppConfig{
			Name:    getEnv("APP_NAME", "hospital-middleware-system"),
			Port:    getEnv("APP_PORT", "8080"),
			Debug:   debug,
			Migrate: migrate,
		},
		Dev: debug,
		Db: DBConfig{
			Dsn: fmt.Sprintf(
				"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
				getEnv("DB_HOST", "localhost"),
				getEnv("DB_PORT", "5432"),
				getEnv("DB_USER", "postgres"),
				getEnv("DB_PASSWORD", "postgres"),
				getEnv("DB_NAME", "postgres"),
				getEnv("DB_SSLMODE", "disable"),
			),
		},
		Jwt: JwtConfig{
			Secret:          getEnv("JWT_SECRET", ""),
			AccessTokenTTL:  getEnv("JWT_ACCESS_TOKEN_TTL", "15m"),
			RefreshTokenTTL: getEnv("JWT_REFRESH_TOKEN_TTL", "168h"),
		},
	}

	configMu.Lock()
	config = cfg
	configMu.Unlock()

	return cfg, nil
}

func GetConfig() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return config
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func parseBoolEnv(key string, fallback bool) (bool, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return b, nil
}
