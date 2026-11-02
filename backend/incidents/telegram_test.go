package incidents

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelegramSenderChecksAPIResultAndChat(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		wantError  bool
	}{
		{"success", `{"ok":true,"result":{"message_id":42,"chat":{"id":-100}}}`, 200, false},
		{"api rejection", `{"ok":false,"description":"bad"}`, 200, true},
		{"rate limit", `{"ok":false,"parameters":{"retry_after":60}}`, 429, true},
		{"wrong destination", `{"ok":true,"result":{"message_id":42,"chat":{"id":-200}}}`, 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Error("expected POST")
				}
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			sender := TelegramSender{Token: "test", Client: server.Client(), baseURL: server.URL}
			id, err := sender.Send(context.Background(), "-100", "camera offline")
			if (err != nil) != tc.wantError {
				t.Fatalf("id=%d err=%v", id, err)
			}
			if err == nil && id != 42 {
				t.Fatal(id)
			}
		})
	}
}

func TestTelegramErrorsDoNotExposeToken(t *testing.T) {
	sender := TelegramSender{Token: "secret-test-token", Client: http.DefaultClient, baseURL: "http://127.0.0.1:1"}
	_, err := sender.Send(context.Background(), "-100", "test")
	if err == nil {
		t.Fatal("expected transport error")
	}
	if err.Error() != "Telegram: hasil pengiriman belum dapat dipastikan" {
		t.Fatal(err)
	}
}
