package insights

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/internal/budget"
	"github.com/ibnuadam/dams-wallet-backend/internal/dashboard"
	"github.com/ibnuadam/dams-wallet-backend/internal/debts"
	"github.com/ibnuadam/dams-wallet-backend/internal/goals"
	"github.com/ibnuadam/dams-wallet-backend/pkg/llm"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// llmClient is a startup-singleton, same convention as db.Connect(). A
// disabled client (no API key) makes every call a safe no-op.
var llmClient = llm.New(llm.Config{})

func SetLLMClient(c *llm.Client) { llmClient = c }

func previousPeriod(period string) string {
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return period
	}
	return t.AddDate(0, -1, 0).Format("2006-01")
}

// buildSignals gathers rule-computed signals across budget/goals/debts/
// financial-health for the given period. Every signal's Narrative starts
// equal to its Message -- a valid rule-authored fallback on its own.
func buildSignals(userID primitive.ObjectID, period, owner string) ([]Signal, error) {
	budgetOverview, err := budget.GetBudgetOverview(period)
	if err != nil {
		return nil, err
	}
	prevBudgetOverview, _ := budget.GetBudgetOverview(previousPeriod(period))

	goalList, err := goals.GetGoals(owner)
	if err != nil {
		return nil, err
	}
	debtList, err := debts.GetDebts(owner)
	if err != nil {
		return nil, err
	}

	t, _ := time.Parse("2006-01", period)
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	health, err := dashboard.GetFinancialHealth(userID, start.Format(time.RFC3339), end.Format(time.RFC3339), owner)
	if err != nil {
		return nil, err
	}

	// dashboard.GetSummary is the same source as the "Real Expense"/"Total
	// Expense" cards shown elsewhere in the app: it excludes PENDING
	// transactions and never counts transfers as spend. GetFinancialHealth's
	// MacroStats differs on both counts (by design, for its Sankey/cash-flow
	// chart) -- using it for income/expense here would show a "pengeluaran"
	// figure that silently disagrees with the rest of the UI.
	summary, err := dashboard.GetSummary(userID, period, "", "", owner, "")
	if err != nil {
		return nil, err
	}

	var signals []Signal
	signals = append(signals, BudgetSignals(budgetOverview)...)
	signals = append(signals, SpendingDeltaSignals(budgetOverview, prevBudgetOverview)...)
	signals = append(signals, GoalSignals(goalList)...)
	signals = append(signals, DebtSignals(debtList)...)
	signals = append(signals, LiabilitySignals(health.Liabilities)...)
	signals = append(signals, NetWorthSignals(summary.TotalIncome, summary.TotalExpense)...)
	signals = append(signals, MonthlySummarySignal(summary.TotalIncome, summary.TotalExpense)...)
	signals = append(signals, FixedVsVariableSignal(health.FixedVsVariable)...)
	signals = append(signals, BorrowingPowerSignal(summary.TotalIncome, summary.TotalFixedIncome, health.FixedVsVariable)...)
	signals = append(signals, TopSpendingCategorySignal(budgetOverview)...)
	signals = append(signals, WantsRatioSignal(budgetOverview)...)

	for i := range signals {
		signals[i].Narrative = signals[i].Message
	}
	return signals, nil
}

// GetInsights returns rule-computed signals -- free and instant, no LLM
// call. If a saved AI analysis exists for this (user, period, owner), its
// narratives/talking points are layered on top; otherwise the response is
// plain rule text with source "rules_only". Call AnalyzeInsights to
// (re)generate the AI analysis.
func GetInsights(userID primitive.ObjectID, period, owner string) (*InsightsResponse, error) {
	if period == "" {
		period = time.Now().Format("2006-01")
	}

	signals, err := buildSignals(userID, period, owner)
	if err != nil {
		return nil, err
	}

	resp := &InsightsResponse{
		Period:        period,
		GeneratedAt:   time.Now(),
		Signals:       signals,
		TalkingPoints: []TalkingPoint{},
		Source:        "rules_only",
	}

	saved, err := getSavedAnalysis(userID, period, owner)
	if err != nil {
		log.Printf("insights: failed to load saved analysis, continuing rules-only: %v", err)
		return resp, nil
	}
	if saved != nil {
		for i := range resp.Signals {
			if n, ok := saved.Narratives[resp.Signals[i].ID]; ok && n != "" {
				resp.Signals[i].Narrative = n
			}
		}
		resp.TalkingPoints = saved.TalkingPoints
		resp.Source = saved.Source
		resp.Provider = saved.Provider
		analyzedAt := saved.AnalyzedAt
		resp.AnalyzedAt = &analyzedAt
	}
	return resp, nil
}

// AnalyzeInsights recomputes signals fresh, asks the LLM to narrate them,
// and overwrites the saved analysis for this (user, period, owner) --
// there is no history, only the most recent AI analysis. If the LLM call
// fails (including when no API key is configured), it returns an error;
// callers should fall back to displaying GetInsights' rules-only result.
func AnalyzeInsights(userID primitive.ObjectID, period, owner string) (*InsightsResponse, error) {
	if period == "" {
		period = time.Now().Format("2006-01")
	}
	if !llmClient.Enabled() {
		return nil, fmt.Errorf("insights: AI analysis is not configured (provider %q unavailable)", llmClient.Provider())
	}

	signals, err := buildSignals(userID, period, owner)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), llmClient.Timeout())
	defer cancel()
	narrativeMap, talkingPoints, err := NarrateBatch(ctx, llmClient, signals)
	if err != nil {
		return nil, err
	}

	for i := range signals {
		if n, ok := narrativeMap[signals[i].ID]; ok && n != "" {
			signals[i].Narrative = n
		}
	}
	now := time.Now()
	provider := llmClient.Provider()

	if err := upsertSavedAnalysis(SavedAnalysis{
		UserID: userID, Period: period, Owner: owner,
		Narratives: narrativeMap, TalkingPoints: talkingPoints,
		Source: "llm", Provider: provider, AnalyzedAt: now,
	}); err != nil {
		log.Printf("insights: failed to save analysis (returning it anyway): %v", err)
	}

	return &InsightsResponse{
		Period:        period,
		GeneratedAt:   now,
		Signals:       signals,
		TalkingPoints: talkingPoints,
		Source:        "llm",
		Provider:      provider,
		AnalyzedAt:    &now,
	}, nil
}
