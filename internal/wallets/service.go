package wallets

import (
	"context"
	"strings"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetWallets(ownerParam string) ([]WalletWithBalance, error) {
	if ownerParam == "" || strings.ToUpper(ownerParam) == "ALL" {
		return fetchWalletsWithBalance(nil)
	}

	// Try as ObjectID first
	if oid, err := primitive.ObjectIDFromHex(ownerParam); err == nil {
		return fetchWalletsWithBalance(bson.M{"owner": oid})
	}

	// Resolve username → ObjectID
	ownerID, err := resolveUsername(ownerParam)
	if err != nil {
		return []WalletWithBalance{}, nil
	}
	return fetchWalletsWithBalance(bson.M{"owner": ownerID})
}

func GetWalletByID(id string) (*WalletWithBalance, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	return findByID(oid)
}

func CreateWallet(req CreateWalletRequest, ownerID primitive.ObjectID) (*WalletWithBalance, error) {
	doc := bson.M{
		"name":           req.Name,
		"type":           req.Type,
		"owner":          ownerID,
		"initialBalance": req.InitialBalance,
		"color":          req.Color,
		"isDeleted":      false,
		"createdAt":      time.Now(),
		"updatedAt":      time.Now(),
	}
	if req.LiabilityDetails != nil {
		doc["liabilityDetails"] = req.LiabilityDetails
	}
	if req.BankDetails != nil {
		doc["bankDetails"] = req.BankDetails
	}

	oid, err := insertOne(doc)
	if err != nil {
		return nil, err
	}
	return findByID(*oid)
}

func UpdateWallet(id string, update bson.M) (*WalletWithBalance, error) {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	update["updatedAt"] = time.Now()
	if err := updateOne(oid, update); err != nil {
		return nil, err
	}
	return findByID(oid)
}

func DeleteWallet(id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	return updateOne(oid, bson.M{"isDeleted": true, "updatedAt": time.Now()})
}

func resolveUsername(username string) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	err := db.Col("users").FindOne(ctx, bson.M{
		"username": bson.M{"$regex": "^" + strings.ToUpper(username) + "$", "$options": "i"},
	}).Decode(&user)
	if err != nil {
		return primitive.NilObjectID, mongo.ErrNoDocuments
	}
	return user.ID, nil
}
