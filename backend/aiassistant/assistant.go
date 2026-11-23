// Package aiassistant provides a bounded, read-only, source-grounded AI workflow.
package aiassistant

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxDocEmbeddingCacheEntries = 1024

var docEmbeddingCache = struct {
	mu      sync.RWMutex
	entries map[string][]float64
}{
	entries: make(map[string][]float64),
}

func docEmbeddingCacheKey(baseURL, model string, d Document) string {
	sum := sha256.Sum256([]byte(d.ID + "\x00" + d.Text))
	return strings.TrimRight(baseURL, "/") + "\x00" + model + "\x00" + hex.EncodeToString(sum[:])
}

func getCachedDocEmbedding(key string) ([]float64, bool) {
	docEmbeddingCache.mu.RLock()
	vec, ok := docEmbeddingCache.entries[key]
	docEmbeddingCache.mu.RUnlock()
	return vec, ok
}

func putCachedDocEmbedding(key string, vec []float64) {
	if len(vec) == 0 {
		return
	}
	copied := make([]float64, len(vec))
	copy(copied, vec)
	docEmbeddingCache.mu.Lock()
	if len(docEmbeddingCache.entries) >= maxDocEmbeddingCacheEntries {
		docEmbeddingCache.entries = make(map[string][]float64)
	}
	docEmbeddingCache.entries[key] = copied
	docEmbeddingCache.mu.Unlock()
}

type Document struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type Citation struct {
	Sumber  string `json:"sumber"`
	Kutipan string `json:"kutipan"`
}

const (
	DefaultMaxAgentSteps = 4
	ModeFull             = "penuh"
	ModeConcise          = "ringkas"
)

type AnswerOptions struct {
	Mode                   string
	MaxSteps               int
	AllowMultiStepToolLoop bool
}

type UsageStats struct {
	EmbeddingPromptTokens int      `json:"embedding_prompt_tokens"`
	EmbeddingCacheHits    int      `json:"embedding_cache_hits"`
	ChatPromptTokens      int      `json:"chat_prompt_tokens"`
	ChatCompletionTokens  int      `json:"chat_completion_tokens"`
	TotalTokens           int      `json:"total_tokens"`
	StepsUsed             int      `json:"steps_used"`
	ToolsCalled           []string `json:"tools_called,omitempty"`
	MaxStepsReached       bool     `json:"max_steps_reached,omitempty"`
	Mode                  string   `json:"mode"`
	LatencyMs             int64    `json:"latency_ms"`
}

type Response struct {
	Jawaban         string      `json:"jawaban"`
	Bukti           []Citation  `json:"bukti"`
	DapatDitentukan bool        `json:"dapat_ditentukan"`
	Usage           *UsageStats `json:"usage,omitempty"`
}
type Client struct {
	BaseURL, APIKey, Model, EmbeddingModel string
	HTTP                                   *http.Client
}

// Validate requires exact source excerpts, preventing fabricated references.
func Validate(r Response, docs []Document) error {
	if strings.TrimSpace(r.Jawaban) == "" {
		return errors.New("jawaban kosong")
	}
	if !r.DapatDitentukan {
		return nil
	}
	if len(r.Bukti) == 0 {
		return errors.New("jawaban tidak memiliki sumber")
	}
	for _, cite := range r.Bukti {
		valid := false
		for _, doc := range docs {
			if cite.Sumber == doc.ID && len([]rune(strings.TrimSpace(cite.Kutipan))) >= 8 && strings.Contains(doc.Text, cite.Kutipan) {
				valid = true
				break
			}
		}
		if !valid {
			return errors.New("kutipan tidak sesuai sumber")
		}
	}
	return nil
}

func (c Client) post(ctx context.Context, path string, body interface{}, out interface{}) error {
	u, err := url.Parse(c.BaseURL)
	isLocalHTTP := u != nil && u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1")
	if err != nil || u.Host == "" || (u.Scheme != "https" && !isLocalHTTP) {
		return errors.New("alamat model gateway tidak aman")
	}
	if c.APIKey == "" || c.Model == "" || c.EmbeddingModel == "" {
		return errors.New("model gateway belum dikonfigurasi")
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	res, err := httpClient.Do(req)
	if err != nil {
		return errors.New("model gateway tidak tersedia")
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != 200 {
		return fmt.Errorf("model gateway gagal (HTTP %d)", res.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(out); err != nil {
		return errors.New("respons gateway tidak valid")
	}
	return nil
}

type apiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type toolCallMsg struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	ExtraContent json.RawMessage `json:"extra_content,omitempty"`
	Function     struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type chatChoiceMsg struct {
	Role      string        `json:"role"`
	Content   *string       `json:"content"`
	ToolCalls []toolCallMsg `json:"tool_calls"`
}

func normalizeAnswerOptions(opts AnswerOptions) AnswerOptions {
	if strings.EqualFold(strings.TrimSpace(opts.Mode), ModeConcise) {
		opts.Mode = ModeConcise
	} else {
		opts.Mode = ModeFull
	}
	if opts.MaxSteps <= 0 {
		opts.MaxSteps = DefaultMaxAgentSteps
	}
	if opts.MaxSteps > 8 {
		opts.MaxSteps = 8
	}
	return opts
}

func (c Client) Answer(ctx context.Context, question string, docs []Document) (Response, error) {
	return c.AnswerWithOptions(ctx, question, docs, AnswerOptions{
		Mode:     ModeFull,
		MaxSteps: DefaultMaxAgentSteps,
	})
}

func (c Client) AnswerWithOptions(ctx context.Context, question string, docs []Document, rawOpts AnswerOptions) (Response, error) {
	started := time.Now()
	opts := normalizeAnswerOptions(rawOpts)
	usage := UsageStats{Mode: opts.Mode}

	if strings.TrimSpace(question) == "" || len([]rune(question)) > 2000 {
		return Response{}, errors.New("pertanyaan wajib diisi dan maksimal 2000 karakter")
	}
	if resp, ok := greetingResponse(question, opts.Mode); ok {
		usage.LatencyMs = time.Since(started).Milliseconds()
		resp.Usage = &usage
		return resp, nil
	}
	if len(docs) == 0 {
		usage.LatencyMs = time.Since(started).Milliseconds()
		return Response{Jawaban: "Tidak dapat ditentukan dari dokumen yang tersedia.", DapatDitentukan: false, Bukti: []Citation{}, Usage: &usage}, nil
	}
	if len(docs) > 100 {
		return Response{}, errors.New("jumlah dokumen melebihi batas")
	}
	docVectors := make([][]float64, len(docs))
	uncachedDocIndices := make([]int, 0, len(docs))
	uncachedKeys := make([]string, 0, len(docs))
	input := make([]string, 1, 1+len(docs))
	input[0] = question
	for i, d := range docs {
		key := docEmbeddingCacheKey(c.BaseURL, c.EmbeddingModel, d)
		if vec, ok := getCachedDocEmbedding(key); ok && len(vec) > 0 {
			docVectors[i] = vec
			usage.EmbeddingCacheHits++
			continue
		}
		uncachedDocIndices = append(uncachedDocIndices, i)
		uncachedKeys = append(uncachedKeys, key)
		input = append(input, d.Text)
	}
	var vectors struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
		Usage apiUsage `json:"usage"`
	}
	if err := c.post(ctx, "/embeddings", map[string]interface{}{"model": c.EmbeddingModel, "input": input}, &vectors); err != nil {
		return Response{}, err
	}
	usage.EmbeddingPromptTokens += vectors.Usage.PromptTokens
	if vectors.Usage.TotalTokens > 0 {
		usage.TotalTokens += vectors.Usage.TotalTokens
	} else {
		usage.TotalTokens += vectors.Usage.PromptTokens
	}
	if len(vectors.Data) != len(input) {
		return Response{}, errors.New("jumlah embedding tidak sesuai")
	}
	ordered := make([][]float64, len(input))
	for _, v := range vectors.Data {
		if v.Index < 0 || v.Index >= len(input) || ordered[v.Index] != nil {
			return Response{}, errors.New("indeks embedding tidak valid")
		}
		ordered[v.Index] = v.Embedding
	}
	for pos, docIdx := range uncachedDocIndices {
		vec := ordered[pos+1]
		docVectors[docIdx] = vec
		putCachedDocEmbedding(uncachedKeys[pos], vec)
	}
	type hit struct {
		doc   Document
		score float64
	}
	hits := make([]hit, 0, len(docs))
	for i, d := range docs {
		score, err := cosine(ordered[0], docVectors[i])
		if err != nil {
			return Response{}, err
		}
		hits = append(hits, hit{d, score})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].score > hits[j].score })
	if len(hits) > 4 {
		hits = hits[:4]
	}
	selected := make([]Document, 0, len(hits))
	selectedIDs := make(map[string]bool, len(hits))
	for _, h := range hits {
		selected = append(selected, h.doc)
		selectedIDs[h.doc.ID] = true
	}
	// Sertakan dokumen ringkasan live bila tersedia agar pertanyaan operasional tidak tersisih dari top-4.
	for _, d := range docs {
		if strings.HasPrefix(d.ID, "live-db://") && !selectedIDs[d.ID] {
			if len(selected) >= 4 {
				selected[len(selected)-1] = d
			} else {
				selected = append(selected, d)
			}
			selectedIDs[d.ID] = true
		}
	}
	system := SystemPrompt
	if opts.Mode == ModeConcise {
		system += ` Mode jawaban RINGKAS aktif: tulis field "jawaban" maksimal 2-3 kalimat padat atau 3 poin singkat tanpa pengantar panjang, tetap isi "bukti" dan "dapat_ditentukan" persis sesuai aturan.`
	}
	messages := []interface{}{map[string]string{"role": "system", "content": system}, map[string]string{"role": "user", "content": question}}
	tool := func(name, description string) map[string]interface{} {
		return map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": name, "description": description, "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "additionalProperties": false}}}
	}
	tools := []interface{}{tool("cari_dokumen", "Ambil hingga empat kutipan dokumen SIGAP yang paling relevan dengan pertanyaan."), tool("lihat_batas_monitoring", "Ambil hanya kutipan terpilih yang membahas monitoring atau verifikasi CCTV."), tool("daftar_sumber", "Lihat ID dokumen yang tersedia tanpa mengambil isi dokumen; jangan menggunakannya sebagai bukti jawaban.")}

	var exposed []Document
	executeTools := func(calls []toolCallMsg) error {
		if len(calls) > 3 {
			return errors.New("jumlah tool call melebihi batas")
		}
		for _, call := range calls {
			if call.ID == "" || call.Type != "function" || (call.Function.Name != "cari_dokumen" && call.Function.Name != "lihat_batas_monitoring" && call.Function.Name != "daftar_sumber") {
				return errors.New("tool tidak diizinkan")
			}
			if strings.TrimSpace(call.Function.Arguments) != "" {
				var args map[string]interface{}
				if json.Unmarshal([]byte(call.Function.Arguments), &args) != nil || len(args) != 0 {
					return errors.New("argumen tool tidak valid")
				}
			}
			usage.ToolsCalled = append(usage.ToolsCalled, call.Function.Name)
			var output interface{}
			switch call.Function.Name {
			case "cari_dokumen":
				output = selected
				exposed = append(exposed, selected...)
			case "lihat_batas_monitoring":
				filtered := []Document{}
				for _, doc := range selected {
					lower := strings.ToLower(doc.Text)
					if strings.Contains(lower, "monitoring") || strings.Contains(lower, "verifikasi") {
						filtered = append(filtered, doc)
					}
				}
				output = filtered
				exposed = append(exposed, filtered...)
			case "daftar_sumber":
				ids := make([]string, 0, len(docs))
				for _, doc := range docs {
					ids = append(ids, doc.ID)
				}
				output = ids
			}
			encoded, err := json.Marshal(output)
			if err != nil {
				return err
			}
			messages = append(messages, map[string]interface{}{"role": "tool", "tool_call_id": call.ID, "content": string(encoded)})
		}
		return nil
	}

	var rawFinalContent string
	for step := 1; step <= opts.MaxSteps; step++ {
		if step > 1 && !opts.AllowMultiStepToolLoop {
			break
		}
		usage.StepsUsed = step
		toolChoice := "auto"
		if step == 1 {
			toolChoice = "required"
		}
		var stepRes struct {
			Choices []struct {
				Message chatChoiceMsg `json:"message"`
			} `json:"choices"`
			Usage apiUsage `json:"usage"`
		}
		if err := c.post(ctx, "/chat/completions", map[string]interface{}{"model": c.Model, "temperature": 0, "messages": messages, "tools": tools, "tool_choice": toolChoice}, &stepRes); err != nil {
			return Response{}, err
		}
		usage.ChatPromptTokens += stepRes.Usage.PromptTokens
		usage.ChatCompletionTokens += stepRes.Usage.CompletionTokens
		if stepRes.Usage.TotalTokens > 0 {
			usage.TotalTokens += stepRes.Usage.TotalTokens
		} else {
			usage.TotalTokens += stepRes.Usage.PromptTokens + stepRes.Usage.CompletionTokens
		}
		if len(stepRes.Choices) == 0 {
			return Response{}, errors.New("model tidak mengembalikan pilihan respons")
		}
		msg := stepRes.Choices[0].Message
		if len(msg.ToolCalls) == 0 {
			if step == 1 {
				return Response{}, errors.New("model tidak memilih tool yang tersedia")
			}
			if msg.Content != nil {
				rawFinalContent = strings.TrimSpace(*msg.Content)
			}
			break
		}
		messages = append(messages, map[string]interface{}{"role": "assistant", "content": msg.Content, "tool_calls": msg.ToolCalls})
		if err := executeTools(msg.ToolCalls); err != nil {
			return Response{}, err
		}
		if step == opts.MaxSteps && opts.AllowMultiStepToolLoop {
			usage.MaxStepsReached = true
		}
	}

	var answer Response
	parsedFinal := false
	if rawFinalContent != "" {
		if err := json.Unmarshal([]byte(rawFinalContent), &answer); err == nil {
			parsedFinal = true
		}
	}
	if !parsedFinal {
		usage.StepsUsed++
		var result struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
			Usage apiUsage `json:"usage"`
		}
		if err := c.post(ctx, "/chat/completions", map[string]interface{}{"model": c.Model, "temperature": 0, "response_format": map[string]string{"type": "json_object"}, "messages": messages}, &result); err != nil {
			return Response{}, err
		}
		usage.ChatPromptTokens += result.Usage.PromptTokens
		usage.ChatCompletionTokens += result.Usage.CompletionTokens
		if result.Usage.TotalTokens > 0 {
			usage.TotalTokens += result.Usage.TotalTokens
		} else {
			usage.TotalTokens += result.Usage.PromptTokens + result.Usage.CompletionTokens
		}
		if len(result.Choices) == 0 {
			return Response{}, errors.New("model tidak mengembalikan jawaban")
		}
		if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &answer); err != nil {
			return Response{}, errors.New("format jawaban model tidak valid")
		}
	}

	usage.LatencyMs = time.Since(started).Milliseconds()
	if err := Validate(answer, exposed); err != nil {
		return Response{Jawaban: "Tidak dapat menentukan jawaban dengan sumber yang dapat diverifikasi.", DapatDitentukan: false, Bukti: []Citation{}, Usage: &usage}, nil
	}
	if !answer.DapatDitentukan {
		answer.Jawaban = "Tidak dapat menentukan jawaban berdasarkan dokumen yang tersedia."
		answer.Bukti = []Citation{}
	}
	answer.Usage = &usage
	return answer, nil
}

func cosine(a, b []float64) (float64, error) {
	if len(a) == 0 || len(a) != len(b) {
		return 0, errors.New("dimensi embedding tidak sesuai")
	}
	var dot, x, y float64
	for i := range a {
		dot += a[i] * b[i]
		x += a[i] * a[i]
		y += b[i] * b[i]
	}
	if x == 0 || y == 0 {
		return 0, errors.New("embedding kosong")
	}
	return dot / (math.Sqrt(x) * math.Sqrt(y)), nil
}

// greetingResponse handles pure conversational greetings/thanks without invoking the RAG gateway,
// keeping factual document grounding untouched for any substantive query.
func greetingResponse(question, mode string) (Response, bool) {
	cleaned := strings.ToLower(strings.TrimSpace(question))
	cleaned = strings.TrimRight(cleaned, "!?., ")
	switch cleaned {
	case "hai", "halo", "hallo", "hi", "hello", "hey",
		"selamat pagi", "selamat siang", "selamat sore", "selamat malam",
		"pagi", "siang", "sore", "malam", "assalamualaikum":
		msg := "Halo! Saya Asisten Operasional SIGAP. Silakan tanyakan seputar dokumentasi SIGAP (akses Docker, backup & restore, log stream CCTV, batasan codec/snapshot, verifikasi aktif, atau multi-site)."
		if mode == ModeConcise {
			msg = "Halo! Asisten SIGAP siap menjawab pertanyaan dokumentasi atau ringkasan CCTV secara ringkas."
		}
		return Response{
			Jawaban:         msg,
			DapatDitentukan: false,
			Bukti:           []Citation{},
		}, true
	case "terima kasih", "makasih", "thanks", "thank you":
		msg := "Sama-sama! Silakan tanyakan kembali jika ada informasi dokumentasi SIGAP lain yang dibutuhkan."
		if mode == ModeConcise {
			msg = "Sama-sama! Ada pertanyaan lain?"
		}
		return Response{
			Jawaban:         msg,
			DapatDitentukan: false,
			Bukti:           []Citation{},
		}, true
	default:
		return Response{}, false
	}
}
