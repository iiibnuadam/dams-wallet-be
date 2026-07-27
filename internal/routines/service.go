package routines

import (
	"context"
	"fmt"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetRoutines(ownerParam string) ([]Routine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

		query := bson.M{}
	if ownerParam != "" && ownerParam != "ALL" {
		if oid, err := primitive.ObjectIDFromHex(ownerParam); err == nil {
			query["owner"] = oid
		} else if ownerID, err := resolveUsername(ownerParam); err == nil {
			query["owner"] = ownerID
		} else {
			return []Routine{}, nil
		}
	}

	cursor, err := db.Col("routines").Find(ctx,
		query,
		options.Find().SetSort(bson.D{{Key: "nextRun", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	var routines []Routine
	cursor.All(ctx, &routines)
	if routines == nil {
		routines = []Routine{}
	}
	return routines, nil
}

func CreateRoutine(req CreateRoutineRequest, ownerID primitive.ObjectID) (*Routine, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startDate, _ := time.Parse(time.RFC3339, req.StartDate)
	walletID, _ := primitive.ObjectIDFromHex(req.Wallet)
	now := time.Now()

	doc := bson.M{
		"description": req.Description, "amount": req.Amount, "type": req.Type,
		"wallet": walletID, "frequency": req.Frequency,
		"startDate": startDate, "nextRun": startDate,
		"status": "ACTIVE", "owner": ownerID,
		"createdAt": now, "updatedAt": now,
	}
	if req.TargetWallet != "" {
		if twID, err := primitive.ObjectIDFromHex(req.TargetWallet); err == nil {
			doc["targetWallet"] = twID
		}
	}
	if req.Category != "" {
		if cID, err := primitive.ObjectIDFromHex(req.Category); err == nil {
			doc["category"] = cID
		}
	}

	res, err := db.Col("routines").InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	var r Routine
	db.Col("routines").FindOne(ctx, bson.M{"_id": res.InsertedID}).Decode(&r)
	return &r, nil
}

func UpdateRoutine(id string, update bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, _ := primitive.ObjectIDFromHex(id)
	update["updatedAt"] = time.Now()

	if status, ok := update["status"].(string); ok && status == "ACTIVE" {
		var r Routine
		if err := db.Col("routines").FindOne(ctx, bson.M{"_id": oid}).Decode(&r); err == nil {
			now := time.Now()
			// Only adjust if it's strictly before today's date (ignoring time)
			// Actually just before now is fine for routines that run exactly on that time
			if r.NextRun.Before(now) {
				nextRun := r.NextRun
				for nextRun.Before(now) {
					switch r.Frequency {
					case "WEEKLY":
						nextRun = nextRun.AddDate(0, 0, 7)
					case "MONTHLY":
						nextRun = nextRun.AddDate(0, 1, 0)
					case "QUARTERLY":
						nextRun = nextRun.AddDate(0, 3, 0)
					case "YEARLY":
						nextRun = nextRun.AddDate(1, 0, 0)
					default:
						nextRun = nextRun.AddDate(0, 1, 0)
					}
				}
				update["nextRun"] = nextRun
			}
		}
	}

	_, err := db.Col("routines").UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": update})
	return err
}

func DeleteRoutine(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, _ := primitive.ObjectIDFromHex(id)
	_, err := db.Col("routines").DeleteOne(ctx, bson.M{"_id": oid})
	return err
}

func CheckAndGenerate(ownerID primitive.ObjectID) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	now := time.Now()
	cursor, err := db.Col("routines").Find(ctx, bson.M{
		"owner": ownerID, "status": "ACTIVE", "nextRun": bson.M{"$lte": now},
	})
	if err != nil {
		return 0, err
	}
	var routines []Routine
	cursor.All(ctx, &routines)

	count := 0
	for _, r := range routines {
		txDoc := bson.M{
			"date":        r.NextRun,
			"amount":      r.Amount,
			"description": fmt.Sprintf("[Routine] %s", r.Description),
			"type":        r.Type,
			"wallet":      r.Wallet,
			"createdBy":   ownerID,
			"status":      "PENDING",
			"routineId":   r.ID,
			"isDeleted":   false,
			"isTransfer":  false,
			"createdAt":   now,
			"updatedAt":   now,
		}
		if r.TargetWallet != nil {
			txDoc["targetWallet"] = *r.TargetWallet
		}
		if r.Category != nil {
			txDoc["category"] = *r.Category
		}
		db.Col("transactions").InsertOne(ctx, txDoc)

		// Calculate next run
		nextRun := r.NextRun
		switch r.Frequency {
		case "WEEKLY":
			nextRun = nextRun.AddDate(0, 0, 7)
		case "MONTHLY":
			nextRun = nextRun.AddDate(0, 1, 0)
		case "QUARTERLY":
			nextRun = nextRun.AddDate(0, 3, 0)
		case "YEARLY":
			nextRun = nextRun.AddDate(1, 0, 0)
		}

		db.Col("routines").UpdateOne(ctx, bson.M{"_id": r.ID}, bson.M{"$set": bson.M{
			"nextRun": nextRun, "lastRun": now, "updatedAt": now,
		}})
		count++
	}
	return count, nil
}

func GetPendingTransactions(ownerID primitive.ObjectID) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("transactions").Find(ctx,
		bson.M{"createdBy": ownerID, "status": "PENDING", "isDeleted": false},
		options.Find().SetSort(bson.D{{Key: "date", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	var result []bson.M
	cursor.All(ctx, &result)
	if result == nil {
		result = []bson.M{}
	}
	return result, nil
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
