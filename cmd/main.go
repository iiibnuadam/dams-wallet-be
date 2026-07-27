package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ibnuadam/dams-wallet-backend/config"
	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"github.com/ibnuadam/dams-wallet-backend/pkg/router"
)

func main() {
	config.Load()
	db.Connect() // Ensure connection is established at startup

	r := router.Setup()

	addr := fmt.Sprintf(":%s", config.App.Port)
	log.Printf("🚀 Server running on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
