package dashboard

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetFinancialHealth(userID primitive.ObjectID, startDateStr, endDateStr, ownerParam string) (*FinancialHealthResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Determine Date Range
	var start, end time.Time
	if startDateStr != "" && endDateStr != "" {
		start, _ = time.Parse(time.RFC3339, startDateStr)
		end, _ = time.Parse(time.RFC3339, endDateStr)
	} else {
		now := time.Now()
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end = start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	}

	// 2. Identify Wallets (Owner Filter)
	filter := bson.M{"isDeleted": false}
	hasOwnerFilter := false

	if ownerParam == "" {
		hasOwnerFilter = true
		filter["owner"] = userID
	} else if strings.ToUpper(ownerParam) != "ALL" {
		oid, err := resolveUsername(ownerParam)
		if err == nil {
			hasOwnerFilter = true
			filter["owner"] = oid
		} else {
			hasOwnerFilter = true
			filter["owner"] = userID
		}
	}

	// Fetch wallets for the target owner (or all)
	wCursor, err := db.Col("wallets").Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	var wallets []bson.M
	if err = wCursor.All(ctx, &wallets); err != nil {
		return nil, err
	}

	walletIDs := []primitive.ObjectID{}
	myWalletIDsMap := make(map[string]bool)
	var currentNetWorth float64

	for _, w := range wallets {
		id := w["_id"].(primitive.ObjectID)
		walletIDs = append(walletIDs, id)
		myWalletIDsMap[id.Hex()] = true

		if bal, ok := w["currentBalance"].(float64); ok {
			currentNetWorth += bal
		} else if bal, ok := w["currentBalance"].(int32); ok {
			currentNetWorth += float64(bal)
		} else if bal, ok := w["currentBalance"].(int64); ok {
			currentNetWorth += float64(bal)
		} else if init, ok := w["initialBalance"].(float64); ok {
			currentNetWorth += init
		} else if init, ok := w["initialBalance"].(int32); ok {
			currentNetWorth += float64(init)
		} else if init, ok := w["initialBalance"].(int64); ok {
			currentNetWorth += float64(init)
		}
	}

	// 3. Granularity
	diffDays := int(end.Sub(start).Hours() / 24)
	granularity := "month"
	if diffDays <= 31 {
		granularity = "day"
	} else if diffDays > 730 {
		granularity = "year"
	}

	// Adjust Base Net Worth if 'end' is in the past
	now := time.Now()
	if end.Before(now) {
		futureFilter := bson.M{
			"date": bson.M{"$gt": end, "$lte": now},
		}
		if hasOwnerFilter && len(walletIDs) > 0 {
			futureFilter["$or"] = []bson.M{
				{"wallet": bson.M{"$in": walletIDs}},
				{"targetWallet": bson.M{"$in": walletIDs}},
			}
		}

		futureCursor, _ := db.Col("transactions").Find(ctx, futureFilter)
		var futureTxns []bson.M
		futureCursor.All(ctx, &futureTxns)

		for _, txn := range futureTxns {
			tType, _ := txn["type"].(string)
			amtVal := 0.0
			if v, ok := txn["amount"].(float64); ok {
				amtVal = v
			} else if v, ok := txn["amount"].(int32); ok {
				amtVal = float64(v)
			} else if v, ok := txn["amount"].(int64); ok {
				amtVal = float64(v)
			}

			wID := ""
			if w, ok := txn["wallet"].(primitive.ObjectID); ok {
				wID = w.Hex()
			}
			tWID := ""
			if tw, ok := txn["targetWallet"].(primitive.ObjectID); ok {
				tWID = tw.Hex()
			}

			isMySource := !hasOwnerFilter || myWalletIDsMap[wID]
			isMyTarget := tWID != "" && (!hasOwnerFilter || myWalletIDsMap[tWID])

			if tType == "INCOME" && isMySource {
				currentNetWorth -= amtVal
			} else if tType == "EXPENSE" && isMySource {
				currentNetWorth += amtVal
			} else if tType == "TRANSFER" {
				if isMySource && !isMyTarget {
					currentNetWorth += amtVal
				} else if !isMySource && isMyTarget {
					currentNetWorth -= amtVal
				}
			}
		}
	}

	// 4. Fetch Transactions for the Interval
	txFilter := bson.M{
		"date": bson.M{"$gte": start, "$lte": end},
	}
	if hasOwnerFilter && len(walletIDs) > 0 {
		txFilter["$or"] = []bson.M{
			{"wallet": bson.M{"$in": walletIDs}},
			{"targetWallet": bson.M{"$in": walletIDs}},
		}
	}

	txCursor, err := db.Col("transactions").Find(ctx, txFilter, options.Find().SetSort(bson.M{"date": -1}))
	if err != nil {
		return nil, err
	}
	var txns []bson.M
	if err = txCursor.All(ctx, &txns); err != nil {
		return nil, err
	}

	// Also fetch with Category & Wallet population for Sankey/Radar
	// To keep it simple, we will fetch Categories and Wallets separately and map them
	catsCursor, _ := db.Col("categories").Find(ctx, bson.M{})
	var cats []bson.M
	catsCursor.All(ctx, &cats)
	catMap := make(map[string]bson.M)
	for _, c := range cats {
		catMap[c["_id"].(primitive.ObjectID).Hex()] = c
	}

	usersCursor, _ := db.Col("users").Find(ctx, bson.M{})
	var users []bson.M
	usersCursor.All(ctx, &users)
	userMap := make(map[string]string)
	for _, u := range users {
		userMap[u["_id"].(primitive.ObjectID).Hex()] = u["name"].(string)
	}

	walletMap := make(map[string]bson.M)
	for _, w := range wallets { // includes all wallets if "ALL", else just owner's
		walletMap[w["_id"].(primitive.ObjectID).Hex()] = w
	}
	// If owner="ALL", we might need ALL wallets to resolve names correctly
	if hasOwnerFilter {
		allWCursor, _ := db.Col("wallets").Find(ctx, bson.M{})
		var allW []bson.M
		allWCursor.All(ctx, &allW)
		for _, w := range allW {
			walletMap[w["_id"].(primitive.ObjectID).Hex()] = w
		}
	}

	// 5. Intervals & Trend Data
	intervalMap := make(map[string]map[string]float64)
	intervals := []time.Time{}
	
	curr := start
	for curr.Before(end) || curr.Equal(end) {
		intervals = append(intervals, curr)
		if granularity == "day" {
			curr = curr.AddDate(0, 0, 1)
		} else if granularity == "month" {
			curr = curr.AddDate(0, 1, 0)
		} else {
			curr = curr.AddDate(1, 0, 0)
		}
	}
	if len(intervals) == 0 {
		intervals = append(intervals, start)
	}

	formatLabel := func(d time.Time) string {
		if granularity == "day" {
			return d.Format("02 Jan")
		} else if granularity == "month" {
			return d.Format("Jan 2006")
		}
		return d.Format("2006")
	}

	for _, d := range intervals {
		intervalMap[formatLabel(d)] = map[string]float64{"income": 0, "expense": 0}
	}

	// Sankey Data
	sankeyLinks := []SankeyLink{}
	sankeyNodesSet := make(map[string]bool)
	
	// Radar Data
	ownerContributionMap := make(map[string]map[string]float64)
	categoriesListSet := make(map[string]bool)
	
	// Fixed vs Variable
	var fixedTotal, variableTotal float64

	for _, txn := range txns {
		tType, _ := txn["type"].(string)
		amtVal := 0.0
		if v, ok := txn["amount"].(float64); ok {
			amtVal = v
		} else if v, ok := txn["amount"].(int32); ok {
			amtVal = float64(v)
		}

		tDate, _ := txn["date"].(primitive.DateTime)
		tTime := tDate.Time()
		label := formatLabel(tTime)

		wID := ""
		if w, ok := txn["wallet"].(primitive.ObjectID); ok {
			wID = w.Hex()
		}
		tWID := ""
		if tw, ok := txn["targetWallet"].(primitive.ObjectID); ok {
			tWID = tw.Hex()
		}
		
		isMySource := !hasOwnerFilter || myWalletIDsMap[wID]
		isMyTarget := tWID != "" && (!hasOwnerFilter || myWalletIDsMap[tWID])

		// Build Trend
		if entry, exists := intervalMap[label]; exists {
			if tType == "INCOME" && isMySource {
				entry["income"] += amtVal
			} else if tType == "EXPENSE" && isMySource {
				entry["expense"] += amtVal
			} else if tType == "TRANSFER" {
				if isMySource && !isMyTarget {
					entry["expense"] += amtVal
				} else if !isMySource && isMyTarget {
					entry["income"] += amtVal
				}
			}
		}

		// Build Sankey & Radar (only if it falls into our wallets logic)
		wName := "Unknown Wallet"
		if wData, ok := walletMap[wID]; ok {
			wName = wData["name"].(string)
		}
		
		catName := "Other"
		catFlex := "VARIABLE"
		if cID, ok := txn["category"].(primitive.ObjectID); ok {
			if cData, ok := catMap[cID.Hex()]; ok {
				catName = cData["name"].(string)
				if f, ok := cData["flexibility"].(string); ok {
					catFlex = f
				}
			}
		}

		if tType == "INCOME" {
			source := catName + " (In)"
			sankeyLinks = append(sankeyLinks, SankeyLink{Source: source, Target: wName, Value: amtVal})
			sankeyNodesSet[source] = true
			sankeyNodesSet[wName] = true
		} else if tType == "EXPENSE" {
			sankeyLinks = append(sankeyLinks, SankeyLink{Source: wName, Target: catName, Value: amtVal})
			sankeyNodesSet[wName] = true
			sankeyNodesSet[catName] = true
			
			if catFlex == "FIXED" {
				fixedTotal += amtVal
			} else {
				variableTotal += amtVal
			}

			wOwner := "Unknown"
			if wData, ok := walletMap[wID]; ok {
				if oID, ok := wData["owner"].(primitive.ObjectID); ok {
					if name, ok := userMap[oID.Hex()]; ok {
						wOwner = name
					}
				}
			}

			if ownerContributionMap[wOwner] == nil {
				ownerContributionMap[wOwner] = make(map[string]float64)
			}
			ownerContributionMap[wOwner][catName] += amtVal
			categoriesListSet[catName] = true
		}
	}

	// Finalize Trend backwards
	trendData := []TrendDataPoint{}
	runningNW := currentNetWorth

	for i := len(intervals) - 1; i >= 0; i-- {
		d := intervals[i]
		label := formatLabel(d)
		flow := intervalMap[label]
		if flow == nil {
			flow = map[string]float64{"income": 0, "expense": 0}
		}
		change := flow["income"] - flow["expense"]

		trendData = append([]TrendDataPoint{{
			Date:     d.Format(time.RFC3339),
			Label:    label,
			NetWorth: runningNW,
			Income:   flow["income"],
			Expense:  flow["expense"],
			CashFlow: change,
		}}, trendData...) // Prepend
		
		runningNW -= change
	}

	// Finalize Sankey
	aggLinksMap := make(map[string]float64)
	for _, l := range sankeyLinks {
		key := l.Source + "->" + l.Target
		aggLinksMap[key] += l.Value
	}
	finalLinks := []SankeyLink{}
	for k, v := range aggLinksMap {
		parts := strings.Split(k, "->")
		finalLinks = append(finalLinks, SankeyLink{Source: parts[0], Target: parts[1], Value: v})
	}
	finalNodes := []SankeyNode{}
	for k := range sankeyNodesSet {
		finalNodes = append(finalNodes, SankeyNode{Name: k})
	}

	// Finalize Radar
	categoriesList := []string{}
	for k := range categoriesListSet {
		categoriesList = append(categoriesList, k)
	}
	if len(categoriesList) > 6 {
		categoriesList = categoriesList[:6]
	}
	radarData := []RadarDataPoint{}
	for _, cat := range categoriesList {
		pt := RadarDataPoint{"subject": cat}
		for owner, cMap := range ownerContributionMap {
			pt[owner] = cMap[cat]
		}
		radarData = append(radarData, pt)
	}

	// Liabilities
	liabWalletsCursor, _ := db.Col("wallets").Find(ctx, bson.M{
		"liabilityDetails.tenorMonths": bson.M{"$exists": true, "$gt": 0},
		"isDeleted": false,
	})
	var liabWallets []bson.M
	liabWalletsCursor.All(ctx, &liabWallets)
	liabilities := []LiabilityProgress{}

	for _, w := range liabWallets {
		name := w["name"].(string)
		var tenor int
		var startDate time.Time
		if l, ok := w["liabilityDetails"].(bson.M); ok {
			if t, ok := l["tenorMonths"].(int32); ok {
				tenor = int(t)
			}
			if sd, ok := l["startDate"].(primitive.DateTime); ok {
				startDate = sd.Time()
			}
		}
		monthsPassed := 0
		if !startDate.IsZero() {
			diffTime := time.Since(startDate).Hours() / 24
			monthsPassed = int(math.Ceil(diffTime / 30))
		}
		monthsLeft := tenor - monthsPassed
		if monthsLeft < 0 {
			monthsLeft = 0
		}
		progress := 0.0
		if tenor > 0 {
			progress = math.Min(100.0, float64(monthsPassed)/float64(tenor)*100.0)
		}
		liabilities = append(liabilities, LiabilityProgress{
			Name: name, TotalMonths: tenor, MonthsPassed: monthsPassed, MonthsLeft: monthsLeft, Progress: progress,
		})
	}

	// MacroStats
	var totalIncome, totalExpense float64
	for _, t := range trendData {
		totalIncome += t.Income
		totalExpense += t.Expense
	}
	savingsRate := 0.0
	if totalIncome > 0 {
		savingsRate = ((totalIncome - totalExpense) / totalIncome) * 100
	}
	fixedRatio := 0.0
	if (fixedTotal + variableTotal) > 0 {
		fixedRatio = (fixedTotal / (fixedTotal + variableTotal)) * 100
	}

	return &FinancialHealthResponse{
		Trend:       trendData,
		Granularity: granularity,
		SankeyData: SankeyData{
			Nodes: finalNodes,
			Links: finalLinks,
		},
		RadarData:   radarData,
		Liabilities: liabilities,
		FixedVsVariable: FixedVsVariable{
			Fixed:    fixedTotal,
			Variable: variableTotal,
			Ratio:    fixedRatio,
		},
		MacroStats: MacroStats{
			CurrentNetWorth: currentNetWorth, // Wait, it should be the net worth at the END of the period.
			TotalIncome:     totalIncome,
			TotalExpense:    totalExpense,
			SavingsRate:     savingsRate,
		},
	}, nil
}
