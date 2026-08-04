package budget

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ErrDuplicateEnvelopeItem is returned by UpsertEnvelopes when the same
// (categoryId, owner) combination appears more than once across the
// request.
var ErrDuplicateEnvelopeItem = errors.New("duplicate category/owner combination")

// Budgeting stays monthly regardless of an envelope's Type -- Limit is
// always the same month-scoped figure, and Spent is always summed over the
// current month, exactly like before Types existed. Type only drives a
// purely informational "pacing" reference showing what that monthly Limit
// works out to in the envelope's own unit (e.g. "≈ Rp30.000/hari" for a
// Daily-paced coffee budget), computed in WIB (Asia/Jakarta) purely for the
// days-in-month figure.
var wib = mustLoadJakarta()

func mustLoadJakarta() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*60*60)
	}
	return loc
}

// pacingAmount converts a monthly Limit into the envelope's own Type unit:
// Daily divides by the actual number of days in the month, Weekly divides
// by 4 (an approximation -- there's no clean "weeks in a month"), Annual
// multiplies by 12. Monthly (or unrecognized) has no separate unit to show.
func pacingAmount(envType string, limit, daysInMonth float64) (unit string, amount float64) {
	if limit <= 0 {
		return "", 0
	}
	switch envType {
	case TypeDaily:
		days := daysInMonth
		if days <= 0 {
			days = 30
		}
		return "day", limit / days
	case TypeWeekly:
		return "week", limit / 4
	case TypeAnnual:
		return "year", limit * 12
	default:
		return "", 0
	}
}

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

func UpsertEnvelopes(req UpsertEnvelopesRequest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Normalize "ALL" -> the canonical OwnerAll ("") sentinel and default
	// Type -> Monthly, then reject exact duplicate (categoryId, owner)
	// combinations across the whole request. Type is purely a pacing
	// label now (spend is always tracked for the current month regardless
	// of Type), so a category used in one envelope can't also be used in
	// another just because they have different Types -- it would just show
	// the same Spent figure twice, not track anything separately.
	seen := map[string]bool{}
	for e, env := range req.Envelopes {
		envType := strings.ToUpper(env.Type)
		if envType == "" {
			envType = TypeMonthly
		}
		req.Envelopes[e].Type = envType
		for i, item := range env.Items {
			owner := strings.ToUpper(item.Owner)
			if owner == "ALL" {
				owner = OwnerAll
			}
			req.Envelopes[e].Items[i].Owner = owner
			key := item.CategoryID + ":" + owner
			if seen[key] {
				return fmt.Errorf("%w: %s", ErrDuplicateEnvelopeItem, key)
			}
			seen[key] = true
		}
	}

	now := time.Now()
	_, err := db.Col("monthlybudgets").UpdateOne(ctx,
		bson.M{"period": req.Period},
		bson.M{"$set": bson.M{
			"period": req.Period,
			"income": req.Income, "envelopes": req.Envelopes,
			"isDeleted": false, "updatedAt": now,
		}, "$setOnInsert": bson.M{"createdAt": now}},
		options.Update().SetUpsert(true),
	)
	return err
}

func GetBudgetOverview(period string) (*BudgetOverview, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Load or carry-over budget -- one shared document per period, covering
	// the whole household rather than whichever account is logged in.
	var budget MonthlyBudget
	err := db.Col("monthlybudgets").FindOne(ctx, bson.M{
		"period": period, "isDeleted": false,
	}).Decode(&budget)

	if err == mongo.ErrNoDocuments || len(budget.Envelopes) == 0 {
		// Carry-over from last month
		var lastBudget MonthlyBudget
		err2 := db.Col("monthlybudgets").FindOne(ctx,
			bson.M{"isDeleted": false},
			options.FindOne().SetSort(bson.D{{Key: "period", Value: -1}}),
		).Decode(&lastBudget)

		if err2 == nil && len(lastBudget.Envelopes) > 0 {
			carried := make([]Envelope, len(lastBudget.Envelopes))
			for i, e := range lastBudget.Envelopes {
				carried[i] = Envelope{Name: e.Name, Type: e.Type, Items: e.Items, Icon: e.Icon, Color: e.Color, Limit: e.Limit}
			}
			now := time.Now()
			db.Col("monthlybudgets").UpdateOne(ctx,
				bson.M{"period": period},
				bson.M{"$set": bson.M{
					"period": period,
					"income": lastBudget.Income, "envelopes": carried,
					"isDeleted": false, "updatedAt": now,
				}, "$setOnInsert": bson.M{"createdAt": now}},
				options.Update().SetUpsert(true),
			)
			budget.Envelopes = carried
			budget.Income = lastBudget.Income
		}
	}

	// Get every household wallet -- budget spend/limits are shared across
	// everyone, not scoped to whichever account made the request. Owner is
	// projected too, to split spend per household member below.
	wCursor, err := db.Col("wallets").Find(ctx, bson.M{"isDeleted": false}, options.Find().SetProjection(bson.M{"_id": 1, "owner": 1}))
	if err != nil {
		return nil, err
	}
	var wallets []struct {
		ID    primitive.ObjectID `bson:"_id"`
		Owner primitive.ObjectID `bson:"owner"`
	}
	if err := wCursor.All(ctx, &wallets); err != nil {
		return nil, err
	}
	walletIDs := make([]primitive.ObjectID, len(wallets))
	walletOwnerID := make(map[primitive.ObjectID]primitive.ObjectID, len(wallets))
	for i, w := range wallets {
		walletIDs[i] = w.ID
		walletOwnerID[w.ID] = w.Owner
	}

	// Username per owner (household member), so per-item spend can be split
	// by "ADAM"/"SASTI" -- an unrecognized owner just means that wallet's
	// spend never lands in a per-owner bucket, it's still counted overall.
	uCursor, err := db.Col("users").Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1, "username": 1}))
	if err != nil {
		return nil, err
	}
	var userDocs []struct {
		ID       primitive.ObjectID `bson:"_id"`
		Username string             `bson:"username"`
	}
	if err := uCursor.All(ctx, &userDocs); err != nil {
		return nil, err
	}
	usernameByOwnerID := make(map[primitive.ObjectID]string, len(userDocs))
	for _, u := range userDocs {
		usernameByOwnerID[u.ID] = strings.ToUpper(u.Username)
	}

	// Parse period dates (WIB) -- everything is scoped to this one month,
	// regardless of any envelope's Type.
	t, _ := time.Parse("2006-01", period)
	startDate := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, wib)
	endDate := startDate.AddDate(0, 1, 0)
	daysInMonth := endDate.Sub(startDate).Hours() / 24

	// Fetch all categories -- Bucket for Needs/Wants totals, Name/Icon/Color for
	// the per-envelope category breakdown.
	catCursor, err := db.Col("categories").Find(ctx, bson.M{"isDeleted": false}, options.Find().SetProjection(bson.M{"_id": 1, "bucket": 1, "name": 1, "icon": 1, "color": 1}))
	if err != nil {
		return nil, err
	}
	var allCats []struct {
		ID     primitive.ObjectID `bson:"_id"`
		Bucket string             `bson:"bucket"`
		Name   string             `bson:"name"`
		Icon   string             `bson:"icon"`
		Color  string             `bson:"color"`
	}
	if err := catCursor.All(ctx, &allCats); err != nil {
		return nil, err
	}
	catToBucket := map[string]string{}
	catByID := map[string]struct {
		Name  string
		Icon  string
		Color string
	}{}
	for _, c := range allCats {
		catToBucket[c.ID.Hex()] = c.Bucket
		catByID[c.ID.Hex()] = struct {
			Name  string
			Icon  string
			Color string
		}{c.Name, c.Icon, c.Color}
	}

	// Expense transactions for the period, fetched raw (not aggregated in
	// Mongo) so spend can be split both by category and by which
	// household member's wallet it came from, in a single pass.
	expCursor, err := db.Col("transactions").Find(ctx, bson.M{
		"wallet":    bson.M{"$in": walletIDs},
		"date":      bson.M{"$gte": startDate, "$lt": endDate},
		"type":      "EXPENSE",
		"isDeleted": false,
	}, options.Find().SetProjection(bson.M{"category": 1, "amount": 1, "wallet": 1}))
	if err != nil {
		return nil, err
	}
	var expTxs []struct {
		Category *primitive.ObjectID `bson:"category,omitempty"`
		Amount   float64             `bson:"amount"`
		Wallet   primitive.ObjectID  `bson:"wallet"`
	}
	if err := expCursor.All(ctx, &expTxs); err != nil {
		return nil, err
	}

	spendingByCatAll := map[string]float64{}
	spendingByCatOwner := map[string]map[string]float64{}
	var totalNeeds, totalWants float64
	for _, tx := range expTxs {
		if tx.Category == nil {
			continue
		}
		catID := tx.Category.Hex()
		spendingByCatAll[catID] += tx.Amount
		switch catToBucket[catID] {
		case "NEEDS":
			totalNeeds += tx.Amount
		case "WANTS":
			totalWants += tx.Amount
		}
		if owner, ok := usernameByOwnerID[walletOwnerID[tx.Wallet]]; ok {
			if spendingByCatOwner[catID] == nil {
				spendingByCatOwner[catID] = map[string]float64{}
			}
			spendingByCatOwner[catID][owner] += tx.Amount
		}
	}

	// Realized income
	incCursor, err := db.Col("transactions").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"wallet": bson.M{"$in": walletIDs},
			"date":   bson.M{"$gte": startDate, "$lt": endDate},
			"type":   "INCOME", "isDeleted": false,
		}}},
		{{Key: "$group", Value: bson.D{{Key: "_id", Value: nil}, {Key: "totalIncome", Value: bson.D{{Key: "$sum", Value: "$amount"}}}}}},
	})
	if err != nil {
		return nil, err
	}
	var incResult []struct {
		TotalIncome float64 `bson:"totalIncome"`
	}
	if err := incCursor.All(ctx, &incResult); err != nil {
		return nil, err
	}
	realizedIncome := 0.0
	if len(incResult) > 0 {
		realizedIncome = incResult[0].TotalIncome
	}

	// Days remaining in the month (shared by every envelope regardless of
	// Type -- budgeting stays monthly).
	now := time.Now()
	daysRemaining := 0
	if period == now.In(wib).Format("2006-01") {
		d := int(endDate.Sub(now).Hours() / 24)
		if d < 1 {
			d = 1
		}
		daysRemaining = d
	}

	// Build envelopes
	totalBudget := 0.0
	coveredSpent := 0.0
	coveredKeysSeen := map[string]bool{}
	envelopes := make([]EnvelopeOverview, 0, len(budget.Envelopes))

	for _, env := range budget.Envelopes {
		if env.Name == "" {
			continue // Skip legacy or invalid envelopes
		}
		envType := env.Type
		if envType == "" {
			envType = TypeMonthly
		}

		spent := 0.0
		categories := make([]CategorySpend, 0, len(env.Items))
		for _, item := range env.Items {
			owner := strings.ToUpper(item.Owner)
			var catSpent float64
			if owner == OwnerAll {
				catSpent = spendingByCatAll[item.CategoryID]
			} else {
				catSpent = spendingByCatOwner[item.CategoryID][owner]
			}
			spent += catSpent
			info := catByID[item.CategoryID]
			categories = append(categories, CategorySpend{
				ID: item.CategoryID, Name: info.Name, Icon: info.Icon, Color: info.Color,
				Owner: item.Owner, Spent: catSpent,
			})

			// A (categoryId, owner) pair only counts once toward
			// "covered" spend, no matter how many envelopes/Types
			// reference it -- otherwise the same category tracked in two
			// envelopes (e.g. a Daily-paced one AND a Monthly rollup)
			// would double-count here.
			coverKey := item.CategoryID + ":" + owner
			if !coveredKeysSeen[coverKey] {
				coveredKeysSeen[coverKey] = true
				coveredSpent += catSpent
			}
		}
		sort.Slice(categories, func(i, j int) bool { return categories[i].Spent > categories[j].Spent })

		remaining := math.Max(0, env.Limit-spent)
		overage := math.Max(0, spent-env.Limit)
		pct := 0.0
		if env.Limit > 0 {
			pct = math.Min(100, (spent/env.Limit)*100)
		}
		safe := 0.0
		if daysRemaining > 0 && remaining > 0 {
			safe = remaining / float64(daysRemaining)
		}
		totalBudget += env.Limit
		pacingUnit, pacingAmt := pacingAmount(envType, env.Limit, daysInMonth)
		envelopes = append(envelopes, EnvelopeOverview{
			Name: env.Name, Type: envType, Icon: env.Icon, Color: env.Color,
			Limit: env.Limit, Spent: spent, Remaining: remaining, Overage: overage,
			Percent: pct, SafeToSpendToday: safe,
			PacingUnit: pacingUnit, PacingAmount: pacingAmt,
			Items: env.Items, Categories: categories,
		})
	}

	sort.Slice(envelopes, func(i, j int) bool {
		return envelopes[i].Name < envelopes[j].Name
	})

	// Unbudgeted spending: total household spend minus whatever's covered
	// by envelope items. Clamped at 0 as a defensive floor.
	totalSpent := 0.0
	for _, tx := range expTxs {
		totalSpent += tx.Amount
	}
	unbudgetedSpent := math.Max(0, totalSpent-coveredSpent)

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

// GetBudgetHistorical returns the budget overviews for the last N months.
// It queries GetBudgetOverview for each month sequentially.
func GetBudgetHistorical(months int) ([]*BudgetOverview, error) {
	if months <= 0 {
		months = 6 // Default
	}
	var results []*BudgetOverview
	now := time.Now()

	// Loop from (now - months + 1) to now.
	// E.g. if months=6, it means we want 6 months including current month.
	// So we start from i = months - 1 down to 0
	for i := months - 1; i >= 0; i-- {
		targetDate := now.AddDate(0, -i, 0)
		period := targetDate.Format("2006-01")
		
		overview, err := GetBudgetOverview(period)
		if err != nil {
			// If no documents found, just append an empty structured overview
			// or handle it gracefully. GetBudgetOverview actually returns default 
			// 0s for missing budgets, but let's be safe.
			overview = &BudgetOverview{
				Period: period,
			}
		}
		results = append(results, overview)
	}

	return results, nil
}
