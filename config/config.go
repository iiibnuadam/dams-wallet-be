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

	// LLM provider selection: "deepseek" or "huggingface".
	LLMProvider string
	// Optional override of the provider's chat-completions endpoint.
	LLMBaseURL string
	// Provider-specific API keys. For DeepSeek this is required.
	// Hugging Face endpoints may be public, so the key is optional.
	DeepSeekAPIKey    string
	HuggingFaceAPIKey string
	LLMModel          string
	LLMTimeoutSeconds int
	// Optional sampling parameters, mainly used by the Hugging Face provider.
	LLMTemperature     float64
	LLMTopP            float64
	LLMReasoningEffort string
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

		LLMProvider:        getEnv("LLM_PROVIDER", "deepseek"),
		LLMBaseURL:         getEnv("LLM_BASE_URL", ""),
		DeepSeekAPIKey:     getEnv("DEEPSEEK_API_KEY", ""),
		HuggingFaceAPIKey:  getEnv("HF_API_KEY", ""),
		LLMModel:           getEnv("LLM_MODEL", ""),
		LLMTimeoutSeconds:  getEnvInt("LLM_TIMEOUT_SECONDS", 60),
		LLMTemperature:     getEnvFloat("LLM_TEMPERATURE", 1.0),
		LLMTopP:            getEnvFloat("LLM_TOP_P", 0.95),
		LLMReasoningEffort: getEnv("LLM_REASONING_EFFORT", "high"),
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

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			return n
		}
	}
	return fallback
}
