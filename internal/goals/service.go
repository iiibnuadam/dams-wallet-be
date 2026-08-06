package goals

import (
	"context"
	"math"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/internal/transactions"
	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetGoals(owner string) ([]GoalWithStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := bson.M{}
	if owner != "" && owner != "ALL" {
		if oid, err := primitive.ObjectIDFromHex(owner); err == nil {
			query["$or"] = bson.A{bson.M{"owner": oid}, bson.M{"visibility": "SHARED"}}
		} else if ownerID, err := resolveUsername(owner); err == nil {
			query["$or"] = bson.A{bson.M{"owner": ownerID}, bson.M{"visibility": "SHARED"}}
		}
	}

	cursor, err := db.Col("goals").Find(ctx, query, options.Find().SetSort(bson.D{{Key: "targetDate", Value: 1}}))
	if err != nil {
		return nil, err
	}
	var goals []Goal
	cursor.All(ctx, &goals)

	results := make([]GoalWithStats, 0, len(goals))
	for _, g := range goals {
		items, _ := db.Col("goalitems").Find(ctx, bson.M{"goalId": g.ID}, options.Find().SetProjection(bson.M{"_id": 1, "estimatedAmount": 1}))
		var itemDocs []struct {
			ID              primitive.ObjectID `bson:"_id"`
			EstimatedAmount float64            `bson:"estimatedAmount"`
		}
		items.All(ctx, &itemDocs)

		totalEst := 0.0
		itemIDs := make([]primitive.ObjectID, len(itemDocs))
		for i, it := range itemDocs {
			totalEst += it.EstimatedAmount
			itemIDs[i] = it.ID
		}

		txAgg, err := db.Col("transactions").Aggregate(ctx, mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"goalItem": bson.M{"$in": itemIDs}, "isDeleted": false, "type": "EXPENSE"}}},
			{{Key: "$group", Value: bson.D{{Key: "_id", Value: nil}, {Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}}}}},
		})
		var txResult []struct{ Total float64 `bson:"total"` }
		if err == nil {
			txAgg.All(ctx, &txResult)
		}
		totalActual := 0.0
		if len(txResult) > 0 {
			totalActual = txResult[0].Total
		}

		results = append(results, GoalWithStats{
			Goal: g, TotalEstimated: totalEst, TotalActual: totalActual,
			Pacing: computePacing(g, totalEst, totalActual),
		})
	}
	return results, nil
}

const goalFallingBehindThresholdPct = 10.0

// computePacing compares elapsed-time% (CreatedAt -> TargetDate) against
// Rupiah-progress% (TotalActual/TotalEstimated) to flag goals losing pace.
func computePacing(g Goal, totalEst, totalActual float64) GoalPacing {
	now := time.Now()
	totalDuration := g.TargetDate.Sub(g.CreatedAt)
	monthsRemaining := int(time.Until(g.TargetDate).Hours() / 24 / 30)

	if g.IsCompleted || totalDuration <= 0 {
		return GoalPacing{MonthsRemaining: monthsRemaining}
	}

	expectedPct := 0.0
	if elapsed := now.Sub(g.CreatedAt); elapsed > 0 {
		expectedPct = math.Min(100, (float64(elapsed)/float64(totalDuration))*100)
	}

	actualPct := 0.0
	if totalEst > 0 {
		actualPct = math.Min(100, (totalActual/totalEst)*100)
	}

	return GoalPacing{
		ExpectedProgressPercent: expectedPct,
		ActualProgressPercent:   actualPct,
		IsFallingBehind:         now.Before(g.TargetDate) && (expectedPct-actualPct) > goalFallingBehindThresholdPct,
		MonthsRemaining:         monthsRemaining,
	}
}

func CreateGoal(req CreateGoalRequest, ownerID primitive.ObjectID) (*Goal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	date, _ := time.Parse(time.RFC3339, req.TargetDate)
	vis := req.Visibility
	if vis == "" {
		vis = "SHARED"
	}
	now := time.Now()
	doc := bson.M{
		"name": req.Name, "owner": ownerID, "targetDate": date,
		"visibility": vis, "sharedWith": req.SharedWith,
		"color": req.Color, "icon": req.Icon,
		"groups": []interface{}{}, "createdAt": now, "updatedAt": now,
	}

	res, err := db.Col("goals").InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	var g Goal
	db.Col("goals").FindOne(ctx, bson.M{"_id": res.InsertedID}).Decode(&g)
	return &g, nil
}

func GetGoalByID(id string) (*GoalDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var goal Goal
	if err := db.Col("goals").FindOne(ctx, bson.M{"_id": oid}).Decode(&goal); err != nil {
		return nil, err
	}

	// Items with actual amounts
	itemCursor, err := db.Col("goalitems").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "goal", Value: id}}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "transactions"},
			{Key: "let", Value: bson.D{{Key: "itemId", Value: "$_id"}}},
			{Key: "pipeline", Value: mongo.Pipeline{
				{{Key: "$match", Value: bson.D{{Key: "$expr", Value: bson.D{{Key: "$eq", Value: bson.A{"$goalItem", "$$itemId"}}}}, {Key: "isDeleted", Value: false}}}},
				{{Key: "$group", Value: bson.D{{Key: "_id", Value: nil}, {Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}}}}},
			}},
			{Key: "as", Value: "txInfo"},
		}}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "actualAmount", Value: bson.D{{Key: "$ifNull", Value: bson.A{
				bson.D{{Key: "$arrayElemAt", Value: bson.A{"$txInfo.total", 0}}}, 0,
			}}}},
		}}},
		{{Key: "$project", Value: bson.D{{Key: "txInfo", Value: 0}}}},
	})
	if err != nil {
		return nil, err
	}
	var items []GoalItem
	if err := itemCursor.All(ctx, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []GoalItem{}
	}

	var history []transactions.Transaction = []transactions.Transaction{}
	var itemIDs []primitive.ObjectID
	for _, it := range items {
		itemIDs = append(itemIDs, it.ID)
	}

	if len(itemIDs) > 0 {
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"goalItem": bson.M{"$in": itemIDs}, "isDeleted": false}}},
			{{Key: "$sort", Value: bson.D{{Key: "date", Value: -1}}}},
		}
		txCursor, err := db.Col("transactions").Aggregate(ctx, pipeline)
		if err == nil {
			txCursor.All(ctx, &history)
		}
		if history == nil {
			history = []transactions.Transaction{}
		}
	}

	return &GoalDetail{Goal: goal, Items: items, History: history}, nil
}

func UpdateGoal(id string, update bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, _ := primitive.ObjectIDFromHex(id)
	update["updatedAt"] = time.Now()
	_, err := db.Col("goals").UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": update})
	return err
}

func DeleteGoal(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, _ := primitive.ObjectIDFromHex(id)
	db.Col("goalitems").DeleteMany(ctx, bson.M{"goalId": oid})
	_, err := db.Col("goals").DeleteOne(ctx, bson.M{"_id": oid})
	return err
}

func AddGoalItem(goalID string, req CreateGoalItemRequest) (*GoalItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gid, _ := primitive.ObjectIDFromHex(goalID)
	now := time.Now()
	doc := bson.M{
		"goalId": gid, "name": req.Name,
		"estimatedAmount": req.EstimatedAmount, "isCompleted": false,
		"createdAt": now, "updatedAt": now,
	}
	if req.GroupID != "" {
		if goid, err := primitive.ObjectIDFromHex(req.GroupID); err == nil {
			doc["groupId"] = goid
		}
	}
	res, err := db.Col("goalitems").InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	var item GoalItem
	db.Col("goalitems").FindOne(ctx, bson.M{"_id": res.InsertedID}).Decode(&item)
	return &item, nil
}

func UpdateGoalItem(itemID string, update bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, _ := primitive.ObjectIDFromHex(itemID)
	update["updatedAt"] = time.Now()
	_, err := db.Col("goalitems").UpdateOne(ctx, bson.M{"_id": oid}, bson.M{"$set": update})
	return err
}

func DeleteGoalItem(itemID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, _ := primitive.ObjectIDFromHex(itemID)
	_, err := db.Col("goalitems").DeleteOne(ctx, bson.M{"_id": oid})
	return err
}

func AddGroup(goalID string, req GroupRequest) (*Goal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	oid, _ := primitive.ObjectIDFromHex(goalID)
	groupDoc := bson.M{"_id": primitive.NewObjectID(), "name": req.Name, "color": req.Color, "icon": req.Icon}
	if req.ParentGroupID != "" && req.ParentGroupID != "ROOT" {
		if pgid, err := primitive.ObjectIDFromHex(req.ParentGroupID); err == nil {
			groupDoc["parentGroupId"] = pgid
		}
	}
	_, err := db.Col("goals").UpdateOne(ctx, bson.M{"_id": oid},
		bson.M{"$push": bson.M{"groups": groupDoc}, "$set": bson.M{"updatedAt": time.Now()}})
	if err != nil {
		return nil, err
	}
	var g Goal
	db.Col("goals").FindOne(ctx, bson.M{"_id": oid}).Decode(&g)
	return &g, nil
}

func UpdateGroup(goalID, groupID string, req GroupRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	goid, _ := primitive.ObjectIDFromHex(goalID)
	ggoid, _ := primitive.ObjectIDFromHex(groupID)

	setFields := bson.M{"updatedAt": time.Now()}
	if req.Name != "" {
		setFields["groups.$[g].name"] = req.Name
	}
	if req.Color != "" {
		setFields["groups.$[g].color"] = req.Color
	}
	if req.Icon != "" {
		setFields["groups.$[g].icon"] = req.Icon
	}

	_, err := db.Col("goals").UpdateOne(ctx, bson.M{"_id": goid},
		bson.M{"$set": setFields},
		options.Update().SetArrayFilters(options.ArrayFilters{
			Filters: []interface{}{bson.M{"g._id": ggoid}},
		}),
	)
	return err
}

func DeleteGroup(goalID, groupID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	goid, _ := primitive.ObjectIDFromHex(goalID)
	ggoid, _ := primitive.ObjectIDFromHex(groupID)
	_, err := db.Col("goals").UpdateOne(ctx, bson.M{"_id": goid},
		bson.M{"$pull": bson.M{"groups": bson.M{"_id": ggoid}}, "$set": bson.M{"updatedAt": time.Now()}})
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
