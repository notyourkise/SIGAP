// ai-evaluate runs the course evaluation against an explicitly configured model.
// Results are evidence for human review, not a semantic grading claim.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sigap-backend/aiassistant"
	"time"
)

type testCase struct {
	ID       string   `json:"id"`
	Question string   `json:"pertanyaan"`
	Type     string   `json:"jenis"`
	Expected []string `json:"sumber_diharapkan"`
}
type result struct {
	ID             string                `json:"id"`
	Type           string                `json:"jenis"`
	Answer         *aiassistant.Response `json:"output,omitempty"`
	SourceCoverage bool                  `json:"semua_sumber_dikutip"`
	Error          string                `json:"error,omitempty"`
	SemanticReview string                `json:"penilaian_makna"`
}

func main() {
	path := flag.String("cases", "../ai-engineering/evaluasi/pertanyaan.json", "file kasus uji JSON")
	flag.Parse()
	if os.Getenv("AI_GATEWAY_KEY") == "" {
		fmt.Fprintln(os.Stderr, "AI_GATEWAY_KEY belum dikonfigurasi; tidak ada evaluasi model nyata yang dijalankan")
		os.Exit(2)
	}
	b, err := os.ReadFile(*path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var cases []testCase
	if err = json.Unmarshal(b, &cases); err != nil || len(cases) == 0 {
		fmt.Fprintln(os.Stderr, "file kasus tidak valid")
		os.Exit(1)
	}
	docs, err := aiassistant.Corpus()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	client := aiassistant.Client{BaseURL: os.Getenv("AI_GATEWAY_URL"), APIKey: os.Getenv("AI_GATEWAY_KEY"), Model: os.Getenv("AI_CHAT_MODEL"), EmbeddingModel: os.Getenv("AI_EMBEDDING_MODEL")}
	results := make([]result, 0, len(cases))
	for _, tc := range cases {
		ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
		answer, err := client.Answer(ctx, tc.Question, docs)
		cancel()
		r := result{ID: tc.ID, Type: tc.Type, SemanticReview: "belum dinilai manusia"}
		if err != nil {
			r.Error = err.Error()
		} else {
			r.Answer = &answer
			if tc.Type == "tidak_ada" {
				r.SourceCoverage = !answer.DapatDitentukan
			} else {
				seen := map[string]bool{}
				for _, citation := range answer.Bukti {
					seen[citation.Sumber] = true
				}
				r.SourceCoverage = answer.DapatDitentukan
				for _, expected := range tc.Expected {
					if !seen[expected] {
						r.SourceCoverage = false
					}
				}
			}
		}
		results = append(results, r)
	}
	output := map[string]interface{}{"waktu_utc": time.Now().UTC().Format(time.RFC3339), "model": client.Model, "embedding_model": client.EmbeddingModel, "hasil": results, "catatan": "Kecocokan kutipan bukan penilaian kebenaran semantik; semua jawaban perlu ditinjau manusia."}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
