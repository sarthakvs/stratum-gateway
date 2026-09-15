package provider

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role
	Content string
}

type Request struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature *float64
	Stream      bool
}

type Chunk struct {
	Content      string
	FinishReason string
	Usage        *Usage
}

type Usage struct {
	InputTokens  int
	OutputTokens int
}
