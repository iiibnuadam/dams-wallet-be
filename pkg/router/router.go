package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ibnuadam/dams-wallet-backend/internal/auth"
	"github.com/ibnuadam/dams-wallet-backend/internal/budget"
	"github.com/ibnuadam/dams-wallet-backend/internal/categories"
	"github.com/ibnuadam/dams-wallet-backend/internal/dashboard"
	"github.com/ibnuadam/dams-wallet-backend/internal/debts"
	"github.com/ibnuadam/dams-wallet-backend/internal/goals"
	"github.com/ibnuadam/dams-wallet-backend/internal/insights"
	"github.com/ibnuadam/dams-wallet-backend/internal/routines"
	"github.com/ibnuadam/dams-wallet-backend/internal/transactions"
	"github.com/ibnuadam/dams-wallet-backend/internal/wallets"
	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
)

func Setup() *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Root & Health check
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]string{
			"app":    "Dams Wallet API",
			"status": "running",
			"docs":   "Check /api/health for system status",
		})
	})
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, map[string]string{"status": "ok"})
	})

	// Auth routes (mixed public and protected)
	r.Route("/api/auth", func(r chi.Router) {
		auth.Routes(r) // Public: /login
		
		r.Group(func(r chi.Router) {
			r.Use(mw.Authenticate)
			auth.ProtectedRoutes(r) // Protected: /profile
		})
	})

	// Other protected routes
	r.Group(func(r chi.Router) {
		r.Use(mw.Authenticate)

		r.Route("/api/wallets", func(r chi.Router) {
			wallets.Routes(r)
		})
		r.Route("/api/transactions", func(r chi.Router) {
			transactions.Routes(r)
		})
		r.Route("/api/categories", func(r chi.Router) {
			categories.Routes(r)
		})
		r.Route("/api/budget", func(r chi.Router) {
			budget.Routes(r)
		})
		r.Route("/api/goals", func(r chi.Router) {
			goals.Routes(r)
		})
		r.Route("/api/routines", func(r chi.Router) {
			routines.Routes(r)
		})
		r.Route("/api/debts", func(r chi.Router) {
			debts.Routes(r)
		})
		r.Route("/api/dashboard", func(r chi.Router) {
			dashboard.Routes(r)
		})
		r.Route("/api/insights", func(r chi.Router) {
			insights.Routes(r)
		})
	})

	return r
}
