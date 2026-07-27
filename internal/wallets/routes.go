package wallets

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/", HandleList)
	r.Post("/", HandleCreate)
	r.Get("/{id}", HandleGetByID)
	r.Put("/{id}", HandleUpdate)
	r.Delete("/{id}", HandleDelete)
}
