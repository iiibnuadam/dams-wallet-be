package auth

import (
	"context"
	"strings"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
)

func findUserByUsername(username string) (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user User
	err := db.Col("users").FindOne(ctx, bson.M{
		"username": bson.M{"$regex": "^" + strings.ToUpper(username) + "$", "$options": "i"},
	}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
