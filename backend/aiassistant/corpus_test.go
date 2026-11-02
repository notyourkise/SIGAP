package aiassistant

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCorpusHasEightDistinctVerifiableSourceSnapshots(t *testing.T) {
	docs, err := Corpus()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 8 {
		t.Fatalf("expected eight source snapshots, got %d", len(docs))
	}
	seen := map[string]bool{}
	for _, doc := range docs {
		parts := strings.Split(doc.ID, "#")
		if len(parts) != 2 || seen[parts[0]] {
			t.Fatalf("invalid or duplicate source %q", doc.ID)
		}
		seen[parts[0]] = true
		original, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(parts[0])))
		if err != nil {
			t.Fatal(err)
		}
		normalized := strings.ReplaceAll(string(original), "\r\n", "\n")
		if !strings.Contains(normalized, doc.Text) {
			t.Fatalf("snapshot differs from cited source: %s", doc.ID)
		}
	}
}

func TestCitationValidationCases(t *testing.T) {
	docs := []Document{{ID: "source-a", Text: "SIGAP menyimpan data CCTV di database PostgreSQL."}, {ID: "source-b", Text: "Admin menugaskan teknisi melalui website."}}
	cases := []struct {
		name      string
		response  Response
		wantValid bool
	}{
		{"single source", Response{Jawaban: "Database PostgreSQL", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-a", Kutipan: "database PostgreSQL"}}}, true},
		{"second source", Response{Jawaban: "Admin menugaskan", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-b", Kutipan: "Admin menugaskan teknisi"}}}, true},
		{"two sources", Response{Jawaban: "Data dan tugas", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-a", Kutipan: "database PostgreSQL"}, {Sumber: "source-b", Kutipan: "Admin menugaskan teknisi"}}}, true},
		{"no evidence", Response{Jawaban: "Jawaban palsu", DapatDitentukan: true}, false},
		{"wrong source", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-b", Kutipan: "database PostgreSQL"}}}, false},
		{"invented source", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "secret", Kutipan: "database PostgreSQL"}}}, false},
		{"invented quote", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-a", Kutipan: "MySQL untuk semua kamera"}}}, false},
		{"short quote", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-a", Kutipan: "SIGAP"}}}, false},
		{"empty quote", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-a", Kutipan: ""}}}, false},
		{"blank answer", Response{Jawaban: "  ", DapatDitentukan: false}, false},
		{"unknown allowed", Response{Jawaban: "Tidak tahu", DapatDitentukan: false}, true},
		{"one bad citation", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-a", Kutipan: "database PostgreSQL"}, {Sumber: "source-b", Kutipan: "tidak terdapat pada sumber"}}}, false},
		{"exact capitalization required", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-a", Kutipan: "DATABASE POSTGRESQL"}}}, false},
		{"source id required", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Kutipan: "database PostgreSQL"}}}, false},
		{"quote contains extra space", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-a", Kutipan: "database  PostgreSQL"}}}, false},
		{"unknown with no sources", Response{Jawaban: "Tidak ada di sumber", DapatDitentukan: false, Bukti: []Citation{}}, true},
		{"valid quote cannot use wrong id", Response{Jawaban: "x", DapatDitentukan: true, Bukti: []Citation{{Sumber: "source-a#other", Kutipan: "database PostgreSQL"}}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := Validate(tc.response, docs)
			if (err == nil) != tc.wantValid {
				t.Fatalf("validation error %v; wantValid=%v", err, tc.wantValid)
			}
		})
	}
}
