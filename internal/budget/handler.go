package budget

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
)

func HandleOverview(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().Format("2006-01")
	}

	overview, err := GetBudgetOverview(period)
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

	if err := UpsertEnvelopes(req); err != nil {
		if errors.Is(err, ErrDuplicateEnvelopeItem) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}
