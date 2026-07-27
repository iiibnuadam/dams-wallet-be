package debts

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func HandleList(w http.ResponseWriter, r *http.Request) {
	ownerParam := r.URL.Query().Get("owner")
	if ownerParam == "" {
		ownerParam = mw.GetUserID(r)
	}
	debts, err := GetDebts(ownerParam)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, debts)
}

func HandleStats(w http.ResponseWriter, r *http.Request) {
	ownerParam := r.URL.Query().Get("owner")
	if ownerParam == "" {
		ownerParam = mw.GetUserID(r)
	}
	stats, err := GetDebtStats(ownerParam)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, stats)
}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	ownerID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	debt, err := CreateDebt(req, ownerID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Created(w, debt)
}

func HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ownerID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	var update bson.M
	json.NewDecoder(r.Body).Decode(&update)
	if err := UpdateDebt(id, ownerID, update); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ownerID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	if err := DeleteDebt(id, ownerID); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleSettle(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ownerID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	var req SettleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.WalletID == "" {
		response.BadRequest(w, "walletId is required")
		return
	}
	if err := SettleDebt(id, req.WalletID, ownerID); err != nil {
		response.Error(w, 400, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}
