package script

const EventVersion = "1.0"

// 传出事件
type ScriptEvent struct {
	Version   string         `json:"version"`
	EventID   string         `json:"event_id"`
	Timestamp int64          `json:"timestamp"`
	Type      string         `json:"event_type"`
	Source    string         `json:"source"`
	Hostname  string         `json:"hostname"`
	Data      map[string]any `json:"data"`
	Context   map[string]any `json:"context,omitempty"`
}

// 传入事件
type ScriptResult struct {
	Action       string         `json:"action"`
	Level        string         `json:"level"`
	Message      string         `json:"message"`
	AllowDefault bool           `json:"allow_default"`
	Data         map[string]any `json:"data,omitempty"`
}

type ScriptExecution struct {
	Time      int64        `json:"time"`
	Script    string       `json:"script"`
	EventID   string       `json:"event_id"`
	EventType string       `json:"event_type"`
	Success   bool         `json:"success"`
	CostMS    int64        `json:"cost_ms"`
	Result    ScriptResult `json:"result,omitempty"`
	Error     string       `json:"error,omitempty"`
}
