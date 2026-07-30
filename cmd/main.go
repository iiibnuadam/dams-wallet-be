package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/config"
	"github.com/ibnuadam/dams-wallet-backend/internal/insights"
	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"github.com/ibnuadam/dams-wallet-backend/pkg/llm"
	"github.com/ibnuadam/dams-wallet-backend/pkg/router"
)

func main() {
	config.Load()
	db.Connect() // Ensure connection is established at startup

	insights.SetLLMClient(llm.New(llm.Config{
		APIKey:  config.App.DeepSeekAPIKey,
		Model:   config.App.LLMModel,
		Timeout: time.Duration(config.App.LLMTimeoutSeconds) * time.Second,
	}))

	r := router.Setup()

	addr := fmt.Sprintf(":%s", config.App.Port)
	log.Printf("🚀 Server running on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
