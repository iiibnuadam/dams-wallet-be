package dashboard

import (
	"net/http"
	"time"

	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func HandleSummary(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")
	owner := r.URL.Query().Get("owner")
	walletId := r.URL.Query().Get("walletId")
	
	if period == "" && startDateStr == "" {
		period = time.Now().Format("2006-01")
	}
	userID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	summary, err := GetSummary(userID, period, startDateStr, endDateStr, owner, walletId)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, summary)
}

func HandleFinancialHealth(w http.ResponseWriter, r *http.Request) {
	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")
	owner := r.URL.Query().Get("owner")
	
	userID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	summary, err := GetFinancialHealth(userID, startDateStr, endDateStr, owner)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, summary)
}
