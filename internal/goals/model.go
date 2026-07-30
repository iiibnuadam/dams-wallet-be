package goals

import (
	"time"

	"github.com/ibnuadam/dams-wallet-backend/internal/transactions"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Group struct {
	ID            primitive.ObjectID  `bson:"_id,omitempty" json:"_id"`
	Name          string              `bson:"name" json:"name"`
	Color         string              `bson:"color,omitempty" json:"color,omitempty"`
	Icon          string              `bson:"icon,omitempty" json:"icon,omitempty"`
	ParentGroupID *primitive.ObjectID `bson:"parentGroupId,omitempty" json:"parentGroupId,omitempty"`
}

type Goal struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name        string             `bson:"name" json:"name"`
	Owner       primitive.ObjectID `bson:"owner" json:"owner"`
	TargetDate  time.Time          `bson:"targetDate" json:"targetDate"`
	Visibility  string             `bson:"visibility" json:"visibility"`
	SharedWith  []string           `bson:"sharedWith" json:"sharedWith"`
	Color       string             `bson:"color,omitempty" json:"color,omitempty"`
	Icon        string             `bson:"icon,omitempty" json:"icon,omitempty"`
	Groups      []Group            `bson:"groups,omitempty" json:"groups,omitempty"`
	IsCompleted bool               `bson:"isCompleted" json:"isCompleted"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type GoalPacing struct {
	ExpectedProgressPercent float64 `json:"expectedProgressPercent"`
	ActualProgressPercent   float64 `json:"actualProgressPercent"`
	IsFallingBehind         bool    `json:"isFallingBehind"`
	MonthsRemaining         int     `json:"monthsRemaining"`
}

type GoalWithStats struct {
	Goal
	TotalEstimated float64    `json:"totalEstimated"`
	TotalActual    float64    `json:"totalActual"`
	Pacing         GoalPacing `json:"pacing"`
}

type GoalItem struct {
	ID              primitive.ObjectID  `bson:"_id,omitempty" json:"_id"`
	GoalID          primitive.ObjectID  `bson:"goalId" json:"goalId"`
	GroupID         *primitive.ObjectID `bson:"groupId,omitempty" json:"groupId,omitempty"`
	Name            string              `bson:"name" json:"name"`
	EstimatedAmount float64             `bson:"estimatedAmount" json:"estimatedAmount"`
	IsCompleted     bool                `bson:"isCompleted" json:"isCompleted"`
	ActualAmount    float64             `bson:"actualAmount,omitempty" json:"actualAmount,omitempty"`
	CreatedAt       time.Time           `bson:"createdAt" json:"createdAt"`
}

type GoalDetail struct {
	Goal
	Items   []GoalItem                 `json:"items"`
	History []transactions.Transaction `json:"history"`
}

type CreateGoalRequest struct {
	Name       string   `json:"name"`
	TargetDate string   `json:"targetDate"`
	Visibility string   `json:"visibility"`
	SharedWith []string `json:"sharedWith"`
	Color      string   `json:"color,omitempty"`
	Icon       string   `json:"icon,omitempty"`
}

type CreateGoalItemRequest struct {
	Name            string  `json:"name"`
	EstimatedAmount float64 `json:"estimatedAmount"`
	GroupID         string  `json:"groupId,omitempty"`
}

type GroupRequest struct {
	Name          string `json:"name"`
	Color         string `json:"color,omitempty"`
	Icon          string `json:"icon,omitempty"`
	ParentGroupID string `json:"parentGroupId,omitempty"`
}
