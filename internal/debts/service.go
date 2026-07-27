package debts

import (
	"context"
	"fmt"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetDebts(ownerParam string) ([]Debt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

		query := bson.M{}
	if ownerParam != "" && ownerParam != "ALL" {
		if oid, err := primitive.ObjectIDFromHex(ownerParam); err == nil {
			query["owner"] = oid
		} else if ownerID, err := resolveUsername(ownerParam); err == nil {
			query["owner"] = ownerID
		} else {
			return []Debt{}, nil
		}
	}

	cursor, err := db.Col("debts").Find(ctx, query, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
	if err != nil {
		return nil, err
	}
	var debts []Debt
	cursor.All(ctx, &debts)
	if debts == nil {
		debts = []Debt{}
	}
	return debts, nil
}

func CreateDebt(req CreateDebtRequest, ownerID primitive.ObjectID) (*Debt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	loanDate, _ := time.Parse(time.RFC3339, req.LoanDate)
	now := time.Now()
	doc := bson.M{
		"type": req.Type, "person": req.Person, "amount": req.Amount,
		"description": req.Description, "loanDate": loanDate,
		"status": "ACTIVE", "owner": ownerID,
		"notes": req.Notes, "createdAt": now, "updatedAt": now,
	}
	if req.DueDate != "" {
		if dd, err := time.Parse(time.RFC3339, req.DueDate); err == nil {
			doc["dueDate"] = dd
		}
	}

	res, err := db.Col("debts").InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	var debt Debt
	db.Col("debts").FindOne(ctx, bson.M{"_id": res.InsertedID}).Decode(&debt)
	return &debt, nil
}

func UpdateDebt(id string, ownerID primitive.ObjectID, update bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, _ := primitive.ObjectIDFromHex(id)
	update["updatedAt"] = time.Now()
	_, err := db.Col("debts").UpdateOne(ctx, bson.M{"_id": oid, "owner": ownerID}, bson.M{"$set": update})
	return err
}

func DeleteDebt(id string, ownerID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, _ := primitive.ObjectIDFromHex(id)
	_, err := db.Col("debts").DeleteOne(ctx, bson.M{"_id": oid, "owner": ownerID})
	return err
}

func GetDebtStats(ownerParam string) (*DebtStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("debts").Aggregate(ctx, mongo.Pipeline{
				{{Key: "$match", Value: func() bson.M {
			q := bson.M{"status": "ACTIVE"}
			if ownerParam != "" && ownerParam != "ALL" {
				if oid, err := primitive.ObjectIDFromHex(ownerParam); err == nil {
					q["owner"] = oid
				} else if ownerID, err := resolveUsername(ownerParam); err == nil {
					q["owner"] = ownerID
				} else {
					q["owner"] = primitive.NilObjectID
				}
			}
			return q
		}()}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$type"},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	})
	if err != nil {
		return nil, err
	}
	var result []struct {
		ID    string  `bson:"_id"`
		Total float64 `bson:"total"`
	}
	cursor.All(ctx, &result)

	stats := &DebtStats{}
	for _, r := range result {
		if r.ID == "LENT" {
			stats.Lent = r.Total
		} else if r.ID == "BORROWED" {
			stats.Borrowed = r.Total
		}
	}
	return stats, nil
}

func SettleDebt(id string, walletID string, ownerID primitive.ObjectID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	oid, _ := primitive.ObjectIDFromHex(id)
	var debt Debt
	if err := db.Col("debts").FindOne(ctx, bson.M{"_id": oid, "owner": ownerID}).Decode(&debt); err != nil {
		return fmt.Errorf("debt not found")
	}
	if debt.Status == "PAID" {
		return fmt.Errorf("debt is already paid")
	}

	wid, _ := primitive.ObjectIDFromHex(walletID)
	txType := "INCOME"
	if debt.Type == "BORROWED" {
		txType = "EXPENSE"
	}

	now := time.Now()
	txDoc := bson.M{
		"date":        now,
		"amount":      debt.Amount,
		"description": fmt.Sprintf("Settlement: %s (%s)", debt.Description, debt.Person),
		"type":        txType,
		"wallet":      wid,
		"createdBy":   ownerID,
		"isDeleted":   false,
		"status":      "COMPLETED",
		"isTransfer":  false,
		"createdAt":   now, "updatedAt": now,
	}
	db.Col("transactions").InsertOne(ctx, txDoc)

	_, err := db.Col("debts").UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": bson.M{"status": "PAID", "updatedAt": now}})
	return err
}

func resolveUsername(username string) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	err := db.Col("users").FindOne(ctx, bson.M{
		"username": bson.M{"$regex": "^" + username + "$", "$options": "i"},
	}).Decode(&user)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return user.ID, nil
}
