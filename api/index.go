package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/config"
	"github.com/ibnuadam/dams-wallet-backend/pkg/llm"
	"github.com/ibnuadam/dams-wallet-backend/pkg/router"
)

var handler http.Handler

func init() {
	config.Load()

	llmCfg := llm.Config{
		Provider:        config.App.LLMProvider,
		BaseURL:         config.App.LLMBaseURL,
		Model:           config.App.LLMModel,
		Timeout:         time.Duration(config.App.LLMTimeoutSeconds) * time.Second,
		Temperature:     config.App.LLMTemperature,
		TopP:            config.App.LLMTopP,
		ReasoningEffort: config.App.LLMReasoningEffort,
	}
	if config.App.LLMProvider == "huggingface" {
		llmCfg.APIKey = config.App.HuggingFaceAPIKey
	} else {
		llmCfg.APIKey = config.App.DeepSeekAPIKey
	}
	router.SetInsightsLLMClient(llm.New(llmCfg))

	handler = router.Setup()
}

// Handler is the main entry point for Vercel Serverless Functions.
func Handler(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, "Vercel Function Crashed: "+fmt.Sprint(err), http.StatusInternalServerError)
		}
	}()
	handler.ServeHTTP(w, r)
}
