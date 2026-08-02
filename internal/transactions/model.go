package transactions

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Transaction struct {
	ID                   primitive.ObjectID  `bson:"_id,omitempty" json:"_id"`
	Date                 time.Time           `bson:"date" json:"date"`
	Amount               float64             `bson:"amount" json:"amount"`
	Description          string              `bson:"description,omitempty" json:"description,omitempty"`
	Type                 string              `bson:"type" json:"type"` // INCOME | EXPENSE | TRANSFER
	Wallet               primitive.ObjectID  `bson:"wallet" json:"wallet"`
	TargetWallet         *primitive.ObjectID `bson:"targetWallet,omitempty" json:"targetWallet,omitempty"`
	Category             *primitive.ObjectID `bson:"category,omitempty" json:"category,omitempty"`
	GoalItem             *primitive.ObjectID `bson:"goalItem,omitempty" json:"goalItem,omitempty"`
	PaymentPhase         string              `bson:"paymentPhase,omitempty" json:"paymentPhase,omitempty"`
	PartnerOwed          float64             `bson:"partnerOwed,omitempty" json:"partnerOwed,omitempty"`
	AdminFee             float64             `bson:"adminFee,omitempty" json:"adminFee,omitempty"`
	CreatedBy            primitive.ObjectID  `bson:"createdBy" json:"createdBy"`
	IsDeleted            bool                `bson:"isDeleted" json:"isDeleted"`
	Status               string              `bson:"status" json:"status"` // PENDING | COMPLETED
	RoutineID            *primitive.ObjectID `bson:"routineId,omitempty" json:"routineId,omitempty"`
	RelatedTransactionID *primitive.ObjectID `bson:"relatedTransactionId,omitempty" json:"relatedTransactionId,omitempty"`
	IsTransfer           bool                `bson:"isTransfer" json:"isTransfer"`
	CreatedAt            time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt            time.Time           `bson:"updatedAt" json:"updatedAt"`
}

type PopulatedTransaction struct {
	ID                 primitive.ObjectID  `bson:"_id,omitempty" json:"_id"`
	Date               time.Time           `bson:"date" json:"date"`
	Amount             float64             `bson:"amount" json:"amount"`
	Description        string              `bson:"description,omitempty" json:"description,omitempty"`
	Type               string              `bson:"type" json:"type"`
	Wallet             *RefDoc             `bson:"wallet" json:"wallet"`
	TargetWallet       *RefDoc             `bson:"targetWallet,omitempty" json:"targetWallet,omitempty"`
	Category           *RefDoc             `bson:"category,omitempty" json:"category,omitempty"`
	GoalItem           *GoalItemRef        `bson:"goalItem,omitempty" json:"goalItem,omitempty"`
	CreatedBy          string              `bson:"createdBy" json:"createdBy"`
	IsDeleted          bool                `bson:"isDeleted" json:"isDeleted"`
	Status             string              `bson:"status" json:"status"`
	RoutineID          *primitive.ObjectID `bson:"routineId,omitempty" json:"routineId,omitempty"`
	IsTransfer         bool                `bson:"isTransfer" json:"isTransfer"`
	RelatedTransaction *RelatedTxnRef      `bson:"relatedTransaction,omitempty" json:"relatedTransaction,omitempty"`
	CreatedAt          time.Time           `bson:"createdAt" json:"createdAt"`
}

type RelatedTxnRef struct {
	Wallet *RefDoc `bson:"wallet" json:"wallet"`
}

type RefDoc struct {
	ID   primitive.ObjectID `bson:"_id" json:"_id"`
	Name string             `bson:"name" json:"name"`
}

type GoalItemRef struct {
	ID   primitive.ObjectID `bson:"_id" json:"_id"`
	Name string             `bson:"name" json:"name"`
	Goal *RefDoc            `bson:"goalId,omitempty" json:"goal,omitempty"`
}

type CategoryBreakdown struct {
	Name  string  `bson:"categoryName" json:"name"`
	Value float64 `bson:"total" json:"value"`
	Color string  `bson:"categoryColor,omitempty" json:"color,omitempty"`
	Icon  string  `bson:"categoryIcon,omitempty" json:"icon,omitempty"`
}

type Summary struct {
	TotalIncome       float64             `json:"totalIncome"`
	TotalExpense      float64             `json:"totalExpense"`
	Net               float64             `json:"net"`
	IncomeCategories  []CategoryBreakdown `json:"incomeCategories"`
	ExpenseCategories []CategoryBreakdown `json:"expenseCategories"`
}

type ListResult struct {
	Transactions []PopulatedTransaction `json:"transactions"`
	Pagination   Pagination             `json:"pagination"`
	Summary      Summary                `json:"summary"`
}

type Pagination struct {
	CurrentPage int   `json:"currentPage"`
	TotalPages  int   `json:"totalPages"`
	TotalItems  int64 `json:"totalItems"`
	Limit       int   `json:"limit"`
}

type CreateTransactionRequest struct {
	Date         string  `json:"date"`
	Amount       float64 `json:"amount"`
	Description  string  `json:"description"`
	Type         string  `json:"type"`
	Wallet       string  `json:"wallet"`
	TargetWallet string  `json:"targetWallet,omitempty"`
	Category     string  `json:"category,omitempty"`
	GoalItem     string  `json:"goalItem,omitempty"`
	PaymentPhase string  `json:"paymentPhase,omitempty"`
	PartnerOwed  float64 `json:"partnerOwed,omitempty"`
	AdminFee     float64 `json:"adminFee,omitempty"`
}

// UpdateTransactionRequest intentionally excludes GoalItem/PaymentPhase/
// PartnerOwed -- goal-payment transactions are edited from the Goal Detail
// page's dedicated flow, not this generic endpoint (see the goalItem guard
// in UpdateTransaction).
type UpdateTransactionRequest struct {
	Date         string  `json:"date"`
	Amount       float64 `json:"amount"`
	Description  string  `json:"description"`
	Type         string  `json:"type"`
	Wallet       string  `json:"wallet"`
	TargetWallet string  `json:"targetWallet,omitempty"`
	Category     string  `json:"category,omitempty"`
	AdminFee     float64 `json:"adminFee,omitempty"`
}

type ListFilters struct {
	Month       string
	Mode        string
	StartDate   string
	EndDate     string
	Type        string
	WalletID    string
	CategoryIDs []string
	View        string
	Q           string
	MinAmount   float64
	MaxAmount   float64
	GoalID      string
	Page        int
	Limit       int
	SummaryOnly bool
}
