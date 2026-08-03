package wallets

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func HandleList(w http.ResponseWriter, r *http.Request) {
	owner := r.URL.Query().Get("owner")
	wallets, err := GetWallets(owner)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, wallets)
}

func HandleGetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wallet, err := GetWalletByID(id)
	if err != nil {
		response.BadRequest(w, "invalid wallet id")
		return
	}
	if wallet == nil {
		response.NotFound(w, "wallet not found")
		return
	}
	response.Success(w, wallet)
}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || req.Type == "" {
		response.BadRequest(w, "name and type are required")
		return
	}

	ownerID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	wallet, err := CreateWallet(req, ownerID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Created(w, wallet)
}

func HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var update bson.M
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if name, ok := update["name"].(string); ok {
		update["name"] = strings.TrimSpace(name)
	}

	wallet, err := UpdateWallet(id, update)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, wallet)
}

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := DeleteWallet(id); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}
