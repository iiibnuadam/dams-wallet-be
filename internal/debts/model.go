package debts

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Debt struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"_id"`
	Type        string              `bson:"type" json:"type"` // LENT | BORROWED
	Person      string              `bson:"person" json:"person"`
	Amount      float64             `bson:"amount" json:"amount"`
	Description string              `bson:"description" json:"description"`
	LoanDate    time.Time           `bson:"loanDate" json:"loanDate"`
	DueDate     *time.Time          `bson:"dueDate,omitempty" json:"dueDate,omitempty"`
	Status      string              `bson:"status" json:"status"` // ACTIVE | PAID
	ProofURL    string              `bson:"proofUrl,omitempty" json:"proofUrl,omitempty"`
	Notes       string              `bson:"notes,omitempty" json:"notes,omitempty"`
	Owner       primitive.ObjectID  `bson:"owner" json:"owner"`
	CreatedAt   time.Time           `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time           `bson:"updatedAt" json:"updatedAt"`
}

type DebtStats struct {
	Lent     float64 `json:"lent"`
	Borrowed float64 `json:"borrowed"`
}

type CreateDebtRequest struct {
	Type        string  `json:"type"`
	Person      string  `json:"person"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description"`
	LoanDate    string  `json:"loanDate"`
	DueDate     string  `json:"dueDate,omitempty"`
	Notes       string  `json:"notes,omitempty"`
}

type SettleRequest struct {
	WalletID string `json:"walletId"`
}
