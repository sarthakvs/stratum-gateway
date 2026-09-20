package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ptr[T any](v T) *T {
	return &v
}
func TestToOllamaReq(t *testing.T) {
	t.Run("no options when nothing is set", func(t *testing.T) {
		got := toOllamaReq(Request{
			Model:    "llama3.2:3b",
			Messages: []Message{{Role: RoleUser, Content: "hi"}},
			Stream:   true,
		})

		if got.Options != nil {
			t.Errorf("Options = %v, want nil", got.Options)
		}
		if got.Model != "llama3.2:3b" {
			t.Errorf("Model = %q, want %q", got.Model, "llama3.2:3b")
		}
		if !got.Stream {
			t.Error("Stream = false, want true")
		}
		if len(got.Messages) != 1 {
			t.Fatalf("len(Messages) = %d, want 1", len(got.Messages))
		}
		if got.Messages[0].Role != "user" {
			t.Errorf("Role = %q,want %q", got.Messages[0].Role, "user")
		}
	})
	t.Run("max_rokens becomes num_predict", func(t *testing.T) {
		got := toOllamaReq(Request{
			Model:     "x",
			Messages:  []Message{{Role: RoleUser, Content: "hi"}},
			MaxTokens: 100,
		})
		if got.Options == nil {
			t.Fatal("Options = nil, want it set")
		}
		if got.Options.NumPredict == nil {
			t.Fatal("NumPredict = nil, want 100")
		}
		if *got.Options.NumPredict != 100 {
			t.Errorf("NumPredict = %d,want 100", *got.Options.NumPredict)
		}
		if got.Options.Temperature != nil {
			t.Errorf("Temperature %v, want nil", *got.Options.Temperature)
		}
	})
	t.Run("temperature passes through", func(t *testing.T) {
		got := toOllamaReq(Request{
			Model:       "x",
			Messages:    []Message{{Role: RoleUser, Content: "hi"}},
			Temperature: ptr(0.5),
		})
		if got.Options == nil {
			t.Fatal("Options = nil, want it set")
		}
		if got.Options.Temperature == nil {
			t.Fatal("Temperature = nil, want 0.5")
		}
		if *got.Options.Temperature != 0.5 {
			t.Errorf("Temperature = %d,want 0.5", *got.Options.NumPredict)
		}
		if got.Options.NumPredict != nil {
			t.Errorf("NumPredict = %d,want nil", *got.Options.NumPredict)
		}
	})
	t.Run("all roles stringified", func(t *testing.T) {
		got := toOllamaReq(Request{
			Model: "x",
			Messages: []Message{
				{Role: RoleSystem, Content: "be brief"},
				{Role: RoleUser, Content: "hi"},
				{Role: RoleAssistant, Content: "hello"},
			},
		})
		want := []string{"system", "user", "assistant"}
		if len(got.Messages) != len(want) {
			t.Fatalf("len(Messages) = %d, want %d", len(got.Messages), len(want))
		}
		for i, w := range want {
			if got.Messages[i].Role != w {
				t.Errorf("Messages[%d].Role = %q, want %q", i, got.Messages[i].Role, w)
			}
		}
	})
}

func TestOllamaStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %q, want /api/chat", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		fmt.Fprintln(w, `{"message":{"role":"assistant","content":"Hel"},"done":false}`)
		fmt.Fprintln(w, `{"message":{"role":"assistant","content":"lo"},"done":true,"done_reason":"stop","prompt_eval_count":5,"eval_count":2}`)
	}))
	defer srv.Close()
	o := NewOllama(srv.URL, srv.Client())
	stream, err := o.Stream(context.Background(), Request{
		Model:    "x",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
		Stream:   true,
	})
	if err != nil {
		t.Fatalf("Stream() error: %v", err)
	}
	defer stream.Close()
	c1, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() #1 error: %v", err)
	}
	if c1.Content != "Hel" {
		t.Errorf("chunk 1 Content = %q, want  %q", c1.Content, "Hel")
	}
	if c1.Usage != nil {
		t.Errorf("chunk 1 Usage = %v,want nil", c1.Usage)
	}
	c2, err := stream.Next()
	if err != nil {
		t.Fatalf("Next() #2 error: %v", err)
	}
	if c2.Content != "lo" {
		t.Errorf("chunk 2 Content = %q, want  %q", c2.Content, "lo")
	}
	if c2.FinishReason != "stop" {
		t.Errorf("FinishReason = %q,want %q", c2.FinishReason, "stop")
	}
	if c2.Usage == nil {
		t.Fatal("chunk 2 Usage = nil, want token counts")
	}
	if c2.Usage.InputTokens != 5 {
		t.Errorf("InputTokens = %d,want 5", c2.Usage.InputTokens)
	}
	if c2.Usage.OutputTokens != 2 {
		t.Errorf("OutputTokens = %d, want 2", c2.Usage.OutputTokens)
	}
	if _, err := stream.Next(); !errors.Is(err, io.EOF) {
		t.Errorf("Next() #3 error = %v, want io.EOF", err)
	}
}

func TestOllamaStream_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "model not found", http.StatusNotFound)
	}))
	defer srv.Close()
	o := NewOllama(srv.URL, srv.Client())
	_, err := o.Stream(context.Background(), Request{
		Model:    "nope",
		Messages: []Message{{Role: RoleUser, Content: "hi"}},
	})
	if err == nil {
		t.Fatal("Stream() = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %q, want it to mention the status code", err)
	}
}
