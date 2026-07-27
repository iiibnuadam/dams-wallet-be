package categories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetCategories(typeFilter string) ([]Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := bson.M{"isDeleted": false}
	if typeFilter != "" && typeFilter != "ALL" {
		query["type"] = typeFilter
	}

	cursor, err := db.Col("categories").Find(ctx, query, options.Find().SetSort(bson.D{{Key: "name", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var cats []Category
	if err := cursor.All(ctx, &cats); err != nil {
		return nil, err
	}
	if cats == nil {
		cats = []Category{}
	}
	return cats, nil
}

func CreateCategory(req CreateCategoryRequest) (*Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check duplicate
	existing := db.Col("categories").FindOne(ctx, bson.M{
		"name":      bson.M{"$regex": "^" + strings.TrimSpace(req.Name) + "$", "$options": "i"},
		"type":      req.Type,
		"isDeleted": false,
	})
	if existing.Err() == nil {
		return nil, fmt.Errorf("category '%s' already exists", req.Name)
	} else if existing.Err() != mongo.ErrNoDocuments {
		return nil, existing.Err()
	}

	now := time.Now()
	doc := bson.M{
		"name":        req.Name,
		"type":        req.Type,
		"flexibility": req.Flexibility,
		"icon":        req.Icon,
		"color":       req.Color,
		"group":       req.Group,
		"bucket":      req.Bucket,
		"isDeleted":   false,
		"createdAt":   now,
		"updatedAt":   now,
	}

	res, err := db.Col("categories").InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	oid := res.InsertedID.(primitive.ObjectID)
	var cat Category
	db.Col("categories").FindOne(ctx, bson.M{"_id": oid}).Decode(&cat)
	return &cat, nil
}

func UpdateCategory(id string, req CreateCategoryRequest) (*Category, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	update := bson.M{"updatedAt": time.Now()}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Type != "" {
		update["type"] = req.Type
	}
	if req.Flexibility != "" {
		update["flexibility"] = req.Flexibility
	}
	if req.Icon != "" {
		update["icon"] = req.Icon
	}
	if req.Color != "" {
		update["color"] = req.Color
	}
	if req.Group != "" {
		update["group"] = req.Group
	}
	if req.Bucket != "" {
		update["bucket"] = req.Bucket
	}

	_, err = db.Col("categories").UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": update}, options.Update())
	if err != nil {
		return nil, err
	}

	var cat Category
	db.Col("categories").FindOne(ctx, bson.M{"_id": oid}).Decode(&cat)
	return &cat, nil
}

func DeleteCategory(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	_, err = db.Col("categories").UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": bson.M{"isDeleted": true, "updatedAt": time.Now()}})
	return err
}
