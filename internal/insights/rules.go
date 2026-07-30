package insights

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ibnuadam/dams-wallet-backend/internal/budget"
	"github.com/ibnuadam/dams-wallet-backend/internal/dashboard"
	"github.com/ibnuadam/dams-wallet-backend/internal/debts"
	"github.com/ibnuadam/dams-wallet-backend/internal/goals"
)

const (
	spendingSpikeThresholdPct  = 30.0
	liabilityNearPayoffPercent = 90.0
	savingsRateLowThreshold    = 0.0
	savingsRateHighThreshold   = 20.0
	fixedRatioHighThreshold    = 60.0
	wantsRatioHighThreshold    = 50.0
	safeDebtRatio              = 0.35 // max share of income safely committed to debt installments
)

func formatRupiah(amount float64) string {
	s := strconv.FormatInt(int64(amount), 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, c)
	}
	prefix := "Rp"
	if neg {
		prefix += "-"
	}
	return prefix + string(out)
}

// BudgetSignals flags envelopes spent beyond their limit this period.
func BudgetSignals(overview *budget.BudgetOverview) []Signal {
	var signals []Signal
	if overview == nil {
		return signals
	}
	for _, env := range overview.Envelopes {
		if env.Overage <= 0 {
			continue
		}
		signals = append(signals, Signal{
			ID:       "budget.envelope.over_limit." + env.Name,
			Category: CategoryBudget,
			Severity: SeverityWarning,
			Title:    "Anggaran " + env.Name + " terlampaui",
			Message: fmt.Sprintf("Pengeluaran untuk %s sudah melebihi batas anggaran sebesar %s bulan ini.",
				env.Name, formatRupiah(env.Overage)),
			Value: formatRupiah(env.Overage),
			Facts: map[string]interface{}{
				"envelope": env.Name, "limit": env.Limit, "spent": env.Spent, "overage": env.Overage,
			},
		})
	}
	return signals
}

// SpendingDeltaSignals compares envelope spend against the previous period
// and flags sizeable increases (e.g. "dining spend up 30%").
func SpendingDeltaSignals(current, previous *budget.BudgetOverview) []Signal {
	var signals []Signal
	if current == nil || previous == nil {
		return signals
	}
	prevByName := make(map[string]float64, len(previous.Envelopes))
	for _, env := range previous.Envelopes {
		prevByName[env.Name] = env.Spent
	}
	for _, env := range current.Envelopes {
		prevSpent, ok := prevByName[env.Name]
		if !ok || prevSpent <= 0 {
			continue
		}
		deltaPct := ((env.Spent - prevSpent) / prevSpent) * 100
		if deltaPct < spendingSpikeThresholdPct {
			continue
		}
		signals = append(signals, Signal{
			ID:       "spending.delta.up." + env.Name,
			Category: CategorySpending,
			Severity: SeverityWarning,
			Title:    "Pengeluaran " + env.Name + " naik",
			Message: fmt.Sprintf("Pengeluaran untuk %s naik %.0f%% dibanding periode sebelumnya (%s -> %s).",
				env.Name, deltaPct, formatRupiah(prevSpent), formatRupiah(env.Spent)),
			Value: fmt.Sprintf("+%.0f%%", deltaPct),
			Facts: map[string]interface{}{
				"envelope": env.Name, "previousSpent": prevSpent, "currentSpent": env.Spent, "deltaPercent": deltaPct,
			},
		})
	}
	return signals
}

// GoalSignals flags goals losing pace against their target date.
func GoalSignals(list []goals.GoalWithStats) []Signal {
	var signals []Signal
	for _, g := range list {
		if !g.Pacing.IsFallingBehind {
			continue
		}
		signals = append(signals, Signal{
			ID:       "goal.behind_pace." + g.ID.Hex(),
			Category: CategoryGoal,
			Severity: SeverityWarning,
			Title:    "Goal " + g.Name + " ketinggalan target",
			Message: fmt.Sprintf("Goal %s baru mencapai %.0f%% padahal seharusnya sudah %.0f%% berdasarkan waktu yang berjalan (%d bulan tersisa).",
				g.Name, g.Pacing.ActualProgressPercent, g.Pacing.ExpectedProgressPercent, g.Pacing.MonthsRemaining),
			Value: fmt.Sprintf("%.0f%%", g.Pacing.ActualProgressPercent),
			Facts: map[string]interface{}{
				"goal": g.Name, "expectedProgressPercent": g.Pacing.ExpectedProgressPercent,
				"actualProgressPercent": g.Pacing.ActualProgressPercent, "monthsRemaining": g.Pacing.MonthsRemaining,
			},
		})
	}
	return signals
}

// DebtSignals flags debts that are due soon (one signal each -- rare and
// individually actionable) and rolls up debts that have sat active for a
// long time with no due date into a single per-type summary signal each
// (LENT / BORROWED) instead of one card per debt -- most informal IOUs
// between friends/family never carry a due date, so emitting one per debt
// drowns out every other insight type for anyone with more than a couple.
func DebtSignals(list []debts.Debt) []Signal {
	var signals []Signal
	type staleGroup struct {
		count   int
		total   float64
		persons []string
	}
	staleByType := map[string]*staleGroup{}

	for _, d := range list {
		isDueSoon, daysUntilDue, isStale := debts.ClassifyUrgency(d)
		switch {
		case isDueSoon:
			signals = append(signals, Signal{
				ID:       "debt.due_soon." + d.ID.Hex(),
				Category: CategoryDebt,
				Severity: SeverityWarning,
				Title:    "Utang ke " + d.Person + " segera jatuh tempo",
				Message: fmt.Sprintf("Utang %s ke %s (%s) jatuh tempo dalam %d hari.",
					strings.ToLower(d.Type), d.Person, formatRupiah(d.Amount), daysUntilDue),
				Value: fmt.Sprintf("%d hari lagi", daysUntilDue),
				Facts: map[string]interface{}{
					"person": d.Person, "type": d.Type, "amount": d.Amount, "daysUntilDue": daysUntilDue,
				},
			})
		case isStale:
			g, ok := staleByType[d.Type]
			if !ok {
				g = &staleGroup{}
				staleByType[d.Type] = g
			}
			g.count++
			g.total += d.Amount
			if len(g.persons) < 3 {
				g.persons = append(g.persons, d.Person)
			}
		}
	}

	for debtType, g := range staleByType {
		noun := "piutang" // LENT
		if debtType == "BORROWED" {
			noun = "utang"
		}
		names := strings.Join(g.persons, ", ")
		if g.count > len(g.persons) {
			names += fmt.Sprintf(", dan %d lainnya", g.count-len(g.persons))
		}
		signals = append(signals, Signal{
			ID:       "debt.stale_summary." + strings.ToLower(debtType),
			Category: CategoryDebt,
			Severity: SeverityNeutral,
			Title:    fmt.Sprintf("%d %s lama belum ada tanggal jatuh tempo", g.count, noun),
			Message: fmt.Sprintf("Ada %d %s (total %s) yang masih aktif sejak lama tanpa tanggal jatuh tempo: %s.",
				g.count, noun, formatRupiah(g.total), names),
			Value: formatRupiah(g.total),
			Facts: map[string]interface{}{
				"type": debtType, "count": g.count, "total": g.total, "examples": g.persons,
			},
		})
	}

	return signals
}

// LiabilitySignals flags loans/liabilities close to being fully paid off --
// this is well-defined here (unlike debts.Debt) because wallets carry
// tenor/progress data.
func LiabilitySignals(items []dashboard.LiabilityProgress) []Signal {
	var signals []Signal
	for _, l := range items {
		if l.Progress < liabilityNearPayoffPercent {
			continue
		}
		signals = append(signals, Signal{
			ID:       "liability.near_payoff." + l.Name,
			Category: CategoryLiability,
			Severity: SeverityPositive,
			Title:    l.Name + " hampir lunas",
			Message: fmt.Sprintf("Cicilan %s sudah %.0f%% selesai, tersisa %d bulan lagi.",
				l.Name, l.Progress, l.MonthsLeft),
			Value: fmt.Sprintf("%.0f%%", l.Progress),
			Facts: map[string]interface{}{
				"liability": l.Name, "progress": l.Progress, "monthsLeft": l.MonthsLeft,
			},
		})
	}
	return signals
}

// NetWorthSignals flags a notably low or high savings rate for the period.
func NetWorthSignals(macro dashboard.MacroStats) []Signal {
	var signals []Signal
	if macro.TotalIncome <= 0 {
		return signals
	}
	switch {
	case macro.SavingsRate < savingsRateLowThreshold:
		signals = append(signals, Signal{
			ID:       "networth.savings_rate.negative",
			Category: CategoryNetWorth,
			Severity: SeverityWarning,
			Title:    "Pengeluaran melebihi pemasukan",
			Message: fmt.Sprintf("Pengeluaran periode ini (%s) melebihi pemasukan (%s), savings rate %.0f%%.",
				formatRupiah(macro.TotalExpense), formatRupiah(macro.TotalIncome), macro.SavingsRate),
			Value: fmt.Sprintf("%.0f%%", macro.SavingsRate),
			Facts: map[string]interface{}{
				"totalIncome": macro.TotalIncome, "totalExpense": macro.TotalExpense, "savingsRate": macro.SavingsRate,
			},
		})
	case macro.SavingsRate >= savingsRateHighThreshold:
		signals = append(signals, Signal{
			ID:       "networth.savings_rate.strong",
			Category: CategoryNetWorth,
			Severity: SeverityPositive,
			Title:    "Savings rate cukup sehat",
			Message: fmt.Sprintf("Savings rate periode ini %.0f%% -- pemasukan %s, pengeluaran %s.",
				macro.SavingsRate, formatRupiah(macro.TotalIncome), formatRupiah(macro.TotalExpense)),
			Value: fmt.Sprintf("%.0f%%", macro.SavingsRate),
			Facts: map[string]interface{}{
				"totalIncome": macro.TotalIncome, "totalExpense": macro.TotalExpense, "savingsRate": macro.SavingsRate,
			},
		})
	}
	return signals
}

// MonthlySummarySignal always surfaces the raw income/expense/net numbers
// for the period, regardless of whether they're extreme -- answers
// "berapa pemasukan/pengeluaran/tabungan bulan ini" directly instead of
// only flagging when something is unusually good or bad.
func MonthlySummarySignal(macro dashboard.MacroStats) []Signal {
	if macro.TotalIncome <= 0 && macro.TotalExpense <= 0 {
		return nil
	}
	net := macro.TotalIncome - macro.TotalExpense
	return []Signal{{
		ID:       "networth.monthly_summary",
		Category: CategoryNetWorth,
		Severity: SeverityNeutral,
		Title:    "Ringkasan Bulan Ini",
		Message: fmt.Sprintf("Pemasukan %s, pengeluaran %s, sisa (tabungan) %s.",
			formatRupiah(macro.TotalIncome), formatRupiah(macro.TotalExpense), formatRupiah(net)),
		Value: formatRupiah(net),
		Facts: map[string]interface{}{
			"totalIncome": macro.TotalIncome, "totalExpense": macro.TotalExpense, "net": net, "savingsRate": macro.SavingsRate,
		},
	}}
}

// FixedVsVariableSignal flags a high fixed-cost ratio -- fixed costs are the
// hardest lever to pull, so a high ratio means less room to maneuver if
// income drops.
func FixedVsVariableSignal(fv dashboard.FixedVsVariable) []Signal {
	if fv.Fixed <= 0 && fv.Variable <= 0 {
		return nil
	}
	severity := SeverityPositive
	title := "Biaya tetap masih sehat"
	message := fmt.Sprintf("Biaya tetap (fixed) %s (%.0f%% dari total pengeluaran) -- masih di batas aman.",
		formatRupiah(fv.Fixed), fv.Ratio)
	if fv.Ratio > fixedRatioHighThreshold {
		severity = SeverityWarning
		title = "Biaya tetap cukup tinggi"
		message = fmt.Sprintf("Biaya tetap (fixed) %s (%.0f%% dari total pengeluaran) -- porsi ini susah dikurangi cepat, jadi kalau mau hemat harus lewat biaya variabel.",
			formatRupiah(fv.Fixed), fv.Ratio)
	}
	return []Signal{{
		ID:       "spending.fixed_ratio",
		Category: CategorySpending,
		Severity: severity,
		Title:    title,
		Message:  message,
		Value:    fmt.Sprintf("%.0f%%", fv.Ratio),
		Facts: map[string]interface{}{
			"fixed": fv.Fixed, "variable": fv.Variable, "ratio": fv.Ratio,
		},
	}}
}

// BorrowingPowerSignal estimates safe additional monthly debt capacity:
// income * 35% (a conventional safe debt-to-income ceiling) minus what's
// already committed to fixed costs.
func BorrowingPowerSignal(macro dashboard.MacroStats, fv dashboard.FixedVsVariable) []Signal {
	if macro.TotalIncome <= 0 {
		return nil
	}
	maxSafeInstallment := macro.TotalIncome * safeDebtRatio
	availableCapacity := maxSafeInstallment - fv.Fixed

	severity := SeverityPositive
	title := "Masih ada ruang buat cicilan baru"
	message := fmt.Sprintf("Batas cicilan bulanan yang aman (maks 35%% dari pemasukan) sekitar %s, dan %s masih tersisa setelah biaya tetap.",
		formatRupiah(maxSafeInstallment), formatRupiah(availableCapacity))
	value := formatRupiah(availableCapacity) + "/bln"
	if availableCapacity <= 0 {
		severity = SeverityWarning
		title = "Sudah lewat batas aman utang"
		message = fmt.Sprintf("Biaya tetap kamu (%s) sudah melebihi batas cicilan aman (35%% dari pemasukan, sekitar %s). Sebaiknya hindari cicilan/utang baru dulu.",
			formatRupiah(fv.Fixed), formatRupiah(maxSafeInstallment))
		value = "Kapasitas maksimal"
	}
	return []Signal{{
		ID:       "debt.borrowing_power",
		Category: CategoryDebt,
		Severity: severity,
		Title:    title,
		Message:  message,
		Value:    value,
		Facts: map[string]interface{}{
			"totalIncome": macro.TotalIncome, "fixedCost": fv.Fixed,
			"maxSafeInstallment": maxSafeInstallment, "availableCapacity": availableCapacity,
		},
	}}
}

// TopSpendingCategorySignal surfaces the single biggest-spend envelope --
// the most direct answer to "what can I cut to save more".
func TopSpendingCategorySignal(overview *budget.BudgetOverview) []Signal {
	if overview == nil || overview.TotalSpent <= 0 {
		return nil
	}
	var top *budget.EnvelopeOverview
	for i := range overview.Envelopes {
		e := &overview.Envelopes[i]
		if e.Spent > 0 && (top == nil || e.Spent > top.Spent) {
			top = e
		}
	}
	if top == nil {
		return nil
	}
	share := (top.Spent / overview.TotalSpent) * 100
	return []Signal{{
		ID:       "spending.top_category",
		Category: CategorySpending,
		Severity: SeverityNeutral,
		Title:    "Pengeluaran terbesar: " + top.Name,
		Message: fmt.Sprintf("%s adalah kategori dengan pengeluaran terbesar bulan ini, %s (%.0f%% dari total pengeluaran) -- kandidat paling masuk akal kalau mau dipangkas.",
			top.Name, formatRupiah(top.Spent), share),
		Value: formatRupiah(top.Spent),
		Facts: map[string]interface{}{
			"envelope": top.Name, "spent": top.Spent, "sharePercent": share,
		},
	}}
}

// WantsRatioSignal surfaces how much of total spend is discretionary
// ("wants" bucket) -- the most flexible lever to cut, as opposed to fixed
// or needs-based spending.
func WantsRatioSignal(overview *budget.BudgetOverview) []Signal {
	if overview == nil || overview.TotalSpent <= 0 {
		return nil
	}
	wantsRatio := (overview.TotalWants / overview.TotalSpent) * 100
	severity := SeverityNeutral
	title := "Porsi pengeluaran wants (non-esensial)"
	if wantsRatio > wantsRatioHighThreshold {
		severity = SeverityWarning
		title = "Porsi wants (non-esensial) cukup besar"
	}
	return []Signal{{
		ID:       "spending.wants_ratio",
		Category: CategorySpending,
		Severity: severity,
		Title:    title,
		Message: fmt.Sprintf("Pengeluaran wants (non-esensial) bulan ini %s (%.0f%% dari total) -- ini bagian paling fleksibel buat dikurangi dibanding kebutuhan pokok.",
			formatRupiah(overview.TotalWants), wantsRatio),
		Value: fmt.Sprintf("%.0f%%", wantsRatio),
		Facts: map[string]interface{}{
			"totalWants": overview.TotalWants, "totalSpent": overview.TotalSpent, "wantsRatioPercent": wantsRatio,
		},
	}}
}
