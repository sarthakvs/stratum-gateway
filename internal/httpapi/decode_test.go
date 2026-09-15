package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sarthakvs/stratum-gateway/internal/provider"
)

func TestDecodeRequest(t *testing.T) {

	tests := []struct {
		name     string
		body     string
		wantErr  bool
		wantCode string
	}{
		{"valid", `{"model":"llama3.2:3b","messages":[{"role":"user","content":"hi"}]}`, false, ""},
		{"empty body", ``, true, "invalid_json"},
		{"malformed json", `{"model":`, true, "invalid_json"},
		{"missing model", `{"messages":[{"role":"user","content":"hi"}]}`, true, "invalid_request"},
		{"no messages", `{"model":"x","messages":[]}`, true, "invalid_request"},
		{"negative max_tokens", `{"model":"x","messages":[{"role":"user","content":"hi"}],"max_tokens":-1}`, true, "invalid_request"},
		{"bad role", `{"model":"x","messages":[{"role":"banana","content":"hi"}]}`, true, "invalid_request"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/chat/completions", strings.NewReader(tt.body))
			got, err := decodeRequest(req)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("decodeRequest() returned unexpected error: %v ", err)
				}
				if got.Model != "llama3.2:3b" {
					t.Errorf("Model = %q, want %q", got.Model, "llama3.2:3b")
				}
				if len(got.Messages) != 1 {
					t.Fatalf("len(Messages) = %d, want 1", len(got.Messages))
				}
				if got.Messages[0].Role != provider.RoleUser {
					t.Errorf("Role = %q, want %q", got.Messages[0].Role, provider.RoleUser)
				}
				if got.Messages[0].Content != "hi" {
					t.Errorf("Content = %q, want %q", got.Messages[0].Content, "hi")
				}
				return
			}

			if err == nil {
				t.Fatalf("decodeRequest() = nil error, want an error")
			}

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error if %T, want *APIError", err)
			}
			if apiErr.Code != tt.wantCode {
				t.Errorf("Code = %q, want %q", apiErr.Code, tt.wantCode)
			}
			if apiErr.Status != http.StatusBadRequest {
				t.Errorf("Status = %d, want %d", apiErr.Status, http.StatusBadRequest)
			}
		})
	}

}
