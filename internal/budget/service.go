package budget

import (
	"context"
	"math"
	"sort"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetAvailableGroups() ([]AvailableGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := db.Col("categories").Find(ctx, bson.M{
		"isDeleted": false,
		"type":      "EXPENSE",
		"group":     bson.M{"$exists": true, "$ne": ""},
	}, options.Find().SetProjection(bson.M{"group": 1, "bucket": 1, "icon": 1, "color": 1}))
	if err != nil {
		return nil, err
	}
	var cats []struct {
		Group  string `bson:"group"`
		Bucket string `bson:"bucket"`
		Icon   string `bson:"icon"`
		Color  string `bson:"color"`
	}
	cursor.All(ctx, &cats)

	seen := map[string]AvailableGroup{}
	for _, cat := range cats {
		if _, ok := seen[cat.Group]; !ok {
			t := cat.Bucket
			if t == "" {
				t = "NEEDS"
			}
			seen[cat.Group] = AvailableGroup{
				GroupName: cat.Group, Type: t,
				Icon: cat.Icon, Color: cat.Color,
			}
		}
	}

	bucketOrder := map[string]int{"NEEDS": 0, "WANTS": 1, "SAVINGS": 2}
	groups := make([]AvailableGroup, 0, len(seen))
	for _, g := range seen {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		d := bucketOrder[groups[i].Type] - bucketOrder[groups[j].Type]
		if d != 0 {
			return d < 0
		}
		return groups[i].GroupName < groups[j].GroupName
	})
	return groups, nil
}

func UpsertEnvelopes(userID primitive.ObjectID, req UpsertEnvelopesRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err := db.Col("monthlybudgets").UpdateOne(ctx,
		bson.M{"user": userID, "period": req.Period},
		bson.M{"$set": bson.M{
			"user": userID, "period": req.Period,
			"income": req.Income, "envelopes": req.Envelopes,
			"isDeleted": false, "updatedAt": now,
		}, "$setOnInsert": bson.M{"createdAt": now}},
		options.Update().SetUpsert(true),
	)
	return err
}

func GetBudgetOverview(userID primitive.ObjectID, period string) (*BudgetOverview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Load or carry-over budget
	var budget MonthlyBudget
	err := db.Col("monthlybudgets").FindOne(ctx, bson.M{
		"user": userID, "period": period, "isDeleted": false,
	}).Decode(&budget)

	if err == mongo.ErrNoDocuments || len(budget.Envelopes) == 0 {
		// Carry-over from last month
		var lastBudget MonthlyBudget
		err2 := db.Col("monthlybudgets").FindOne(ctx,
			bson.M{"user": userID, "isDeleted": false},
			options.FindOne().SetSort(bson.D{{Key: "period", Value: -1}}),
		).Decode(&lastBudget)

		if err2 == nil && len(lastBudget.Envelopes) > 0 {
			carried := make([]Envelope, len(lastBudget.Envelopes))
			for i, e := range lastBudget.Envelopes {
				carried[i] = Envelope{Name: e.Name, CategoryIDs: e.CategoryIDs, Icon: e.Icon, Color: e.Color, Limit: e.Limit}
			}
			now := time.Now()
			db.Col("monthlybudgets").UpdateOne(ctx,
				bson.M{"user": userID, "period": period},
				bson.M{"$set": bson.M{
					"user": userID, "period": period,
					"income": lastBudget.Income, "envelopes": carried,
					"isDeleted": false, "updatedAt": now,
				}, "$setOnInsert": bson.M{"createdAt": now}},
				options.Update().SetUpsert(true),
			)
			budget.Envelopes = carried
			budget.Income = lastBudget.Income
		}
	}

	// Get wallet IDs for this user
	wCursor, _ := db.Col("wallets").Find(ctx, bson.M{"owner": userID, "isDeleted": false}, options.Find().SetProjection(bson.M{"_id": 1}))
	var wallets []struct{ ID primitive.ObjectID `bson:"_id"` }
	wCursor.All(ctx, &wallets)
	walletIDs := make([]primitive.ObjectID, len(wallets))
	for i, w := range wallets {
		walletIDs[i] = w.ID
	}

	// Parse period dates
	t, _ := time.Parse("2006-01", period)
	startDate := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// Aggregate expense by category
	expCursor, _ := db.Col("transactions").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"wallet":    bson.M{"$in": walletIDs},
			"date":      bson.M{"$gte": startDate, "$lte": endDate},
			"type":      "EXPENSE",
			"isDeleted": false,
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$category"},
			{Key: "totalSpent", Value: bson.D{{Key: "$sum", Value: "$amount"}}},
		}}},
	})
	var expResult []struct {
		ID         primitive.ObjectID `bson:"_id"`
		TotalSpent float64            `bson:"totalSpent"`
	}
	expCursor.All(ctx, &expResult)
	spendingByCat := map[string]float64{}
	for _, e := range expResult {
		spendingByCat[e.ID.Hex()] = e.TotalSpent
	}

	// Fetch all categories to determine Bucket for Needs/Wants total
	catCursor, _ := db.Col("categories").Find(ctx, bson.M{"isDeleted": false}, options.Find().SetProjection(bson.M{"_id": 1, "bucket": 1}))
	var allCats []struct {
		ID     primitive.ObjectID `bson:"_id"`
		Bucket string             `bson:"bucket"`
	}
	catCursor.All(ctx, &allCats)
	catToBucket := map[string]string{}
	for _, c := range allCats {
		catToBucket[c.ID.Hex()] = c.Bucket
	}

	// Calculate Needs and Wants totals from all spending
	var totalNeeds, totalWants float64
	for catID, amt := range spendingByCat {
		if catToBucket[catID] == "NEEDS" {
			totalNeeds += amt
		} else if catToBucket[catID] == "WANTS" {
			totalWants += amt
		}
	}

	// Realized income
	incCursor, _ := db.Col("transactions").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"wallet": bson.M{"$in": walletIDs},
			"date":   bson.M{"$gte": startDate, "$lte": endDate},
			"type":   "INCOME", "isDeleted": false,
		}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: nil}, {Key: "totalIncome", Value: bson.D{{Key: "$sum", Value: "$amount"}}}}}},
	})
	var incResult []struct{ TotalIncome float64 `bson:"totalIncome"` }
	incCursor.All(ctx, &incResult)
	realizedIncome := 0.0
	if len(incResult) > 0 {
		realizedIncome = incResult[0].TotalIncome
	}

	// Days remaining
	now := time.Now()
	daysRemaining := 0
	currentPeriod := now.Format("2006-01")
	if period == currentPeriod {
		d := int(endDate.Sub(now).Hours() / 24)
		if d < 1 {
			d = 1
		}
		daysRemaining = d
	}

	// Build envelopes
	totalBudget := 0.0
	budgetedCategoryIDs := map[string]bool{}
	envelopes := make([]EnvelopeOverview, 0, len(budget.Envelopes))

	for _, env := range budget.Envelopes {
		if env.Name == "" {
			continue // Skip legacy or invalid envelopes
		}
		spent := 0.0
		for _, catID := range env.CategoryIDs {
			spent += spendingByCat[catID]
			budgetedCategoryIDs[catID] = true
		}

		remaining := math.Max(0, env.Limit-spent)
		pct := 0.0
		if env.Limit > 0 {
			pct = math.Min(100, (spent/env.Limit)*100)
		}
		safe := 0.0
		if daysRemaining > 0 && remaining > 0 {
			safe = remaining / float64(daysRemaining)
		}
		totalBudget += env.Limit
		envelopes = append(envelopes, EnvelopeOverview{
			Name: env.Name, Icon: env.Icon, Color: env.Color,
			Limit: env.Limit, Spent: spent, Remaining: remaining,
			Percent: pct, SafeToSpendToday: safe,
			CategoryIDs: env.CategoryIDs,
		})
	}

	sort.Slice(envelopes, func(i, j int) bool {
		return envelopes[i].Name < envelopes[j].Name
	})

	// Unbudgeted spending
	unbudgetedSpent := 0.0
	totalSpent := 0.0
	for catID, amt := range spendingByCat {
		totalSpent += amt
		if !budgetedCategoryIDs[catID] {
			unbudgetedSpent += amt
		}
	}

	return &BudgetOverview{
		Period:          period,
		Income:          budget.Income,
		RealizedIncome:  realizedIncome,
		Envelopes:       envelopes,
		UnbudgetedSpent: unbudgetedSpent,
		TotalBudget:     totalBudget,
		TotalSpent:      totalSpent,
		TotalNeeds:      totalNeeds,
		TotalWants:      totalWants,
		DaysRemaining:   daysRemaining,
	}, nil
}
