package dashboard

import "github.com/go-chi/chi/v5"

func Routes(r chi.Router) {
	r.Get("/summary", HandleSummary)
	r.Get("/financial-health", HandleFinancialHealth)
}
