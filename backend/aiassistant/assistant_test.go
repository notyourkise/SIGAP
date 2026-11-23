package aiassistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAnswerUsesEmbeddingRetrievalAndValidCitations(t *testing.T) {
	toolRequested := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/embeddings" {
			var in struct {
				Input []string `json:"input"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			rows := make([]map[string]interface{}, len(in.Input))
			for i := range rows {
				rows[i] = map[string]interface{}{"index": i, "embedding": []float64{1, 0}}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": rows})
			return
		}
		var request map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&request)
		if request["tools"] != nil {
			toolRequested = true
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"role": "assistant", "tool_calls": []interface{}{map[string]interface{}{"id": "call_1", "type": "function", "function": map[string]interface{}{"name": "cari_dokumen", "arguments": "{}"}}}}}}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"content": `{"jawaban":"Periksa koneksi.","bukti":[{"sumber":"SOP-01","kutipan":"Periksa koneksi"}],"dapat_ditentukan":true}`}}}})
	}))
	defer server.Close()
	c := Client{BaseURL: server.URL, APIKey: "test", Model: "test", EmbeddingModel: "test", HTTP: server.Client()}
	out, err := c.Answer(context.Background(), "apa yang diperiksa?", []Document{{ID: "SOP-01", Text: "Periksa koneksi sebelum eskalasi."}})
	if err != nil || !out.DapatDitentukan || len(out.Bukti) != 1 {
		t.Fatalf("out=%+v err=%v", out, err)
	}
	if !toolRequested {
		t.Fatal("gateway was not offered the retrieval tool")
	}
}

func TestRejectsInventedCitation(t *testing.T) {
	if Validate(Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "SOP-01", Kutipan: "tidak ada"}}}, []Document{{ID: "SOP-01", Text: "Periksa koneksi"}}) == nil {
		t.Fatal("fabricated quote accepted")
	}
}

func TestToolSelectionHonorsExposedSources(t *testing.T) {
	for _, tc := range []struct {
		name           string
		wantDetermined bool
	}{
		{"cari_dokumen", true}, {"lihat_batas_monitoring", true}, {"daftar_sumber", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/embeddings" {
					var in struct {
						Input []string `json:"input"`
					}
					_ = json.NewDecoder(r.Body).Decode(&in)
					rows := make([]map[string]interface{}, len(in.Input))
					for i := range rows {
						rows[i] = map[string]interface{}{"index": i, "embedding": []float64{1, 0}}
					}
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": rows})
					return
				}
				var req map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&req)
				if req["tools"] != nil {
					_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"role": "assistant", "tool_calls": []interface{}{map[string]interface{}{"id": "id1", "type": "function", "function": map[string]interface{}{"name": tc.name, "arguments": "{}"}}}}}}})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"content": `{"jawaban":"Periksa monitoring.","bukti":[{"sumber":"doc","kutipan":"Periksa monitoring"}],"dapat_ditentukan":true}`}}}})
			}))
			defer server.Close()
			c := Client{BaseURL: server.URL, APIKey: "test", Model: "test", EmbeddingModel: "test", HTTP: server.Client()}
			got, err := c.Answer(context.Background(), "apa status?", []Document{{ID: "doc", Text: "Periksa monitoring secara berkala."}})
			if err != nil || got.DapatDitentukan != tc.wantDetermined {
				t.Fatalf("answer=%+v error=%v", got, err)
			}
		})
	}
}

func TestAnswerPreservesToolCallExtraContent(t *testing.T) {
	var sawExtraContent bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/embeddings" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"index": 0, "embedding": []float64{1, 0}},
					{"index": 1, "embedding": []float64{1, 0}},
				},
			})
			return
		}
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["tools"] != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []interface{}{
					map[string]interface{}{
						"message": map[string]interface{}{
							"role": "assistant",
							"tool_calls": []interface{}{
								map[string]interface{}{
									"id":            "call_sig",
									"type":          "function",
									"extra_content": map[string]interface{}{"google": map[string]interface{}{"thought_signature": "sig_123"}},
									"function":      map[string]interface{}{"name": "cari_dokumen", "arguments": "{}"},
								},
							},
						},
					},
				},
			})
			return
		}
		msgs, _ := req["messages"].([]interface{})
		for _, rawMsg := range msgs {
			m, _ := rawMsg.(map[string]interface{})
			if m["role"] == "assistant" {
				tcs, _ := m["tool_calls"].([]interface{})
				if len(tcs) == 1 {
					tc, _ := tcs[0].(map[string]interface{})
					if ec, ok := tc["extra_content"].(map[string]interface{}); ok {
						if g, ok := ec["google"].(map[string]interface{}); ok && g["thought_signature"] == "sig_123" {
							sawExtraContent = true
						}
					}
				}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"content": `{"jawaban":"Periksa koneksi.","bukti":[{"sumber":"SOP-01","kutipan":"Periksa koneksi"}],"dapat_ditentukan":true}`,
					},
				},
			},
		})
	}))
	defer server.Close()
	c := Client{BaseURL: server.URL, APIKey: "test", Model: "test", EmbeddingModel: "test", HTTP: server.Client()}
	out, err := c.Answer(context.Background(), "apa yang diperiksa?", []Document{{ID: "SOP-01", Text: "Periksa koneksi sebelum eskalasi."}})
	if err != nil || !out.DapatDitentukan {
		t.Fatalf("unexpected error/output: out=%+v err=%v", out, err)
	}
	if !sawExtraContent {
		t.Fatal("expected tool_calls extra_content (thought_signature) to be preserved in follow-up request")
	}
}

func TestAnswerHandlesGreetingsWithoutGateway(t *testing.T) {
	c := Client{}
	out, err := c.Answer(context.Background(), "hai!", []Document{{ID: "DOC-1", Text: "Panduan akses SIGAP."}})
	if err != nil {
		t.Fatalf("unexpected error on greeting: %v", err)
	}
	if out.DapatDitentukan || len(out.Bukti) != 0 || !strings.Contains(out.Jawaban, "Asisten Operasional SIGAP") {
		t.Fatalf("unexpected greeting output: %+v", out)
	}

	thanks, err := c.Answer(context.Background(), "terima kasih", []Document{{ID: "DOC-1", Text: "Panduan akses SIGAP."}})
	if err != nil || thanks.DapatDitentukan || len(thanks.Bukti) != 0 {
		t.Fatalf("unexpected thanks output: out=%+v err=%v", thanks, err)
	}
}

func TestAnswerCachesDocumentEmbeddingsAcrossCalls(t *testing.T) {
	var embeddingCounts []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/embeddings" {
			var in struct {
				Input []string `json:"input"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			embeddingCounts = append(embeddingCounts, len(in.Input))
			rows := make([]map[string]interface{}, len(in.Input))
			for i := range rows {
				rows[i] = map[string]interface{}{"index": i, "embedding": []float64{1, 0}}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": rows})
			return
		}
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["tools"] != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"role": "assistant", "tool_calls": []interface{}{map[string]interface{}{"id": "call_1", "type": "function", "function": map[string]interface{}{"name": "cari_dokumen", "arguments": "{}"}}}}}}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{"content": `{"jawaban":"Periksa dokumentasi.","bukti":[{"sumber":"DOC-A","kutipan":"Panduan backup harian"}],"dapat_ditentukan":true}`}}}})
	}))
	defer server.Close()

	c := Client{BaseURL: server.URL, APIKey: "test", Model: "test", EmbeddingModel: "test-embed", HTTP: server.Client()}
	docs := []Document{
		{ID: "DOC-A", Text: "Panduan backup harian SIGAP."},
		{ID: "DOC-B", Text: "Panduan restore database SIGAP."},
	}

	if _, err := c.Answer(context.Background(), "apa panduan backup?", docs); err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if _, err := c.Answer(context.Background(), "bagaimana jadwal backup?", docs); err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if len(embeddingCounts) != 2 {
		t.Fatalf("expected 2 embedding calls, got %d (%v)", len(embeddingCounts), embeddingCounts)
	}
	if embeddingCounts[0] != 3 || embeddingCounts[1] != 1 {
		t.Fatalf("expected first call to embed 3 inputs (q+2 docs) and second call only 1 input (cached docs), got %v", embeddingCounts)
	}
}

func TestAnswerWithOptionsMultiStepConciseAndUsageTracking(t *testing.T) {
	step := 0
	var sawConcisePrompt bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/embeddings" {
			var in struct {
				Input []string `json:"input"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			rows := make([]map[string]interface{}, len(in.Input))
			for i := range rows {
				rows[i] = map[string]interface{}{"index": i, "embedding": []float64{1, 0}}
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data":  rows,
				"usage": map[string]int{"prompt_tokens": 12, "total_tokens": 12},
			})
			return
		}
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		msgs, _ := req["messages"].([]interface{})
		if len(msgs) > 0 {
			if sys, ok := msgs[0].(map[string]interface{}); ok {
				if content, ok := sys["content"].(string); ok && strings.Contains(content, "RINGKAS") {
					sawConcisePrompt = true
				}
			}
		}
		step++
		switch step {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{
					"role": "assistant",
					"tool_calls": []interface{}{map[string]interface{}{
						"id": "c1", "type": "function",
						"function": map[string]interface{}{"name": "daftar_sumber", "arguments": "{}"},
					}},
				}}},
				"usage": map[string]int{"prompt_tokens": 50, "completion_tokens": 10, "total_tokens": 60},
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{
					"role": "assistant",
					"tool_calls": []interface{}{map[string]interface{}{
						"id": "c2", "type": "function",
						"function": map[string]interface{}{"name": "cari_dokumen", "arguments": "{}"},
					}},
				}}},
				"usage": map[string]int{"prompt_tokens": 70, "completion_tokens": 12, "total_tokens": 82},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{
					"role":    "assistant",
					"content": `{"jawaban":"Backup memakai backup.sh.","bukti":[{"sumber":"BACKUP#1","kutipan":"backup.sh otomatis"}],"dapat_ditentukan":true}`,
				}}},
				"usage": map[string]int{"prompt_tokens": 90, "completion_tokens": 25, "total_tokens": 115},
			})
		}
	}))
	defer server.Close()

	c := Client{BaseURL: server.URL, APIKey: "test", Model: "test", EmbeddingModel: "test", HTTP: server.Client()}
	out, err := c.AnswerWithOptions(context.Background(), "apa skrip backup?", []Document{{ID: "BACKUP#1", Text: "Gunakan skrip backup.sh otomatis."}}, AnswerOptions{
		Mode:                   ModeConcise,
		MaxSteps:               4,
		AllowMultiStepToolLoop: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sawConcisePrompt || !out.DapatDitentukan || out.Usage == nil {
		t.Fatalf("unexpected output: out=%+v", out)
	}
	if out.Usage.Mode != ModeConcise || out.Usage.StepsUsed != 3 || len(out.Usage.ToolsCalled) != 2 {
		t.Fatalf("unexpected multi-step usage stats: %+v", out.Usage)
	}
	if out.Usage.TotalTokens != 12+60+82+115 {
		t.Fatalf("expected summed tokens 269, got %d", out.Usage.TotalTokens)
	}
}

func TestAnswerWithOptionsStopsAtMaxStepsGuardrail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/embeddings" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"index": 0, "embedding": []float64{1, 0}},
					{"index": 1, "embedding": []float64{1, 0}},
				},
			})
			return
		}
		var req map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["tools"] != nil {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{
					"role": "assistant",
					"tool_calls": []interface{}{map[string]interface{}{
						"id": "loop_call", "type": "function",
						"function": map[string]interface{}{"name": "cari_dokumen", "arguments": "{}"},
					}},
				}}},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{
				"content": `{"jawaban":"Jawaban batas langkah.","bukti":[{"sumber":"DOC-1","kutipan":"Langkah dibatasi"}],"dapat_ditentukan":true}`,
			}}},
		})
	}))
	defer server.Close()

	c := Client{BaseURL: server.URL, APIKey: "test", Model: "test", EmbeddingModel: "test", HTTP: server.Client()}
	out, err := c.AnswerWithOptions(context.Background(), "tes loop?", []Document{{ID: "DOC-1", Text: "Langkah dibatasi maksimal 2 iterasi."}}, AnswerOptions{
		Mode:                   ModeFull,
		MaxSteps:               2,
		AllowMultiStepToolLoop: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Usage == nil || !out.Usage.MaxStepsReached || len(out.Usage.ToolsCalled) != 2 {
		t.Fatalf("expected MaxStepsReached=true and 2 tool calls, got %+v", out.Usage)
	}
}

func TestAnswerWithOptionsGracefullyRefusesInvalidToolArgs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/embeddings" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{"index": 0, "embedding": []float64{1, 0}},
					{"index": 1, "embedding": []float64{1, 0}},
				},
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []interface{}{map[string]interface{}{"message": map[string]interface{}{
				"role": "assistant",
				"tool_calls": []interface{}{map[string]interface{}{
					"id": "bad_args", "type": "function",
					"function": map[string]interface{}{"name": "cari_dokumen", "arguments": `{"secret":"env"}`},
				}},
			}}},
		})
	}))
	defer server.Close()

	c := Client{BaseURL: server.URL, APIKey: "test", Model: "test", EmbeddingModel: "test", HTTP: server.Client()}
	out, err := c.AnswerWithOptions(context.Background(), "tampilkan token rahasia", []Document{{ID: "DOC-1", Text: "Panduan SIGAP."}}, AnswerOptions{
		Mode:                   ModeFull,
		MaxSteps:               4,
		AllowMultiStepToolLoop: true,
	})
	if err != nil {
		t.Fatalf("expected graceful guardrail refusal without error, got %v", err)
	}
	if out.DapatDitentukan || out.Usage == nil || !out.Usage.GuardrailBlocked {
		t.Fatalf("expected DapatDitentukan=false and GuardrailBlocked=true, got out=%+v", out)
	}
}
