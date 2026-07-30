package insights

import (
	"net/http"
	"time"

	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func HandleGetInsights(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	owner := r.URL.Query().Get("owner")

	userID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	result, err := GetInsights(userID, period, owner)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, result)
}

// HandleAnalyzeInsights triggers a fresh AI narration and overwrites the
// saved analysis for this (user, period, owner). This is the explicit,
// user-triggered "Analisis dengan AI" action -- unlike HandleGetInsights,
// it always calls the LLM and costs tokens.
func HandleAnalyzeInsights(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	owner := r.URL.Query().Get("owner")

	userID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	result, err := AnalyzeInsights(userID, period, owner)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, result)
}
