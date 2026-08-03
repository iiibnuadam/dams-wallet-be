package insights

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type SignalCategory string

const (
	CategoryBudget    SignalCategory = "BUDGET"
	CategoryGoal      SignalCategory = "GOAL"
	CategoryDebt      SignalCategory = "DEBT"
	CategorySpending  SignalCategory = "SPENDING"
	CategoryLiability SignalCategory = "LIABILITY"
	CategoryNetWorth  SignalCategory = "NET_WORTH"
)

type Severity string

const (
	SeverityPositive Severity = "positive"
	SeverityWarning  Severity = "warning"
	SeverityNeutral  Severity = "neutral"
)

// Signal is one rule-computed fact. Message/Value/Facts are always
// rule-authored and trustworthy; Narrative starts equal to Message and is
// only ever overwritten by the LLM layer with a rephrasing of the same facts
// -- never new numbers.
type Signal struct {
	ID        string                 `json:"id"`
	Category  SignalCategory         `json:"category"`
	Severity  Severity               `json:"severity"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Narrative string                 `json:"narrative"`
	Value     string                 `json:"value"`
	Facts     map[string]interface{} `json:"facts"`
}

type TalkingPoint struct {
	Question         string   `json:"question"`
	RelatedSignalIDs []string `json:"relatedSignalIds"`
}

type InsightsResponse struct {
	Period        string         `json:"period"`
	GeneratedAt   time.Time      `json:"generatedAt"`
	Signals       []Signal       `json:"signals"`
	TalkingPoints []TalkingPoint `json:"talkingPoints"`
	Source        string         `json:"source"`   // "llm" | "rules_only"
	Provider      string         `json:"provider"` // e.g. "deepseek", "huggingface", "" when rules-only
	AnalyzedAt    *time.Time     `json:"analyzedAt,omitempty"`
}

// SavedAnalysis is the last AI-narrated analysis for a given (user, period,
// owner) combination. Triggering a new analysis overwrites it in place --
// there is no history, only the most recent result.
type SavedAnalysis struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	UserID        primitive.ObjectID `bson:"userId"`
	Period        string             `bson:"period"`
	Owner         string             `bson:"owner"`
	Narratives    map[string]string  `bson:"narratives"`
	TalkingPoints []TalkingPoint     `bson:"talkingPoints"`
	Source        string             `bson:"source"`
	Provider      string             `bson:"provider"`
	AnalyzedAt    time.Time          `bson:"analyzedAt"`
}
