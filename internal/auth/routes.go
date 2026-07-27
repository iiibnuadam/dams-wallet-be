package auth

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Post("/login", HandleLogin)
}

func ProtectedRoutes(r chi.Router) {
	r.Get("/profile", HandleGetProfile)
	r.Put("/profile", HandleUpdateProfile)
}
