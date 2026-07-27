package dashboard


type SankeyNode struct {
	Name string `json:"name"`
}

type SankeyLink struct {
	Source string  `json:"source"`
	Target string  `json:"target"`
	Value  float64 `json:"value"`
}

type SankeyData struct {
	Nodes []SankeyNode `json:"nodes"`
	Links []SankeyLink `json:"links"`
}

type RadarDataPoint map[string]interface{}

type LiabilityProgress struct {
	Name         string  `json:"name"`
	TotalMonths  int     `json:"totalMonths"`
	MonthsPassed int     `json:"monthsPassed"`
	MonthsLeft   int     `json:"monthsLeft"`
	Progress     float64 `json:"progress"`
}

type TrendDataPoint struct {
	Date     string  `json:"date"`
	Label    string  `json:"label"`
	NetWorth float64 `json:"netWorth"`
	Income   float64 `json:"income"`
	Expense  float64 `json:"expense"`
	CashFlow float64 `json:"cashFlow"`
}

type FinancialHealthResponse struct {
	Trend           []TrendDataPoint    `json:"trend"`
	Granularity     string              `json:"granularity"`
	SankeyData      SankeyData          `json:"sankeyData"`
	RadarData       []RadarDataPoint    `json:"radarData"`
	Liabilities     []LiabilityProgress `json:"liabilities"`
	FixedVsVariable FixedVsVariable     `json:"fixedVsVariable"`
	MacroStats      MacroStats          `json:"macroStats"`
}

type MacroStats struct {
	CurrentNetWorth float64 `json:"currentNetWorth"`
	TotalIncome     float64 `json:"totalIncome"`
	TotalExpense    float64 `json:"totalExpense"`
	SavingsRate     float64 `json:"savingsRate"`
}
