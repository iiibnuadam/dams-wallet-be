package goals

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/", HandleList)
	r.Post("/", HandleCreate)
	r.Get("/{id}", HandleGetByID)
	r.Put("/{id}", HandleUpdate)
	r.Patch("/{id}/complete", HandleToggleGoalComplete)
	r.Delete("/{id}", HandleDelete)

	// Groups
	r.Post("/{id}/groups", HandleAddGroup)
	r.Put("/{id}/groups/{groupId}", HandleUpdateGroup)
	r.Delete("/{id}/groups/{groupId}", HandleDeleteGroup)

	// Items
	r.Post("/{id}/items", HandleAddItem)
	r.Put("/items/{itemId}", HandleUpdateItem)
	r.Delete("/items/{itemId}", HandleDeleteItem)
	r.Patch("/items/{itemId}/complete", HandleToggleItemComplete)
}
