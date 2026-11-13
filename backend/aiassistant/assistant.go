// Package aiassistant provides a bounded, read-only, source-grounded AI workflow.
package aiassistant

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Document struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type Citation struct {
	Sumber  string `json:"sumber"`
	Kutipan string `json:"kutipan"`
}
type Response struct {
	Jawaban         string     `json:"jawaban"`
	Bukti           []Citation `json:"bukti"`
	DapatDitentukan bool       `json:"dapat_ditentukan"`
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

func (c Client) Answer(ctx context.Context, question string, docs []Document) (Response, error) {
	if strings.TrimSpace(question) == "" || len([]rune(question)) > 2000 {
		return Response{}, errors.New("pertanyaan wajib diisi dan maksimal 2000 karakter")
	}
	if resp, ok := greetingResponse(question); ok {
		return resp, nil
	}
	if len(docs) == 0 {
		return Response{Jawaban: "Tidak dapat ditentukan dari dokumen yang tersedia.", DapatDitentukan: false}, nil
	}
	if len(docs) > 100 {
		return Response{}, errors.New("jumlah dokumen melebihi batas")
	}
	input := []string{question}
	for _, d := range docs {
		input = append(input, d.Text)
	}
	var vectors struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := c.post(ctx, "/embeddings", map[string]interface{}{"model": c.EmbeddingModel, "input": input}, &vectors); err != nil {
		return Response{}, err
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
	type hit struct {
		doc   Document
		score float64
	}
	hits := make([]hit, 0, len(docs))
	for i, d := range docs {
		score, err := cosine(ordered[0], ordered[i+1])
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
	for _, h := range hits {
		selected = append(selected, h.doc)
	}
	system := SystemPrompt
	messages := []interface{}{map[string]string{"role": "system", "content": system}, map[string]string{"role": "user", "content": question}}
	tool := func(name, description string) map[string]interface{} {
		return map[string]interface{}{"type": "function", "function": map[string]interface{}{"name": name, "description": description, "parameters": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}, "additionalProperties": false}}}
	}
	tools := []interface{}{tool("cari_dokumen", "Ambil hingga empat kutipan dokumen SIGAP yang paling relevan dengan pertanyaan."), tool("lihat_batas_monitoring", "Ambil hanya kutipan terpilih yang membahas monitoring atau verifikasi CCTV."), tool("daftar_sumber", "Lihat ID dokumen yang tersedia tanpa mengambil isi dokumen; jangan menggunakannya sebagai bukti jawaban.")}
	var decision struct {
		Choices []struct {
			Message struct {
				Role      string  `json:"role"`
				Content   *string `json:"content"`
				ToolCalls []struct {
					ID           string          `json:"id"`
					Type         string          `json:"type"`
					ExtraContent json.RawMessage `json:"extra_content,omitempty"`
					Function     struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := c.post(ctx, "/chat/completions", map[string]interface{}{"model": c.Model, "temperature": 0, "messages": messages, "tools": tools, "tool_choice": "required"}, &decision); err != nil {
		return Response{}, err
	}
	if len(decision.Choices) == 0 || len(decision.Choices[0].Message.ToolCalls) == 0 || len(decision.Choices[0].Message.ToolCalls) > 3 {
		return Response{}, errors.New("model tidak memilih tool yang tersedia")
	}
	msg := decision.Choices[0].Message
	messages = append(messages, map[string]interface{}{"role": "assistant", "content": msg.Content, "tool_calls": msg.ToolCalls})
	var exposed []Document
	for _, call := range msg.ToolCalls {
		if call.ID == "" || call.Type != "function" || (call.Function.Name != "cari_dokumen" && call.Function.Name != "lihat_batas_monitoring" && call.Function.Name != "daftar_sumber") {
			return Response{}, errors.New("tool tidak diizinkan")
		}
		var args map[string]interface{}
		if json.Unmarshal([]byte(call.Function.Arguments), &args) != nil || len(args) != 0 {
			return Response{}, errors.New("argumen tool tidak valid")
		}
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
			return Response{}, err
		}
		messages = append(messages, map[string]interface{}{"role": "tool", "tool_call_id": call.ID, "content": string(encoded)})
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := c.post(ctx, "/chat/completions", map[string]interface{}{"model": c.Model, "temperature": 0, "response_format": map[string]string{"type": "json_object"}, "messages": messages}, &result); err != nil {
		return Response{}, err
	}
	if len(result.Choices) == 0 {
		return Response{}, errors.New("model tidak mengembalikan jawaban")
	}
	var answer Response
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &answer); err != nil {
		return Response{}, errors.New("format jawaban model tidak valid")
	}
	if err := Validate(answer, exposed); err != nil {
		return Response{Jawaban: "Tidak dapat menentukan jawaban dengan sumber yang dapat diverifikasi.", DapatDitentukan: false, Bukti: []Citation{}}, nil
	}
	if !answer.DapatDitentukan {
		answer.Jawaban = "Tidak dapat menentukan jawaban berdasarkan dokumen yang tersedia."
		answer.Bukti = []Citation{}
	}
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
func greetingResponse(question string) (Response, bool) {
	cleaned := strings.ToLower(strings.TrimSpace(question))
	cleaned = strings.TrimRight(cleaned, "!?., ")
	switch cleaned {
	case "hai", "halo", "hallo", "hi", "hello", "hey",
		"selamat pagi", "selamat siang", "selamat sore", "selamat malam",
		"pagi", "siang", "sore", "malam", "assalamualaikum":
		return Response{
			Jawaban:         "Halo! Saya Asisten Operasional SIGAP. Silakan tanyakan seputar dokumentasi SIGAP (akses Docker, backup & restore, log stream CCTV, batasan codec/snapshot, verifikasi aktif, atau multi-site).",
			DapatDitentukan: false,
			Bukti:           []Citation{},
		}, true
	case "terima kasih", "makasih", "thanks", "thank you":
		return Response{
			Jawaban:         "Sama-sama! Silakan tanyakan kembali jika ada informasi dokumentasi SIGAP lain yang dibutuhkan.",
			DapatDitentukan: false,
			Bukti:           []Citation{},
		}, true
	default:
		return Response{}, false
	}
}
