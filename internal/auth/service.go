package auth

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/ibnuadam/dams-wallet-backend/config"
	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	mw "github.com/ibnuadam/dams-wallet-backend/pkg/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

func Login(req LoginRequest) (*LoginResponse, error) {
	user, err := findUserByUsername(req.Username)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.New("invalid credentials")
		}
		return nil, err
	}

	// If password is set, verify it; otherwise allow plain-text comparison for legacy
	if user.Password != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			return nil, errors.New("invalid credentials")
		}
	}

	// Generate JWT
	claims := &mw.Claims{
		UserID:   user.ID.Hex(),
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.App.JWTSecret))
	if err != nil {
		return nil, err
	}

	resp := &LoginResponse{Token: tokenStr}
	resp.User.ID = user.ID.Hex()
	resp.User.Name = user.Name
	resp.User.Username = user.Username
	resp.User.Role = user.Role

	return resp, nil
}
func UpdateProfile(userID string, req UpdateProfileRequest) error {
	objID, _ := primitive.ObjectIDFromHex(userID)
	update := bson.M{
		"$set": bson.M{
			"name": req.Name,
		},
	}

	if req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), 10)
		if err != nil {
			return err
		}
		update["$set"].(bson.M)["password"] = string(hashed)
	}

	_, err := db.Col("users").UpdateOne(context.Background(), bson.M{"_id": objID}, update)
	return err
}

func findUserByID(id string) (*User, error) {
	objID, _ := primitive.ObjectIDFromHex(id)
	var user User
	err := db.Col("users").FindOne(context.Background(), bson.M{"_id": objID}).Decode(&user)
	return &user, err
}
