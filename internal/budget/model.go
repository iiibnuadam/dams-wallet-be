package budget

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OwnerAll is the sentinel EnvelopeItem.Owner value meaning "combined
// across everyone in the household" -- the only behavior that existed
// before per-owner items were introduced.
const OwnerAll = ""

// EnvelopeItem is one category tracked within an envelope, optionally
// scoped to a single household member's spend on that category (e.g. the
// same "Makan" category can be tracked separately for Adam and Sasti).
type EnvelopeItem struct {
	CategoryID string `bson:"categoryId" json:"categoryId"`
	Owner      string `bson:"owner,omitempty" json:"owner,omitempty"` // "" = combined; "ADAM"/"SASTI" = that person's wallets only
}

// Envelope frequency types. Empty/unrecognized Type is treated as
// TypeMonthly -- backward compatible with envelopes created before this
// field existed.
const (
	TypeDaily   = "DAILY"
	TypeWeekly  = "WEEKLY"
	TypeMonthly = "MONTHLY"
	TypeAnnual  = "ANNUAL"
)

type Envelope struct {
	Name  string         `bson:"name" json:"name"`
	Type  string         `bson:"type,omitempty" json:"type,omitempty"` // DAILY | WEEKLY | MONTHLY | ANNUAL
	Items []EnvelopeItem `bson:"items" json:"items"`
	Icon  string         `bson:"icon" json:"icon"`
	Color string         `bson:"color" json:"color"`
	Limit float64        `bson:"limit" json:"limit"`
}

// MonthlyBudget is a single household-wide budget per period -- shared by
// everyone in the household, not scoped to whichever account is logged in.
type MonthlyBudget struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Period    string             `bson:"period" json:"period"`
	Income    float64            `bson:"income" json:"income"`
	Envelopes []Envelope         `bson:"envelopes" json:"envelopes"`
	IsDeleted bool               `bson:"isDeleted" json:"isDeleted"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type CategorySpend struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Icon  string  `json:"icon"`
	Color string  `json:"color"`
	Owner string  `json:"owner,omitempty"`
	Spent float64 `json:"spent"`
}

type EnvelopeOverview struct {
	Name             string         `json:"name"`
	Type             string         `json:"type"`
	Items            []EnvelopeItem `json:"items"`
	Icon             string         `json:"icon"`
	Color            string         `json:"color"`
	Limit            float64        `json:"limit"`
	Spent            float64        `json:"spent"`
	Remaining        float64        `json:"remaining"`
	Overage          float64        `json:"overage"`
	Percent          float64        `json:"percent"`
	SafeToSpendToday float64        `json:"safeToSpendToday"`
	// PacingUnit/PacingAmount are a purely informational reference (e.g.
	// "≈ Rp30.000/day") showing what the (always month-scoped) Limit works
	// out to in the envelope's own Type unit. They never change
	// Spent/Limit/Overage/Percent -- budgeting stays monthly regardless of
	// Type, this is just a display aid. PacingUnit is "" for Monthly.
	PacingUnit   string          `json:"pacingUnit,omitempty"`
	PacingAmount float64         `json:"pacingAmount,omitempty"`
	Categories   []CategorySpend `json:"categories"`
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
