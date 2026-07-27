package budget

import (
	"encoding/json"
	"net/http"
	"time"

	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func HandleOverview(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().Format("2006-01")
	}

	userID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	overview, err := GetBudgetOverview(userID, period)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, overview)
}

func HandleAvailableGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := GetAvailableGroups()
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, groups)
}

func HandleUpsertEnvelopes(w http.ResponseWriter, r *http.Request) {
	var req UpsertEnvelopesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Period == "" {
		response.BadRequest(w, "period is required")
		return
	}

	userID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	if err := UpsertEnvelopes(userID, req); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}
