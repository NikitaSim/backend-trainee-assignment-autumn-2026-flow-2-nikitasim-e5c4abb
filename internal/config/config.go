package config

import "os"

type Config struct {
	HTTPPort      string
	DatabaseURL   string
	RestaurantURL string
}

func Load() Config {
	return Config{
		HTTPPort:      getenv("HTTP_PORT", "8080"),
		DatabaseURL:   getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5433/kitchen?sslmode=disable"),
		RestaurantURL: getenv("RESTAURANT_URL", "http://localhost:8081"),
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
