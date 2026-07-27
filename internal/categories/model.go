package categories

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Category struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	Name        string             `bson:"name" json:"name"`
	Type        string             `bson:"type" json:"type"` // INCOME | EXPENSE
	Flexibility string             `bson:"flexibility" json:"flexibility"`
	Icon        string             `bson:"icon,omitempty" json:"icon,omitempty"`
	Color       string             `bson:"color,omitempty" json:"color,omitempty"`
	Group       string             `bson:"group,omitempty" json:"group,omitempty"`
	Bucket      string             `bson:"bucket,omitempty" json:"bucket,omitempty"` // NEEDS | WANTS | SAVINGS
	IsDeleted   bool               `bson:"isDeleted" json:"isDeleted"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Flexibility string `json:"flexibility"`
	Icon        string `json:"icon,omitempty"`
	Color       string `json:"color,omitempty"`
	Group       string `json:"group,omitempty"`
	Bucket      string `json:"bucket,omitempty"`
}
