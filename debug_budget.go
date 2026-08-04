package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/config"
	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	config.Load()
	
	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	defer db.Disconnect(ctx)

	// 1. Get all wallets
	wCursor, err := db.Col("wallets").Find(ctx, bson.M{"isDeleted": false})
	if err != nil {
		log.Fatal(err)
	}
	var wallets []struct {
		ID   interface{} `bson:"_id"`
		Name string      `bson:"name"`
		Type string      `bson:"type"`
	}
	if err = wCursor.All(ctx, &wallets); err != nil {
		log.Fatal(err)
	}
	var walletIDs []interface{}
	for _, w := range wallets {
		walletIDs = append(walletIDs, w.ID)
	}

	// 2. Dashboard monthlyTrend for 2026-07 (UTC)
	sixMonthsAgo, _ := time.Parse(time.RFC3339, "2026-02-01T00:00:00Z")
	endDate, _ := time.Parse(time.RFC3339, "2026-08-31T23:59:59Z")
	
	monthlyAgg, err := db.Col("transactions").Aggregate(ctx, mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"wallet":    bson.M{"$in": walletIDs},
			"date":      bson.M{"$gte": sixMonthsAgo, "$lte": endDate},
			"isDeleted": false,
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{{Key: "$dateToString", Value: bson.D{{Key: "format", Value: "%Y-%m"}, {Key: "date", Value: "$date"}}}}},
			{Key: "expense", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$cond", Value: bson.A{
				bson.D{{Key: "$eq", Value: bson.A{"$type", "EXPENSE"}}}, "$amount", 0,
			}}}}}},
		}}},
	})
	if err != nil {
		log.Fatal(err)
	}
	var mRes []bson.M
	if err = monthlyAgg.All(ctx, &mRes); err != nil {
		log.Fatal(err)
	}
	for _, r := range mRes {
		if r["_id"] == "2026-07" {
			fmt.Printf("Dashboard monthlyTrend 2026-07 Expense: %v\n", r["expense"])
		}
	}

	// 3. Budget expTxs for 2026-07 (WIB)
	loc, _ := time.LoadLocation("Asia/Jakarta")
	if loc == nil {
		loc = time.FixedZone("WIB", 7*3600)
	}
	t, _ := time.Parse("2006-01", "2026-07")
	bStart := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, loc)
	bEnd := bStart.AddDate(0, 1, 0)
	
	expCursor, err := db.Col("transactions").Find(ctx, bson.M{
		"wallet":    bson.M{"$in": walletIDs},
		"date":      bson.M{"$gte": bStart, "$lt": bEnd},
		"type":      "EXPENSE",
		"isDeleted": false,
	})
	if err != nil {
		log.Fatal(err)
	}
	var bTxs []struct {
		ID       interface{} `bson:"_id"`
		Amount   float64     `bson:"amount"`
		Category interface{} `bson:"category"`
		Date     time.Time   `bson:"date"`
		Title    string      `bson:"title"`
		Status   string      `bson:"status"`
	}
	if err = expCursor.All(ctx, &bTxs); err != nil {
		log.Fatal(err)
	}

	totalB := 0.0
	for _, tx := range bTxs {
		totalB += tx.Amount
	}
	fmt.Printf("Budget Total Spent 2026-07: %v\n", totalB)
	
	fmt.Println("Pending/Other transactions in Budget:")
	for _, tx := range bTxs {
		if tx.Status == "PENDING" {
			fmt.Printf(" - PENDING %v: %f\n", tx.Title, tx.Amount)
		}
	}
}
