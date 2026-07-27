package debts

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/", HandleList)
	r.Get("/stats", HandleStats)
	r.Post("/", HandleCreate)
	r.Put("/{id}", HandleUpdate)
	r.Delete("/{id}", HandleDelete)
	r.Post("/{id}/settle", HandleSettle)
}
