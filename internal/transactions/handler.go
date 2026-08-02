package transactions

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func HandleList(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{}
	for key := range r.URL.Query() {
		params[key] = r.URL.Query().Get(key)
	}
	f := ParseFilters(params)

	result, err := ListTransactions(f)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, result)
}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Amount <= 0 || req.Type == "" || req.Wallet == "" {
		response.BadRequest(w, "amount, type, and wallet are required")
		return
	}

	userID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	id, err := CreateTransaction(req, userID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Created(w, map[string]string{"_id": id.Hex()})
}

func HandleCreateBatch(w http.ResponseWriter, r *http.Request) {
	var reqs []CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&reqs); err != nil {
		response.BadRequest(w, "invalid request body array")
		return
	}

	if len(reqs) == 0 {
		response.Success(w, map[string]interface{}{"inserted": 0})
		return
	}

	userID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	count, err := CreateBatchTransactions(reqs, userID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Created(w, map[string]interface{}{"inserted": count})
}

func HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Amount <= 0 || req.Type == "" || req.Wallet == "" {
		response.BadRequest(w, "amount, type, and wallet are required")
		return
	}

	if err := UpdateTransaction(id, req); err != nil {
		if errors.Is(err, ErrGoalLinkedTransaction) {
			response.BadRequest(w, err.Error())
			return
		}
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := SoftDeleteTransaction(id); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleConfirm(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := ConfirmTransaction(id); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}
