package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Ollama struct {
	baseURL string
	client  *http.Client
}

type ollamaReq struct {
	Model    string         `json:"model"`
	Messages []ollamaMsg    `json:"messages"`
	Stream   bool           `json:"stream"`
	Options  *ollamaOptions `json:"options,omitempty"`
}

type ollamaMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	NumPredict  *int     `json:"num_predict,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
}

type ollamaChunk struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done            bool   `json:"done"`
	DoneReason      string `json:"done_reason"`
	PromptEvalCount int    `json:"prompt_eval_count"`
	EvalCount       int    `json:"eval_count"`
}

type ollamaStream struct {
	resp    *http.Response
	scanner *bufio.Scanner
	done    bool
}

func NewOllama(baseURL string, c *http.Client) *Ollama {
	if c == nil {
		c = http.DefaultClient
	}
	return &Ollama{baseURL: baseURL, client: c}
}
func (o *Ollama) Name() string {
	return "ollama"
}
func toOllamaReq(req Request) ollamaReq {
	out := ollamaReq{
		Model:    req.Model,
		Stream:   req.Stream,
		Messages: make([]ollamaMsg, 0, len(req.Messages)),
	}
	for _, m := range req.Messages {
		out.Messages = append(out.Messages, ollamaMsg{
			Role:    string(m.Role),
			Content: m.Content,
		})
	}
	var opts ollamaOptions
	hasOpts := false
	if req.MaxTokens > 0 {
		n := req.MaxTokens
		opts.NumPredict = &n
		hasOpts = true
	}
	if req.Temperature != nil {
		opts.Temperature = req.Temperature
		hasOpts = true
	}
	if hasOpts {
		out.Options = &opts
	}
	return out
}
func (o *Ollama) Stream(ctx context.Context, req Request) (ChunkStream, error) {
	body, err := json.Marshal(toOllamaReq(req))
	if err != nil {
		return nil, fmt.Errorf("ollama: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ollama: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama: request failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return nil, fmt.Errorf("ollama: upstream status %d: %s", resp.StatusCode, msg)
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &ollamaStream{resp: resp, scanner: sc}, nil
}

func (s *ollamaStream) Next() (Chunk, error) {
	if s.done {
		return Chunk{}, io.EOF
	}
	for s.scanner.Scan() {
		line := bytes.TrimSpace(s.scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var oc ollamaChunk
		if err := json.Unmarshal(line, &oc); err != nil {
			return Chunk{}, fmt.Errorf("ollama: bad chunk: %w", err)
		}
		c := Chunk{Content: oc.Message.Content}
		if oc.Done {
			s.done = true
			c.FinishReason = oc.DoneReason
			c.Usage = &Usage{
				InputTokens:  oc.PromptEvalCount,
				OutputTokens: oc.EvalCount,
			}
		}
		return c, nil
	}
	if err := s.scanner.Err(); err != nil {
		return Chunk{}, fmt.Errorf("ollama: read stream %w", err)
	}
	return Chunk{}, io.EOF
}

func (s *ollamaStream) Close() error {
	return s.resp.Body.Close()
}
