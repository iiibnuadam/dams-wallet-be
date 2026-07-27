package categories

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
)

func HandleList(w http.ResponseWriter, r *http.Request) {
	t := r.URL.Query().Get("type")
	cats, err := GetCategories(t)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, cats)
}

func HandleCreate(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Name == "" || req.Type == "" {
		response.BadRequest(w, "name and type are required")
		return
	}

	cat, err := CreateCategory(req)
	if err != nil {
		response.Error(w, 409, err.Error())
		return
	}
	response.Created(w, cat)
}

func HandleUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	cat, err := UpdateCategory(id, req)
	if err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, cat)
}

func HandleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := DeleteCategory(id); err != nil {
		response.InternalError(w, err.Error())
		return
	}
	response.Success(w, map[string]bool{"success": true})
}
