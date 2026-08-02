package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/config"
	"github.com/ibnuadam/dams-wallet-backend/internal/insights"
	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"github.com/ibnuadam/dams-wallet-backend/pkg/llm"
	"github.com/ibnuadam/dams-wallet-backend/pkg/router"
)

func main() {
	// Catch any panic during startup (e.g. a future init step that isn't
	// already fatal-exiting cleanly) and turn it into a single log line +
	// clean exit instead of a raw stack-trace crash.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Fatal error during startup: %v", r)
			os.Exit(1)
		}
	}()

	config.Load()
	db.Connect() // Ensure connection is established at startup

	insights.SetLLMClient(llm.New(llm.Config{
		APIKey:  config.App.DeepSeekAPIKey,
		Model:   config.App.LLMModel,
		Timeout: time.Duration(config.App.LLMTimeoutSeconds) * time.Second,
	}))

	r := router.Setup()

	addr := fmt.Sprintf(":%s", config.App.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// Run the server in a goroutine so the main goroutine is free to wait
	// for a shutdown signal.
	go func() {
		log.Printf("🚀 Server running on http://localhost%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Block until Ctrl+C (SIGINT) or a termination request (SIGTERM, e.g.
	// from a process manager) arrives, then shut down gracefully instead of
	// dropping in-flight requests.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	if err := db.Disconnect(ctx); err != nil {
		log.Printf("MongoDB disconnect error: %v", err)
	}
	log.Println("Shutdown complete")
}
