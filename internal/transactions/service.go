package transactions

import (
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ListTransactions(f ListFilters) (*ListResult, error) {
	return listTransactions(f)
}

func CreateTransaction(req CreateTransactionRequest, createdByID primitive.ObjectID) (primitive.ObjectID, error) {
	walletID, err := primitive.ObjectIDFromHex(req.Wallet)
	if err != nil {
		return primitive.NilObjectID, err
	}

	date, err := time.Parse(time.RFC3339, req.Date)
	if err != nil {
		date = time.Now()
	}

	doc := bson.M{
		"date":        date,
		"amount":      req.Amount,
		"description": req.Description,
		"type":        req.Type,
		"wallet":      walletID,
		"createdBy":   createdByID,
		"isDeleted":   false,
		"status":      "COMPLETED",
		"isTransfer":  false,
	}

	if req.TargetWallet != "" {
		if twID, err := primitive.ObjectIDFromHex(req.TargetWallet); err == nil {
			doc["targetWallet"] = twID
			doc["isTransfer"] = true
		}
	}
	if req.Category != "" {
		if cID, err := primitive.ObjectIDFromHex(req.Category); err == nil {
			doc["category"] = cID
		}
	}
	if req.GoalItem != "" {
		if giID, err := primitive.ObjectIDFromHex(req.GoalItem); err == nil {
			doc["goalItem"] = giID
		}
	}
	if req.PaymentPhase != "" {
		doc["paymentPhase"] = req.PaymentPhase
	}
	if req.AdminFee > 0 {
		doc["adminFee"] = req.AdminFee
	}
	if req.PartnerOwed > 0 {
		doc["partnerOwed"] = req.PartnerOwed
	}

	return createTransaction(doc)
}

func CreateBatchTransactions(reqs []CreateTransactionRequest, createdByID primitive.ObjectID) (int, error) {
	var docs []interface{}
	for _, req := range reqs {
		walletID, err := primitive.ObjectIDFromHex(req.Wallet)
		if err != nil {
			return 0, fmt.Errorf("invalid wallet id for transaction '%s': %v", req.Description, err)
		}

		date, err := time.Parse(time.RFC3339, req.Date)
		if err != nil {
			date = time.Now()
		}

		doc := bson.M{
			"date":        date,
			"amount":      req.Amount,
			"description": req.Description,
			"type":        req.Type,
			"wallet":      walletID,
			"createdBy":   createdByID,
			"isDeleted":   false,
			"status":      "COMPLETED",
			"isTransfer":  false,
			"createdAt":   time.Now(),
			"updatedAt":   time.Now(),
		}

		if req.TargetWallet != "" {
			if twID, err := primitive.ObjectIDFromHex(req.TargetWallet); err == nil {
				doc["targetWallet"] = twID
				doc["isTransfer"] = true
			}
		}
		if req.Category != "" {
			if cID, err := primitive.ObjectIDFromHex(req.Category); err == nil {
				doc["category"] = cID
			}
		}
		if req.GoalItem != "" {
			if giID, err := primitive.ObjectIDFromHex(req.GoalItem); err == nil {
				doc["goalItem"] = giID
			}
		}
		if req.PaymentPhase != "" {
			doc["paymentPhase"] = req.PaymentPhase
		}
		if req.AdminFee > 0 {
			doc["adminFee"] = req.AdminFee
		}
		if req.PartnerOwed > 0 {
			doc["partnerOwed"] = req.PartnerOwed
		}

		docs = append(docs, doc)
	}

	return createBatchTransactions(docs)
}

func SoftDeleteTransaction(id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	return updateTransaction(oid, bson.M{"isDeleted": true})
}

func ConfirmTransaction(id string) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	return updateTransaction(oid, bson.M{"status": "COMPLETED"})
}

func UpdateTransaction(id string, update bson.M) error {
	oid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}
	return updateTransaction(oid, update)
}

func ParseFilters(params map[string]string) ListFilters {
	owner := params["owner"]
	if owner == "" {
		owner = params["view"]
	}
	f := ListFilters{
		Month:       params["month"],
		Mode:        params["mode"],
		StartDate:   params["startDate"],
		EndDate:     params["endDate"],
		Type:        params["type"],
		WalletID:    params["walletId"],
		View:        owner,
		Q:           params["q"],
		GoalID:      params["goalId"],
		SummaryOnly: params["summaryOnly"] == "true",
	}

	if catID := params["categoryId"]; catID != "" && catID != "ALL" {
		f.CategoryIDs = strings.Split(catID, ",")
	}
	if p := params["page"]; p != "" {
		fmt.Sscanf(p, "%d", &f.Page)
	}
	if l := params["limit"]; l != "" {
		fmt.Sscanf(l, "%d", &f.Limit)
	}
	if min := params["minAmount"]; min != "" {
		fmt.Sscanf(min, "%f", &f.MinAmount)
	}
	if max := params["maxAmount"]; max != "" {
		fmt.Sscanf(max, "%f", &f.MaxAmount)
	}
	return f
}
