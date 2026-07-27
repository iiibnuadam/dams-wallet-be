package goals

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
	owner := r.URL.Query().Get("owner")
	goals, err := GetGoals(owner)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, goals)
}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateGoalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	userID, _ := primitive.ObjectIDFromHex(mw.GetUserID(r))
	goal, err := CreateGoal(req, userID)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Created(w, goal)
}

func HandleGetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	detail, err := GetGoalByID(id)
	if err != nil {
		response.NotFound(w, "goal not found")
		return
	}
	response.Success(w, detail)
}

func HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var update bson.M
	json.NewDecoder(r.Body).Decode(&update)
	if err := UpdateGoal(id, update); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := DeleteGoal(id); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleAddItem(w http.ResponseWriter, r *http.Request) {
	goalID := chi.URLParam(r, "id")
	var req CreateGoalItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid body")
		return
	}
	item, err := AddGoalItem(goalID, req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Created(w, item)
}

func HandleUpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemId")
	var update bson.M
	json.NewDecoder(r.Body).Decode(&update)
	if err := UpdateGoalItem(itemID, update); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleDeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemId")
	if err := DeleteGoalItem(itemID); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleToggleItemComplete(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemId")
	var body struct{ IsCompleted bool `json:"isCompleted"` }
	json.NewDecoder(r.Body).Decode(&body)
	if err := UpdateGoalItem(itemID, bson.M{"isCompleted": body.IsCompleted}); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleAddGroup(w http.ResponseWriter, r *http.Request) {
	goalID := chi.URLParam(r, "id")
	var req GroupRequest
	json.NewDecoder(r.Body).Decode(&req)
	g, err := AddGroup(goalID, req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, g)
}

func HandleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	goalID := chi.URLParam(r, "id")
	groupID := chi.URLParam(r, "groupId")
	var req GroupRequest
	json.NewDecoder(r.Body).Decode(&req)
	if err := UpdateGroup(goalID, groupID, req); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	goalID := chi.URLParam(r, "id")
	groupID := chi.URLParam(r, "groupId")
	if err := DeleteGroup(goalID, groupID); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}

func HandleToggleGoalComplete(w http.ResponseWriter, r *http.Request) {
	goalID := chi.URLParam(r, "id")
	var req map[string]bool
	json.NewDecoder(r.Body).Decode(&req)
	isCompleted := req["isCompleted"]
	if err := UpdateGoal(goalID, bson.M{"isCompleted": isCompleted}); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true, "isCompleted": isCompleted})
}
