package routines

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Routine struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"_id"`
	Description  string              `bson:"description" json:"description"`
	Amount       float64             `bson:"amount" json:"amount"`
	Type         string              `bson:"type" json:"type"`
	Wallet       primitive.ObjectID  `bson:"wallet" json:"wallet"`
	TargetWallet *primitive.ObjectID `bson:"targetWallet,omitempty" json:"targetWallet,omitempty"`
	Category     *primitive.ObjectID `bson:"category,omitempty" json:"category,omitempty"`
	Frequency    string              `bson:"frequency" json:"frequency"`
	StartDate    time.Time           `bson:"startDate" json:"startDate"`
	NextRun      time.Time           `bson:"nextRun" json:"nextRun"`
	LastRun      *time.Time          `bson:"lastRun,omitempty" json:"lastRun,omitempty"`
	Status       string              `bson:"status" json:"status"`
	Owner        primitive.ObjectID  `bson:"owner" json:"owner"`
	CreatedAt    time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time           `bson:"updatedAt" json:"updatedAt"`
}

type CreateRoutineRequest struct {
	Description  string  `json:"description"`
	Amount       float64 `json:"amount"`
	Type         string  `json:"type"`
	Wallet       string  `json:"wallet"`
	TargetWallet string  `json:"targetWallet,omitempty"`
	Category     string  `json:"category,omitempty"`
	Frequency    string  `json:"frequency"`
	StartDate    string  `json:"startDate"`
}
