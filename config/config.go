package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI  string
	DBName    string
	JWTSecret string
	Port      string

	DeepSeekAPIKey    string
	LLMModel          string
	LLMTimeoutSeconds int
}

var App Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	App = Config{
		MongoURI:  getEnv("MONGODB_URI", ""),
		DBName:    getEnv("DB_NAME", "dams-wallet"),
		JWTSecret: getEnv("JWT_SECRET", "changeme"),
		Port:      getEnv("PORT", "8080"),

		DeepSeekAPIKey:    getEnv("DEEPSEEK_API_KEY", ""),
		LLMModel:          getEnv("LLM_MODEL", "deepseek-chat"),
		LLMTimeoutSeconds: getEnvInt("LLM_TIMEOUT_SECONDS", 15),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
