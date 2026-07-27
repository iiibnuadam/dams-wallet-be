package budget

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/overview", HandleOverview)
	r.Get("/available-groups", HandleAvailableGroups)
	r.Put("/envelopes", HandleUpsertEnvelopes)
}
