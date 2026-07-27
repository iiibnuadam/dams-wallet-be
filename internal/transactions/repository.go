package transactions

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

func buildQuery(f ListFilters) (bson.M, error) {
	query := bson.M{"isDeleted": false}

	// Date filter
	mode := f.Mode
	if mode == "" {
		mode = "MONTH"
	}
	switch mode {
	case "MONTH":
		monthStr := f.Month
		if monthStr == "" {
			monthStr = time.Now().Format("2006-01")
		}
		t, _ := time.Parse("2006-01", monthStr)
		start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
		query["date"] = bson.M{"$gte": start, "$lte": end}
	case "RANGE", "PRESET":
		if f.StartDate != "" && f.EndDate != "" {
			layout := time.RFC3339
			start, err1 := time.Parse(layout, f.StartDate)
			end, err2 := time.Parse(layout, f.EndDate)
			if err1 == nil && err2 == nil {
				query["date"] = bson.M{"$gte": start, "$lte": end}
			}
		}
	case "ALL":
		// no date filter
	}

	// Type filter
	if f.Type != "" && f.Type != "ALL" {
		switch f.Type {
		case "TRANSFER":
			query["$or"] = bson.A{bson.M{"type": "TRANSFER"}, bson.M{"isTransfer": true}}
		case "GOAL":
			query["goalItem"] = bson.M{"$exists": true, "$ne": nil}
		default:
			query["type"] = f.Type
		}
	}

	// Wallet filter
	if f.WalletID != "" && f.WalletID != "ALL" {
		if oid, err := primitive.ObjectIDFromHex(f.WalletID); err == nil {
			query["$or"] = bson.A{bson.M{"wallet": oid}, bson.M{"targetWallet": oid}}
		}
	}

	// Category filter
	if len(f.CategoryIDs) > 0 {
		oids := make([]primitive.ObjectID, 0, len(f.CategoryIDs))
		for _, cid := range f.CategoryIDs {
			if oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(cid)); err == nil {
				oids = append(oids, oid)
			}
		}
		if len(oids) > 0 {
			query["category"] = bson.M{"$in": oids}
		}
	}

	// Search
	if f.Q != "" {
		query["description"] = bson.M{"$regex": f.Q, "$options": "i"}
	}

	// Amount filter
	if f.MinAmount > 0 || f.MaxAmount > 0 {
		amtFilter := bson.M{}
		if f.MinAmount > 0 {
			amtFilter["$gte"] = f.MinAmount
		}
		if f.MaxAmount > 0 {
			amtFilter["$lte"] = f.MaxAmount
		}
		query["amount"] = amtFilter
	}

	// Goal filter
	if f.GoalID != "" && f.GoalID != "ALL" {
		if goid, err := primitive.ObjectIDFromHex(f.GoalID); err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cursor, err := db.Col("goalitems").Find(ctx, bson.M{"goalId": goid}, options.Find().SetProjection(bson.M{"_id": 1}))
			if err == nil {
				var items []struct{ ID primitive.ObjectID `bson:"_id"` }
				cursor.All(ctx, &items)
				itemIDs := make([]primitive.ObjectID, len(items))
				for i, it := range items {
					itemIDs[i] = it.ID
				}
				query["goalItem"] = bson.M{"$in": itemIDs}
			}
		}
	}

	// View/owner filter
	if f.View != "" && f.View != "ALL" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var user struct {
			ID primitive.ObjectID `bson:"_id"`
		}
		err := db.Col("users").FindOne(ctx, bson.M{
			"username": bson.M{"$regex": fmt.Sprintf("^%s$", strings.ToUpper(f.View)), "$options": "i"},
		}).Decode(&user)
		if err == nil {
			var wallets []struct{ ID primitive.ObjectID `bson:"_id"` }
			wCursor, _ := db.Col("wallets").Find(ctx, bson.M{"owner": user.ID, "isDeleted": false}, options.Find().SetProjection(bson.M{"_id": 1}))
			wCursor.All(ctx, &wallets)
			wIDs := make([]primitive.ObjectID, len(wallets))
			for i, w := range wallets {
				wIDs[i] = w.ID
			}
			query["wallet"] = bson.M{"$in": wIDs}
		}
	}

	return query, nil
}

func listTransactions(f ListFilters) (*ListResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query, err := buildQuery(f)
	if err != nil {
		return nil, err
	}

	page := f.Page
	if page < 1 {
		page = 1
	}
	limit := f.Limit
	if limit < 1 {
		limit = 15
	}
	skip := int64((page - 1) * limit)

	// Summary aggregation
	summaryPipeline := mongo.Pipeline{
		{{Key: "$match", Value: query}},
		{{Key: "$facet", Value: bson.D{
			{Key: "totals", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: nil},
					{Key: "totalIncome", Value: bson.D{{Key: "$sum", Value: bson.D{
						{Key: "$cond", Value: bson.A{bson.D{{Key: "$eq", Value: bson.A{"$type", "INCOME"}}}, "$amount", 0}},
					}}}},
					{Key: "totalExpense", Value: bson.D{{Key: "$sum", Value: bson.D{
						{Key: "$cond", Value: bson.A{bson.D{{Key: "$eq", Value: bson.A{"$type", "EXPENSE"}}}, "$amount", 0}},
					}}}},
				}}},
			}},
			{Key: "byCategory", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: bson.D{
						{Key: "key", Value: bson.D{{Key: "$cond", Value: bson.A{
							bson.D{{Key: "$ifNull", Value: bson.A{"$goalItem", false}}},
							"GOAL", "$category",
						}}}},
						{Key: "type", Value: "$type"},
					}},
					{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
				}}},
				bson.D{{Key: "$lookup", Value: bson.D{
					{Key: "from", Value: "categories"},
					{Key: "localField", Value: "_id.key"},
					{Key: "foreignField", Value: "_id"},
					{Key: "as", Value: "categoryDoc"},
				}}},
				bson.D{{Key: "$project", Value: bson.D{
					{Key: "type", Value: "$_id.type"},
					{Key: "categoryName", Value: bson.D{{Key: "$cond", Value: bson.A{
						bson.D{{Key: "$eq", Value: bson.A{"$_id.key", "GOAL"}}},
						"Goals",
						bson.D{{Key: "$arrayElemAt", Value: bson.A{"$categoryDoc.name", 0}}},
					}}}},
					{Key: "categoryColor", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$categoryDoc.color", 0}}}},
					{Key: "categoryIcon", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$categoryDoc.icon", 0}}}},
					{Key: "total", Value: 1},
				}}},
				bson.D{{Key: "$sort", Value: bson.D{{Key: "total", Value: -1}}}},
			}},
		}}},
	}

	summaryCursor, err := db.Col("transactions").Aggregate(ctx, summaryPipeline)
	if err != nil {
		return nil, err
	}
	var summaryResults []struct {
		Totals []struct {
			TotalIncome  float64 `bson:"totalIncome"`
			TotalExpense float64 `bson:"totalExpense"`
		} `bson:"totals"`
		ByCategory []struct {
			Type          string  `bson:"type"`
			CategoryName  string  `bson:"categoryName"`
			CategoryColor string  `bson:"categoryColor"`
			CategoryIcon  string  `bson:"categoryIcon"`
			Total         float64 `bson:"total"`
		} `bson:"byCategory"`
	}
	summaryCursor.All(ctx, &summaryResults)

	summary := Summary{}
	if len(summaryResults) > 0 && len(summaryResults[0].Totals) > 0 {
		t := summaryResults[0].Totals[0]
		summary.TotalIncome = t.TotalIncome
		summary.TotalExpense = t.TotalExpense
		summary.Net = t.TotalIncome - t.TotalExpense
	}
	if len(summaryResults) > 0 {
		for _, c := range summaryResults[0].ByCategory {
			bd := CategoryBreakdown{Name: c.CategoryName, Value: c.Total, Color: c.CategoryColor, Icon: c.CategoryIcon}
			if c.Type == "INCOME" {
				found := false
				for i, existing := range summary.IncomeCategories {
					if existing.Name == bd.Name {
						summary.IncomeCategories[i].Value += bd.Value
						found = true
						break
					}
				}
				if !found {
					summary.IncomeCategories = append(summary.IncomeCategories, bd)
				}
			} else {
				found := false
				for i, existing := range summary.ExpenseCategories {
					if existing.Name == bd.Name {
						summary.ExpenseCategories[i].Value += bd.Value
						found = true
						break
					}
				}
				if !found {
					summary.ExpenseCategories = append(summary.ExpenseCategories, bd)
				}
			}
		}
	}
	if summary.IncomeCategories == nil {
		summary.IncomeCategories = []CategoryBreakdown{}
	}
	if summary.ExpenseCategories == nil {
		summary.ExpenseCategories = []CategoryBreakdown{}
	}

	if f.SummaryOnly {
		return &ListResult{
			Transactions: []PopulatedTransaction{},
			Pagination:   Pagination{CurrentPage: page, Limit: limit},
			Summary:      summary,
		}, nil
	}

	// Fetch paginated transactions with populate
	lookupPipeline := mongo.Pipeline{
		{{Key: "$match", Value: query}},
		{{Key: "$sort", Value: bson.D{{Key: "date", Value: -1}, {Key: "createdAt", Value: -1}}}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: int64(limit)}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "wallets"}, {Key: "localField", Value: "wallet"},
			{Key: "foreignField", Value: "_id"}, {Key: "as", Value: "walletDoc"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "wallets"}, {Key: "localField", Value: "targetWallet"},
			{Key: "foreignField", Value: "_id"}, {Key: "as", Value: "targetWalletDoc"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "categories"}, {Key: "localField", Value: "category"},
			{Key: "foreignField", Value: "_id"}, {Key: "as", Value: "categoryDoc"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "goalitems"},
			{Key: "let", Value: bson.D{{Key: "gId", Value: "$goalItem"}}},
			{Key: "pipeline", Value: mongo.Pipeline{
				{{Key: "$match", Value: bson.D{{Key: "$expr", Value: bson.D{{Key: "$eq", Value: bson.A{"$_id", "$$gId"}}}}}}},
				{{Key: "$lookup", Value: bson.D{
					{Key: "from", Value: "goals"},
					{Key: "localField", Value: "goalId"},
					{Key: "foreignField", Value: "_id"},
					{Key: "as", Value: "goalId"},
				}}},
				{{Key: "$unwind", Value: bson.D{
					{Key: "path", Value: "$goalId"},
					{Key: "preserveNullAndEmptyArrays", Value: true},
				}}},
				{{Key: "$addFields", Value: bson.D{
					{Key: "goalId", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$goalId", nil}}}},
				}}},
			}},
			{Key: "as", Value: "goalItemDoc"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"}, {Key: "localField", Value: "createdBy"},
			{Key: "foreignField", Value: "_id"}, {Key: "as", Value: "createdByDoc"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "transactions"}, {Key: "localField", Value: "relatedTransactionId"},
			{Key: "foreignField", Value: "_id"}, {Key: "as", Value: "relatedTransactionDoc"},
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "wallets"},
			{Key: "localField", Value: "relatedTransactionDoc.wallet"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "relatedTransactionWalletDoc"},
		}}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "wallet", Value: bson.D{{Key: "$ifNull", Value: bson.A{bson.D{{Key: "$arrayElemAt", Value: bson.A{"$walletDoc", 0}}}, nil}}}},
			{Key: "targetWallet", Value: bson.D{{Key: "$ifNull", Value: bson.A{bson.D{{Key: "$arrayElemAt", Value: bson.A{"$targetWalletDoc", 0}}}, nil}}}},
			{Key: "category", Value: bson.D{{Key: "$ifNull", Value: bson.A{bson.D{{Key: "$arrayElemAt", Value: bson.A{"$categoryDoc", 0}}}, nil}}}},
			{Key: "goalItem", Value: bson.D{{Key: "$ifNull", Value: bson.A{bson.D{{Key: "$arrayElemAt", Value: bson.A{"$goalItemDoc", 0}}}, nil}}}},
			{Key: "createdBy", Value: bson.D{{Key: "$ifNull", Value: bson.A{
				bson.D{{Key: "$arrayElemAt", Value: bson.A{"$createdByDoc.name", 0}}}, "Unknown",
			}}}},
			{Key: "relatedTransaction", Value: bson.D{{Key: "$cond", Value: bson.A{
				bson.D{{Key: "$gt", Value: bson.A{bson.D{{Key: "$size", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$relatedTransactionWalletDoc", bson.A{}}}}}}, 0}}},
				bson.D{
					{Key: "wallet", Value: bson.D{{Key: "$arrayElemAt", Value: bson.A{"$relatedTransactionWalletDoc", 0}}}},
				},
				nil,
			}}}},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "walletDoc", Value: 0},
			{Key: "targetWalletDoc", Value: 0},
			{Key: "categoryDoc", Value: 0},
			{Key: "goalItemDoc", Value: 0},
			{Key: "createdByDoc", Value: 0},
			{Key: "relatedTransactionDoc", Value: 0},
			{Key: "relatedTransactionWalletDoc", Value: 0},
		}}},
	}

	txCursor, err := db.Col("transactions").Aggregate(ctx, lookupPipeline)
	if err != nil {
		return nil, err
	}
	var txns []PopulatedTransaction
	if err := txCursor.All(ctx, &txns); err != nil {
		return nil, err
	}
	if txns == nil {
		txns = []PopulatedTransaction{}
	}

	totalItems, _ := db.Col("transactions").CountDocuments(ctx, query)
	totalPages := int((totalItems + int64(limit) - 1) / int64(limit))

	return &ListResult{
		Transactions: txns,
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  totalItems,
			Limit:       limit,
		},
		Summary: summary,
	}, nil
}

func createTransaction(doc bson.M) (primitive.ObjectID, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	doc["createdAt"] = time.Now()
	doc["updatedAt"] = time.Now()
	res, err := db.Col("transactions").InsertOne(ctx, doc)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

func updateTransaction(id primitive.ObjectID, update bson.M) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	update["updatedAt"] = time.Now()
	_, err := db.Col("transactions").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}
