package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Code    int         `json:"code"`
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func JSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Envelope{
		Code:    http.StatusOK,
		Status:  "OK",
		Message: "Success",
		Data:    data,
	})
}

func Created(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusCreated, Envelope{
		Code:    http.StatusCreated,
		Status:  "Created",
		Message: "Resource created successfully",
		Data:    data,
	})
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, Envelope{
		Code:    status,
		Status:  http.StatusText(status),
		Message: msg,
	})
}

func BadRequest(w http.ResponseWriter, msg string) {
	Error(w, http.StatusBadRequest, msg)
}

func Unauthorized(w http.ResponseWriter, msg string) {
	Error(w, http.StatusUnauthorized, msg)
}

func NotFound(w http.ResponseWriter, msg string) {
	Error(w, http.StatusNotFound, msg)
}

func InternalError(w http.ResponseWriter, msg string) {
	Error(w, http.StatusInternalServerError, msg)
}
