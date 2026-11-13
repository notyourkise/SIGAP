package aiassistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
