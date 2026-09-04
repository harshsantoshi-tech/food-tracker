
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)


type Config struct {
	Env            string
	HTTPPort       string
	RequestTimeout time.Duration

	Postgres  PostgresConfig
	Redis     RedisConfig
	WhatsApp  WhatsAppConfig
	LLM       LLMConfig
	Nutrition NutritionConfig
}

type PostgresConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	DBName       string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.DBName, p.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}


type WhatsAppConfig struct {
	AccessToken   string
	PhoneNumberID string
	VerifyToken   string
	APIVersion    string
}

type LLMConfig struct {
	APIKey string
	Model  string
}


type NutritionConfig struct {
	Provider string
	APIKey   string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Env:      getEnv("APP_ENV", "development"),
		HTTPPort: getEnv("HTTP_PORT", "8080"),
		Postgres: PostgresConfig{
			Host:         getEnv("POSTGRES_HOST", "localhost"),
			Port:         getEnv("POSTGRES_PORT", "5432"),
			User:         getEnv("POSTGRES_USER", "foodtracker"),
			Password:     getEnv("POSTGRES_PASSWORD", "foodtracker"),
			DBName:       getEnv("POSTGRES_DB", "foodtracker"),
			SSLMode:      getEnv("POSTGRES_SSLMODE", "disable"),
			MaxOpenConns: getEnvInt("POSTGRES_MAX_OPEN_CONNS", 10),
			MaxIdleConns: getEnvInt("POSTGRES_MAX_IDLE_CONNS", 5),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		WhatsApp: WhatsAppConfig{
			AccessToken:   getEnv("WHATSAPP_ACCESS_TOKEN", ""),
			PhoneNumberID: getEnv("WHATSAPP_PHONE_NUMBER_ID", ""),
			VerifyToken:   getEnv("WHATSAPP_VERIFY_TOKEN", ""),
			APIVersion:    getEnv("WHATSAPP_API_VERSION", "v20.0"),
		},
		LLM: LLMConfig{
			APIKey: getEnv("LLM_API_KEY", ""),
			Model:  getEnv("LLM_MODEL", "claude-sonnet-4-6"),
		},
		Nutrition: NutritionConfig{
			Provider: getEnv("NUTRITION_PROVIDER", "usda"),
			APIKey:   getEnv("NUTRITION_API_KEY", ""),
		},
	}

	timeoutSec := getEnvInt("REQUEST_TIMEOUT_SECONDS", 10)
	cfg.RequestTimeout = time.Duration(timeoutSec) * time.Second

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.HTTPPort == "" {
		return fmt.Errorf("HTTP_PORT must not be empty")
	}
	if c.Postgres.Host == "" || c.Postgres.DBName == "" {
		return fmt.Errorf("postgres configuration is incomplete")
	}
	if c.Redis.Addr == "" {
		return fmt.Errorf("REDIS_ADDR must not be empty")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}