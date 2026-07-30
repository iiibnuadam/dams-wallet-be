package insights

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ibnuadam/dams-wallet-backend/pkg/llm"
)

// DeepSeek's json_object mode guarantees syntactically valid JSON but does
// not enforce an external schema, so the exact expected shape is spelled
// out here as literal JSON for the model to follow.
const systemPrompt = `Kamu membantu menulis ulang fakta keuangan yang sudah dihitung menjadi bahasa yang lebih natural dan enak dibaca, dalam Bahasa Indonesia.

ATURAN PENTING:
- JANGAN mengubah, menambah, atau mengarang angka apa pun. Setiap angka di "narrative" harus persis sama dengan yang ada di "message" atau "facts" pada input.
- "narrative" adalah versi percakapan dari "message" -- tetap singkat (1-2 kalimat), tapi terasa seperti obrolan, bukan laporan.
- Setelah menulis ulang semua insight, buat 2-4 "talking point" berupa pertanyaan diskusi yang relevan (misalnya untuk dibahas bersama pasangan/keluarga), masing-masing mereferensikan id insight yang relevan lewat relatedSignalIds.
- Keluarkan HANYA JSON valid, tanpa markdown fence, tanpa penjelasan tambahan, persis mengikuti struktur berikut:

{
  "narratives": [
    {"id": "<id insight, sama persis dengan input>", "narrative": "<versi percakapan dari message>"}
  ],
  "talkingPoints": [
    {"question": "<pertanyaan diskusi>", "relatedSignalIds": ["<id insight terkait>"]}
  ]
}`

type narrateResult struct {
	Narratives []struct {
		ID        string `json:"id"`
		Narrative string `json:"narrative"`
	} `json:"narratives"`
	TalkingPoints []TalkingPoint `json:"talkingPoints"`
}

// llmSignalInput is the trimmed view of a Signal sent to the model -- only
// what it needs to rephrase, nothing about internal Narrative state.
type llmSignalInput struct {
	ID       string                 `json:"id"`
	Category SignalCategory         `json:"category"`
	Severity Severity               `json:"severity"`
	Title    string                 `json:"title"`
	Message  string                 `json:"message"`
	Value    string                 `json:"value"`
	Facts    map[string]interface{} `json:"facts"`
}

// NarrateBatch sends all signals in a single request and returns a map of
// signal ID -> narrative text, plus the generated talking points.
func NarrateBatch(ctx context.Context, client *llm.Client, signals []Signal) (map[string]string, []TalkingPoint, error) {
	input := make([]llmSignalInput, len(signals))
	for i, s := range signals {
		input[i] = llmSignalInput{
			ID: s.ID, Category: s.Category, Severity: s.Severity,
			Title: s.Title, Message: s.Message, Value: s.Value, Facts: s.Facts,
		}
	}
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, nil, fmt.Errorf("insights: marshal signals: %w", err)
	}

	raw, err := client.GenerateJSON(ctx, systemPrompt, string(payload))
	if err != nil {
		return nil, nil, err
	}

	var result narrateResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, nil, fmt.Errorf("insights: unmarshal LLM response: %w", err)
	}

	narrativeMap := make(map[string]string, len(result.Narratives))
	for _, n := range result.Narratives {
		narrativeMap[n.ID] = n.Narrative
	}
	return narrativeMap, result.TalkingPoints, nil
}
