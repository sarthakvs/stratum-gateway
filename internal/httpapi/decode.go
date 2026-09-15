package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template/parse"

	"github.com/sarthakvs/statum-gateway/internal/provider"
	"github.com/sarthakvs/stratum-gateway/internal/provider"
)

type wireRequest struct {
	Model       string        `json:"model"`
	Messages    []wireMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature *float64      `json:"temperature"`
	Stream      bool          `json:"stream"`
}

type wireMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type APIError struct {
	Status  int
	Code    string
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}
func badRequest(format string, args ...any) *APIError {
	return &APIError{
		Status:  http.StatusBadRequest,
		Code:    "invalid_request",
		Message: fmt.Sprintf(format, args...),
	}
}
func decodeRequest(r *http.Request) (provider.Request, error) {
	var wire wireRequest
	if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
		return provider.Request{}, &APIError{
			Status:  http.StatusBadRequest,
			Code:    "invalid_json",
			Message: fmt.Sprintf("request body is not a valid JSON: %V", err),
		}
	}

	if wire.Model == "" {
		return provider.Request{}, badRequest("Model is Required")
	}
	if len(wire.Messages) == 0 {
		return provider.Request{}, badRequest("messages must contain at least one message")
	}
	if wire.MaxTokens < 0 {
		return provider.Request{}, badRequest("max_tokens must not be negative, got %d", wire.MaxTokens)
	}
	if wire.Temperature != nil && (*wire.Temperature < 0 || *wire.Temperature > 2) {
		return provider.Request{}, badRequest("Temperature must be between 0 and 2, got %v", *wire.Temperature)
	}

	msgs := make([]provider.Message, 0, len(wire.Messages))
	for i, m := range wire.Messages {
		role, err := parseRole(m.Role)
		if err != nil {
			return provider.Request{}, badRequest("messages[%d]: %v", i, err)
		}
		msgs = append(msgs, provider.Message{Role: role, Content: m.Content})
	}

	return provider.Request{
		Model:       wire.Model,
		Messages:    msgs,
		MaxTokens:   wire.MaxTokens,
		Temperature: wire.Temperature,
		Stream:      wire.Stream,
	}, nil
}

func parseRole(s string) (provider.Role, error) {
	switch provider.Role(s) {
	case provider.RoleSystem, provider.RoleUser, provider.RoleAssistant:
		return provider.Role(s), nil
	default:
		return "", fmt.Errorf("unknown role %q", s)
	}
}
