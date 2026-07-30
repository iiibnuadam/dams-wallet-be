package insights

import (
	"context"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/pkg/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func analysisCol() *mongo.Collection {
	return db.Col("insight_analyses")
}

// getSavedAnalysis returns the last saved AI analysis for this
// (user, period, owner) combination, or nil if none exists yet.
func getSavedAnalysis(userID primitive.ObjectID, period, owner string) (*SavedAnalysis, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var saved SavedAnalysis
	err := analysisCol().FindOne(ctx, bson.M{
		"userId": userID, "period": period, "owner": owner,
	}).Decode(&saved)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

// upsertSavedAnalysis overwrites any existing analysis for this
// (user, period, owner) combination -- there is no history kept.
func upsertSavedAnalysis(analysis SavedAnalysis) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := analysisCol().UpdateOne(ctx,
		bson.M{"userId": analysis.UserID, "period": analysis.Period, "owner": analysis.Owner},
		bson.M{"$set": bson.M{
			"narratives":    analysis.Narratives,
			"talkingPoints": analysis.TalkingPoints,
			"source":        analysis.Source,
			"analyzedAt":    analysis.AnalyzedAt,
		}},
		options.Update().SetUpsert(true),
	)
	return err
}
