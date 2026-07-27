package auth

import (
	"encoding/json"
	"net/http"

	"github.com/ibnuadam/dams-wallet-backend/pkg/response"
	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
)

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}
	if req.Username == "" || req.Password == "" {
		response.BadRequest(w, "username and password are required")
		return
	}

	resp, err := Login(req)
	if err != nil {
		response.Unauthorized(w, err.Error())
		return
	}
	response.Success(w, resp)
}
func HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r)
	user, err := findUserByID(userID)
	if err != nil {
		response.NotFound(w, "user not found")
		return
	}
	response.Success(w, user)
}

func HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := mw.GetUserID(r)
	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "invalid request body")
		return
	}

	if err := UpdateProfile(userID, req); err != nil {
		response.InternalError(w, err.Error())
		return
	}

	response.Success(w, map[string]string{"message": "profile updated successfully"})
}
