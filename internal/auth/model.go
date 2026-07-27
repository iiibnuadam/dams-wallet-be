package auth

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name     string             `bson:"name" json:"name"`
	Username string             `bson:"username" json:"username"`
	Password string             `bson:"password" json:"-"`
	Role     string             `bson:"role" json:"role"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       string `json:"_id"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user"`
}
type UpdateProfileRequest struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
