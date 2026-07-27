package budget

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Envelope struct {
	Name        string   `bson:"name" json:"name"`
	CategoryIDs []string `bson:"categoryIds" json:"categoryIds"` // Map to specific category ObjectIDs
	Icon        string   `bson:"icon" json:"icon"`
	Color       string   `bson:"color" json:"color"`
	Limit       float64  `bson:"limit" json:"limit"`
}

type MonthlyBudget struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	User      primitive.ObjectID `bson:"user" json:"user"`
	Period    string             `bson:"period" json:"period"`
	Income    float64            `bson:"income" json:"income"`
	Envelopes []Envelope         `bson:"envelopes" json:"envelopes"`
	IsDeleted bool               `bson:"isDeleted" json:"isDeleted"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type EnvelopeOverview struct {
	Name            string   `json:"name"`
	CategoryIDs     []string `json:"categoryIds"`
	Icon            string   `json:"icon"`
	Color           string   `json:"color"`
	Limit           float64  `json:"limit"`
	Spent           float64  `json:"spent"`
	Remaining       float64  `json:"remaining"`
	Percent         float64  `json:"percent"`
	SafeToSpendToday float64 `json:"safeToSpendToday"`
}

type BudgetOverview struct {
	Period          string             `json:"period"`
	Income          float64            `json:"income"`
	RealizedIncome  float64            `json:"realizedIncome"`
	Envelopes       []EnvelopeOverview `json:"envelopes"`
	UnbudgetedSpent float64            `json:"unbudgetedSpent"`
	TotalBudget     float64            `json:"totalBudget"`
	TotalSpent      float64            `json:"totalSpent"`
	TotalNeeds      float64            `json:"totalNeeds"`
	TotalWants      float64            `json:"totalWants"`
	DaysRemaining   int                `json:"daysRemaining"`
}

type AvailableGroup struct {
	GroupName string `json:"groupName"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
}

type UpsertEnvelopesRequest struct {
	Period    string     `json:"period"`
	Envelopes []Envelope `json:"envelopes"`
	Income    float64    `json:"income"`
}
