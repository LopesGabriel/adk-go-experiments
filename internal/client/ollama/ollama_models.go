package ollama

import "time"

type ChatRequest struct {
	Model    string           `json:"model"`
	Messages []RequestMessage `json:"messages"`
	Tools    []AvailableTool  `json:"tools,omitempty"`
	Format   string           `json:"format,omitempty"`
	Options  *Options         `json:"options"`
	Stream   bool             `json:"stream"`
	Think    bool             `json:"think"`
}

type RequestMessage struct {
	Role      Role       `json:"role"`
	Content   string     `json:"content"`
	Images    []string   `json:"images,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type ToolCall struct {
	Function FunctionTool `json:"function"`
}

type ToolCallResponse struct {
	Function FunctionToolResponse `json:"function"`
}

type AvailableTool struct {
	Type     ToolType     `json:"type"`
	Function FunctionTool `json:"function"`
}

type FunctionTool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type FunctionToolResponse struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ToolType string

const (
	ToolTypeFunction ToolType = "function"
)

type Options struct {
	Seed        int32   `json:"seed,omitempty"`
	Temperature float32 `json:"temperature,omitempty"`
	TopK        float32 `json:"top_k,omitempty"`
	TopP        float32 `json:"top_p,omitempty"`
	MinP        float32 `json:"min_p,omitempty"`
	Stop        string  `json:"stop,omitempty"`
	NumCtx      int     `json:"num_ctx,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ChatResponse struct {
	Model     string          `json:"model"`
	CreatedAt time.Time       `json:"created_at"`
	Message   ResponseMessage `json:"message"`
}

type ResponseMessage struct {
	Role               Role               `json:"role"`
	Content            string             `json:"content"`
	Thinking           string             `json:"thinking,omitempty"`
	ToolCalls          []ToolCallResponse `json:"tool_calls,omitempty"`
	Images             []string           `json:"images,omitempty"`
	Done               bool               `json:"done"`
	DoneReason         string             `json:"done_reason"`
	TotalDuration      int64              `json:"total_duration"`
	LoadDuration       int64              `json:"load_duration"`
	PromptEvalCount    int                `json:"prompt_eval_count"`
	PromptEvalDuration int64              `json:"prompt_eval_duration"`
	EvalCount          int64              `json:"eval_count"`
}
