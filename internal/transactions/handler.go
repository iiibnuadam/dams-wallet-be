package transactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
	"github.com/ibnuadam/dams-wallet-backend/internal/categories"
	"github.com/ibnuadam/dams-wallet-backend/internal/wallets"
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

type ParseTextRequest struct {
	Text string `json:"text"`
}

func HandleParseText(w http.ResponseWriter, r *http.Request) {
	var req ParseTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid JSON body")
		return
	}

	if req.Text == "" {
		response.BadRequest(w, "text cannot be empty")
		return
	}

	ctx := r.Context()
	
	// Fetch context for LLM (categories and wallets)
	cats, _ := categories.GetCategories("ALL")
	catNames := make([]string, len(cats))
	for i, c := range cats {
		catNames[i] = c.Name
	}

	ws, _ := wallets.GetWallets("ALL")
	walletNames := make([]string, len(ws))
	for i, wlt := range ws {
		walletNames[i] = wlt.Name
	}

	// Create context with timeout for AI processing (increased to 120s for slow LLMs)
	aiCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	parsedResult, err := ParseTransactionFromText(aiCtx, req.Text, catNames, walletNames)
	if err != nil {
		response.InternalError(w, fmt.Sprintf("AI parsing failed: %v", err))
		return
	}

	response.Success(w, parsedResult)
}
