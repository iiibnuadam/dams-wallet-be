package wallets

import (
	"context"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func fetchWalletsWithBalance(filter bson.M) ([]WalletWithBalance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Ensure we only fetch non-deleted wallets
	if filter == nil {
		filter = bson.M{}
	}
	filter["isDeleted"] = false

	matchStage := bson.D{{Key: "$match", Value: filter}}

	pipeline := mongo.Pipeline{
		matchStage,
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "transactions"},
			{Key: "let", Value: bson.D{{Key: "walletId", Value: "$_id"}}},
			{Key: "pipeline", Value: mongo.Pipeline{
				{{Key: "$match", Value: bson.D{
					{Key: "$expr", Value: bson.D{
						{Key: "$and", Value: bson.A{
							bson.D{{Key: "$eq", Value: bson.A{"$isDeleted", false}}},
							bson.D{{Key: "$ne", Value: bson.A{"$status", "PENDING"}}},
							bson.D{{Key: "$eq", Value: bson.A{"$wallet", "$$walletId"}}},
						}},
					}},
				}}},
			}},
			{Key: "as", Value: "transactions"},
		}}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "balanceImpact", Value: bson.D{
				{Key: "$sum", Value: bson.D{
					{Key: "$map", Value: bson.D{
						{Key: "input", Value: "$transactions"},
						{Key: "as", Value: "txn"},
						{Key: "in", Value: bson.D{
							{Key: "$cond", Value: bson.A{
								bson.D{{Key: "$eq", Value: bson.A{"$$txn.type", "EXPENSE"}}},
								bson.D{{Key: "$multiply", Value: bson.A{"$$txn.amount", -1}}},
								bson.D{{Key: "$multiply", Value: bson.A{"$$txn.amount", 1}}},
							}},
						}},
					}},
				}},
			}},
		}}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "currentBalance", Value: bson.D{{Key: "$add", Value: bson.A{"$initialBalance", "$balanceImpact"}}}},
			{Key: "color", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$color", "BLUE"}}}},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "owner"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "ownerDetails"},
		}}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "ownerName", Value: bson.D{{Key: "$ifNull", Value: bson.A{
				bson.D{{Key: "$arrayElemAt", Value: bson.A{"$ownerDetails.name", 0}}},
				"Unknown",
			}}}},
			{Key: "owner", Value: bson.D{{Key: "$toString", Value: "$owner"}}},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "transactions", Value: 0},
			{Key: "ownerDetails", Value: 0},
			{Key: "balanceImpact", Value: 0},
		}}},
	}

	cursor, err := db.Col("wallets").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var wallets []WalletWithBalance
	if err := cursor.All(ctx, &wallets); err != nil {
		return nil, err
	}
	return wallets, nil
}

func findByID(id primitive.ObjectID) (*WalletWithBalance, error) {
	wallets, err := fetchWalletsWithBalance(bson.M{"_id": id})
	if err != nil {
		return nil, err
	}
	if len(wallets) > 0 {
		return &wallets[0], nil
	}
	return nil, nil
}

func insertOne(doc bson.M) (*primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := db.Col("wallets").InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	id := res.InsertedID.(primitive.ObjectID)
	return &id, nil
}

func updateOne(id primitive.ObjectID, update bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := db.Col("wallets").UpdateOne(ctx,
		bson.M{"_id": id},
		bson.M{"$set": update},
		options.Update(),
	)
	return err
}
