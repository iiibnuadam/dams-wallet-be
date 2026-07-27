package api

import (
	"fmt"
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
	defer func() {
		if err := recover(); err != nil {
			http.Error(w, "Vercel Function Crashed: "+fmt.Sprint(err), http.StatusInternalServerError)
		}
	}()
	handler.ServeHTTP(w, r)
}
