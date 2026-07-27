package wallets

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LiabilityDetails struct {
	StartDate   *time.Time `bson:"startDate,omitempty" json:"startDate,omitempty"`
	TenorMonths *int       `bson:"tenorMonths,omitempty" json:"tenorMonths,omitempty"`
}

type BankDetails struct {
	BankName      string `bson:"bankName,omitempty" json:"bankName,omitempty"`
	AccountNumber string `bson:"accountNumber,omitempty" json:"accountNumber,omitempty"`
	AccountHolder string `bson:"accountHolder,omitempty" json:"accountHolder,omitempty"`
}

type Wallet struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name            string             `bson:"name" json:"name"`
	Type            string             `bson:"type" json:"type"`
	Owner           primitive.ObjectID `bson:"owner" json:"owner"`
	InitialBalance  float64            `bson:"initialBalance" json:"initialBalance"`
	Color           string             `bson:"color" json:"color"`
	LiabilityDetails *LiabilityDetails `bson:"liabilityDetails,omitempty" json:"liabilityDetails,omitempty"`
	BankDetails     *BankDetails       `bson:"bankDetails,omitempty" json:"bankDetails,omitempty"`
	IsDeleted       bool               `bson:"isDeleted" json:"isDeleted"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// WalletWithBalance is the enriched response type returned from aggregation
type WalletWithBalance struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name            string             `bson:"name" json:"name"`
	Type            string             `bson:"type" json:"type"`
	Owner           string             `bson:"owner" json:"owner"`
	OwnerName       string             `bson:"ownerName" json:"ownerName"`
	InitialBalance  float64            `bson:"initialBalance" json:"initialBalance"`
	CurrentBalance  float64            `bson:"currentBalance" json:"currentBalance"`
	Color           string             `bson:"color" json:"color"`
	LiabilityDetails *LiabilityDetails `bson:"liabilityDetails,omitempty" json:"liabilityDetails,omitempty"`
	BankDetails     *BankDetails       `bson:"bankDetails,omitempty" json:"bankDetails,omitempty"`
	IsDeleted       bool               `bson:"isDeleted" json:"isDeleted"`
	CreatedAt       time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt       time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type CreateWalletRequest struct {
	Name            string            `json:"name"`
	Type            string            `json:"type"`
	InitialBalance  float64           `json:"initialBalance"`
	Color           string            `json:"color"`
	LiabilityDetails *LiabilityDetails `json:"liabilityDetails,omitempty"`
	BankDetails     *BankDetails      `json:"bankDetails,omitempty"`
}
