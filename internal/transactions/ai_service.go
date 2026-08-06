package transactions

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ibnuadam/dams-wallet-backend/config"
	"github.com/ibnuadam/dams-wallet-backend/pkg/llm"
)



type ParsedTransactionResult struct {
	Amount       float64 `json:"amount"`
	Date         string  `json:"date"` // YYYY-MM-DD
	Description  string  `json:"description"`
	Type         string  `json:"type"` // INCOME | EXPENSE | TRANSFER
	CategoryName string  `json:"categoryName"` // Guessed category name
	WalletName   string  `json:"walletName"`   // Guessed wallet name
}

// ParseTransactionFromText uses the LLM to extract structured data from raw OCR text.
func ParseTransactionFromText(ctx context.Context, rawText string, availableCategories []string, availableWallets []string) ([]ParsedTransactionResult, error) {
	llmClient := llm.New(llm.Config{
		Provider:        config.App.LLMProvider,
		BaseURL:         config.App.LLMBaseURL,
		APIKey:          config.App.DeepSeekAPIKey, // Ensure we use the main API key
		Model:           config.App.LLMModel,
		Timeout:         time.Duration(config.App.LLMTimeoutSeconds) * time.Second,
		Temperature:     0.1, // Low temperature for deterministic extraction
		TopP:            0.95,
		ReasoningEffort: "low",
	})

	if !llmClient.Enabled() {
		return nil, fmt.Errorf("LLM client is not enabled. Please check DEEPSEEK_API_KEY configuration.")
	}

	catsBytes, _ := json.Marshal(availableCategories)
	walletsBytes, _ := json.Marshal(availableWallets)

	systemPrompt := `You are an AI assistant that extracts structured financial transaction data from raw, messy OCR text.
The OCR text comes from a receipt, invoice, or mobile banking screenshot, which may contain multiple transactions.

Extract ALL transactions found in the text and return them strictly as a JSON array matching this exact schema for each object (no markdown, no preamble):
[
  {
    "amount": <number, float, the total or most significant amount, no currency symbols>,
    "date": "<string, YYYY-MM-DD format, guess from text if possible, else empty string>",
    "description": "<string, a short meaningful description of the transaction>",
    "type": "<string, one of: 'INCOME', 'EXPENSE', 'TRANSFER'. Default to EXPENSE for receipts>",
    "categoryName": "<string, the closest matching category from the available list, or empty if unknown>",
    "walletName": "<string, the closest matching wallet from the available list, or empty if unknown>"
  }
]

Available Categories: ` + string(catsBytes) + `
Available Wallets: ` + string(walletsBytes) + `

Guidelines:
- Strip all formatting and return raw JSON.
- If it's a receipt (Grab, Gojek, Indomaret, etc.), it's likely an EXPENSE.
- If it's a transfer receipt to someone else, it's an EXPENSE.
- If it's a transfer receipt receiving money, it's an INCOME.
- If it looks like a transfer between own accounts, it's a TRANSFER.
- For "amount", return only the number. Example: 150000.00
- If the text contains multiple distinct transactions (like a bank statement list), extract each one as a separate object in the array.
`

	userPrompt := "Here is the raw OCR text:\n\n" + rawText

	jsonResp, err := llmClient.GenerateJSON(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JSON from LLM: %w", err)
	}

	var results []ParsedTransactionResult
	if err := json.Unmarshal(jsonResp, &results); err != nil {
		// Sometimes LLM might return a single object instead of an array if it only found one,
		// let's try falling back to a single object parse just in case
		var singleResult ParsedTransactionResult
		if errSingle := json.Unmarshal(jsonResp, &singleResult); errSingle == nil {
			return []ParsedTransactionResult{singleResult}, nil
		}

		return nil, fmt.Errorf("failed to parse LLM JSON response as array: %w\nRaw response: %s", err, string(jsonResp))
	}

	return results, nil
}
