package aiassistant

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed corpus.json
var corpusJSON []byte

//go:embed prompts/v2.txt
var SystemPrompt string

// Corpus contains bounded, source-attributed snapshots, not live system state.
func Corpus() ([]Document, error) {
	var docs []Document
	if err := json.Unmarshal(corpusJSON, &docs); err != nil {
		return nil, fmt.Errorf("corpus inválido: %w", err)
	}
	if len(docs) < 5 {
		return nil, fmt.Errorf("corpus insuficiente")
	}
	return docs, nil
}
