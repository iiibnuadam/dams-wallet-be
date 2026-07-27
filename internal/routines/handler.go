package routines

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
	routines, err := GetRoutines(ownerParam)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, routines)
}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateRoutineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	ownerID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	routine, err := CreateRoutine(req, ownerID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Created(w, routine)
}

func HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var update bson.M
	json.NewDecoder(r.Body).Decode(&update)
	if err := UpdateRoutine(id, update); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := DeleteRoutine(id); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleCheck(w http.ResponseWriter, r *http.Request) {
	ownerID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	count, err := CheckAndGenerate(ownerID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]int{"generated": count})
}

func HandlePending(w http.ResponseWriter, r *http.Request) {
	ownerID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	pending, err := GetPendingTransactions(ownerID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, pending)
}
