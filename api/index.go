package api

import (
	"net/http"

	"github.com/ibnuadam/dams-wallet-backend/config"
	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"github.com/ibnuadam/dams-wallet-backend/pkg/router"
)

var handler http.Handler

func init() {
	config.Load()
	handler = router.Setup()
}

// Handler is the main entry point for Vercel Serverless Functions.
func Handler(w http.ResponseWriter, r *http.Request) {
	handler.ServeHTTP(w, r)
}
