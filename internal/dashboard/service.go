package dashboard

import (
	"context"
	"strings"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DailyTrendItem struct {
	Label   string  `json:"label"`
	Date    string  `json:"date"`
	Expense float64 `json:"expense"`
}

type MonthlyTrendItem struct {
	Label   string  `json:"label"`
	Income  float64 `json:"income"`
	Expense float64 `json:"expense"`
}

type FixedVsVariable struct {
	Fixed    float64 `json:"fixed"`
	Variable float64 `json:"variable"`
	Ratio    float64 `json:"ratio"`
}

type LiabilityItem struct {
	// We'll skip liability items for now if not implemented, but add structure
}

type Summary struct {
	NetWorth         float64            `json:"netWorth"`
	TotalIncome      float64            `json:"totalIncome"`
	TotalFixedIncome float64            `json:"totalFixedIncome"`
	TotalExpense     float64            `json:"totalExpense"`
	Net              float64            `json:"net"`
	Period           string             `json:"period"`
	DailyTrend       []DailyTrendItem   `json:"dailyTrend"`
	MonthlyTrend     []MonthlyTrendItem `json:"monthlyTrend"`
	FixedVsVariable  FixedVsVariable    `json:"fixedVsVariable"`
}

func GetSummary(userID primitive.ObjectID, period string, startDateStr string, endDateStr string, ownerParam string, walletIdParam string) (*Summary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if period == "" && startDateStr == "" {
		period = time.Now().Format("2006-01")
	}

	filter := bson.M{"isDeleted": false}

	if ownerParam == "" {
		filter["owner"] = userID
	} else if strings.ToUpper(ownerParam) != "ALL" {
		ownerID, err := resolveUsername(ownerParam)
		if err == nil {
			filter["owner"] = ownerID
		} else {
			// If resolving fails, we can just fallback to userID or no owner
			filter["owner"] = userID
		}
	}
	// If ownerParam is "ALL", we don't set "owner" filter so it fetches all wallets.

	if walletIdParam != "" {
		if oid, err := primitive.ObjectIDFromHex(walletIdParam); err == nil {
			filter["_id"] = oid
		}
	}
	wCursor, err := db.Col("wallets").Find(ctx, filter, options.Find().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		return nil, err
	}
	var wallets []struct {
		ID primitive.ObjectID `bson:"_id"`
	}
	if err = wCursor.All(ctx, &wallets); err != nil {
		return nil, err
	}
	walletIDs := make([]primitive.ObjectID, len(wallets))
	for i, w := range wallets {
		walletIDs[i] = w.ID
	}

	// Parse period or explicit date range
	var startDate, endDate time.Time

	if startDateStr != "" && endDateStr != "" {
		// Use explicit dates (e.g. 2026-01-31T17:00:00.000Z)
		t1, err1 := time.Parse(time.RFC3339, startDateStr)
		t2, err2 := time.Parse(time.RFC3339, endDateStr)
		if err1 == nil && err2 == nil {
			startDate = t1
			endDate = t2
		} else {
			// Fallback if parsing fails
			t, _ := time.Parse("2006-01", period)
			startDate = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
		}
	} else {
		// Default to period (1 month span)
		t, _ := time.Parse("2006-01", period)
		startDate = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)
	}

	// Wallet -> owner lookup covering every wallet (not just the scoped
	// ones) so a TRANSFER's counterparty owner can be resolved even when it
	// falls outside walletIDs.
	allWalletsCursor, err := db.Col("wallets").Find(ctx, bson.M{"isDeleted": false}, options.Find().SetProjection(bson.M{"_id": 1, "owner": 1}))
	if err != nil {
		return nil, err
	}
	var allWalletDocs []struct {
		ID    primitive.ObjectID `bson:"_id"`
		Owner primitive.ObjectID `bson:"owner"`
	}
	if err := allWalletsCursor.All(ctx, &allWalletDocs); err != nil {
		return nil, err
	}
	ownerByWallet := make(map[primitive.ObjectID]primitive.ObjectID, len(allWalletDocs))
	for _, w := range allWalletDocs {
		ownerByWallet[w.ID] = w.Owner
	}
	inScope := make(map[primitive.ObjectID]bool, len(walletIDs))
	for _, id := range walletIDs {
		inScope[id] = true
	}

	// Category -> flexibility lookup, used to split INCOME into a "fixed"
	// (reliable, e.g. salary) subset for borrowing-power calculations that
	// shouldn't be inflated by one-off bonus/freelance income.
	catCursor, err := db.Col("categories").Find(ctx, bson.M{"isDeleted": false}, options.Find().SetProjection(bson.M{"_id": 1, "flexibility": 1}))
	if err != nil {
		return nil, err
	}
	var catDocs []struct {
		ID          primitive.ObjectID `bson:"_id"`
		Flexibility string             `bson:"flexibility"`
	}
	if err := catCursor.All(ctx, &catDocs); err != nil {
		return nil, err
	}
	categoryFlexByID := make(map[primitive.ObjectID]string, len(catDocs))
	for _, c := range catDocs {
		categoryFlexByID[c.ID] = c.Flexibility
	}

	// Income/Expense for this period. INCOME/EXPENSE transactions count
	// normally. TRANSFER only counts when it crosses between wallets with
	// different owners -- an expense for the sender, income for the
	// receiver -- since that's real money leaving one person's control.
	// A transfer between two wallets owned by the same person is just
	// internal reallocation and is never counted as either.
	txCursor, err := db.Col("transactions").Find(ctx, bson.M{
		"$or": []bson.M{
			{"wallet": bson.M{"$in": walletIDs}},
			{"targetWallet": bson.M{"$in": walletIDs}},
		},
		"date":      bson.M{"$gte": startDate, "$lte": endDate},
		"isDeleted": false,
		"status":    bson.M{"$ne": "PENDING"},
	})
	if err != nil {
		return nil, err
	}
	var txs []struct {
		Type         string              `bson:"type"`
		Amount       float64             `bson:"amount"`
		Wallet       primitive.ObjectID  `bson:"wallet"`
		TargetWallet *primitive.ObjectID `bson:"targetWallet,omitempty"`
		Category     *primitive.ObjectID `bson:"category,omitempty"`
	}
	if err := txCursor.All(ctx, &txs); err != nil {
		return nil, err
	}

	totalIncome, totalFixedIncome, totalExpense := 0.0, 0.0, 0.0
	for _, tx := range txs {
		switch tx.Type {
		case "INCOME":
			if inScope[tx.Wallet] {
				totalIncome += tx.Amount
				if tx.Category != nil && categoryFlexByID[*tx.Category] == "FIXED" {
					totalFixedIncome += tx.Amount
				}
			}
		case "EXPENSE":
			if inScope[tx.Wallet] {
				totalExpense += tx.Amount
			}
		case "TRANSFER":
			if tx.TargetWallet == nil {
				continue
			}
			sourceOwner, hasSourceOwner := ownerByWallet[tx.Wallet]
			targetOwner, hasTargetOwner := ownerByWallet[*tx.TargetWallet]
			if !hasSourceOwner || !hasTargetOwner || sourceOwner == targetOwner {
				continue // same-owner reallocation, or an unresolvable wallet -- not counted
			}
			if inScope[tx.Wallet] {
				totalExpense += tx.Amount
			}
			if inScope[*tx.TargetWallet] {
				totalIncome += tx.Amount
			}
		}
	}

	nwFilter := bson.M{}
	for k, v := range filter {
		nwFilter[k] = v
	}
	nwFilter["type"] = bson.M{"$ne": "LIABILITY"}

	// Net worth: sum of all wallet current balances (initial + transactions)
	netWorthAgg, err := db.Col("wallets").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: nwFilter}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "transactions"},
			{Key: "let", Value: bson.D{{Key: "walletId", Value: "$_id"}}},
			{Key: "pipeline", Value: mongo.Pipeline{
				{{Key: "$match", Value: bson.D{
					{Key: "$expr", Value: bson.D{{Key: "$and", Value: bson.A{
						bson.D{{Key: "$eq", Value: bson.A{"$isDeleted", false}}},
						bson.D{{Key: "$ne", Value: bson.A{"$status", "PENDING"}}},
						bson.D{{Key: "$eq", Value: bson.A{"$wallet", "$$walletId"}}},
					}}}},
				}}},
			}},
			{Key: "as", Value: "txns"},
		}}},
		{{Key: "$addFields", Value: bson.D{
			{Key: "impact", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$map", Value: bson.D{
				{Key: "input", Value: "$txns"}, {Key: "as", Value: "t"},
				{Key: "in", Value: bson.D{{Key: "$cond", Value: bson.A{
					bson.D{{Key: "$eq", Value: bson.A{"$$t.type", "EXPENSE"}}},
					bson.D{{Key: "$multiply", Value: bson.A{"$$t.amount", -1}}},
					"$$t.amount",
				}}}},
			}}}}}},
		}}},
		{{Key: "$addFields", Value: bson.D{{Key: "balance", Value: bson.D{{Key: "$add", Value: bson.A{"$initialBalance", "$impact"}}}}}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: nil}, {Key: "netWorth", Value: bson.D{{Key: "$sum", Value: "$balance"}}}}}},
	})
	if err != nil {
		return nil, err
	}
	var nwResult []struct {
		NetWorth float64 `bson:"netWorth"`
	}
	if err = netWorthAgg.All(ctx, &nwResult); err != nil {
		return nil, err
	}
	netWorth := 0.0
	if len(nwResult) > 0 {
		netWorth = nwResult[0].NetWorth
	}

	// Daily Trend (for current month)
	dailyAgg, err := db.Col("transactions").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"wallet":    bson.M{"$in": walletIDs},
			"date":      bson.M{"$gte": startDate, "$lte": endDate},
			"isDeleted": false,
			"type":      "EXPENSE", // Only match expenses for daily trend
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{{Key: "format", Value: "%Y-%m-%d"}, {Key: "date", Value: "$date"}}}}},
			{Key: "expense", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var dailyTrend []DailyTrendItem
	var dailyRaw []struct {
		ID      string  `bson:"_id"`
		Expense float64 `bson:"expense"`
	}
	if err = dailyAgg.All(ctx, &dailyRaw); err != nil {
		return nil, err
	}
	for _, d := range dailyRaw {
		// Extract day from YYYY-MM-DD
		label := d.ID
		if len(d.ID) >= 10 {
			label = d.ID[8:10] // just the day part
		}
		dailyTrend = append(dailyTrend, DailyTrendItem{
			Label:   label,
			Date:    d.ID,
			Expense: d.Expense,
		})
	}

	// Monthly Trend (last 6 months)
	sixMonthsAgo := startDate.AddDate(0, -6, 0)
	monthlyAgg, err := db.Col("transactions").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"wallet":    bson.M{"$in": walletIDs},
			"date":      bson.M{"$gte": sixMonthsAgo, "$lte": endDate},
			"isDeleted": false,
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{{Key: "format", Value: "%Y-%m"}, {Key: "date", Value: "$date"}}}}},
			{Key: "income", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{
				bson.D{{Key: "$eq", Value: bson.A{"$type", "INCOME"}}}, "$amount", 0,
			}}}}}},
			{Key: "expense", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{
				bson.D{{Key: "$eq", Value: bson.A{"$type", "EXPENSE"}}}, "$amount", 0,
			}}}}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	})
	if err != nil {
		return nil, err
	}
	var monthlyTrend []MonthlyTrendItem
	var monthlyRaw []struct {
		ID      string  `bson:"_id"`
		Income  float64 `bson:"income"`
		Expense float64 `bson:"expense"`
	}
	if err = monthlyAgg.All(ctx, &monthlyRaw); err != nil {
		return nil, err
	}
	for _, m := range monthlyRaw {
		monthlyTrend = append(monthlyTrend, MonthlyTrendItem{
			Label:   m.ID,
			Income:  m.Income,
			Expense: m.Expense,
		})
	}

	// Fixed vs Variable (for current month)
	fvAgg, err := db.Col("transactions").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"wallet":    bson.M{"$in": walletIDs},
			"date":      bson.M{"$gte": startDate, "$lte": endDate},
			"isDeleted": false,
			"type":      "EXPENSE",
		}}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "categories"},
			{Key: "localField", Value: "category"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "cat"},
		}}},
		{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$cat"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$cat.flexibility"},
			{Key: "total", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	})

	fixedVsVariable := FixedVsVariable{}
	if err == nil {
		var fvRaw []struct {
			ID    string  `bson:"_id"`
			Total float64 `bson:"total"`
		}
		if fvAgg.All(ctx, &fvRaw) == nil {
			for _, item := range fvRaw {
				if item.ID == "FIXED" {
					fixedVsVariable.Fixed = item.Total
				} else if item.ID == "VARIABLE" {
					fixedVsVariable.Variable = item.Total
				}
			}
			totalFV := fixedVsVariable.Fixed + fixedVsVariable.Variable
			if totalFV > 0 {
				fixedVsVariable.Ratio = (fixedVsVariable.Fixed / totalFV) * 100
			}
		}
	}

	return &Summary{
		NetWorth:         netWorth,
		TotalIncome:      totalIncome,
		TotalFixedIncome: totalFixedIncome,
		TotalExpense:     totalExpense,
		Net:              totalIncome - totalExpense,
		Period:           period,
		DailyTrend:       dailyTrend,
		MonthlyTrend:     monthlyTrend,
		FixedVsVariable:  fixedVsVariable,
	}, nil
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
		return primitive.NilObjectID, err
	}
	return user.ID, nil
}
