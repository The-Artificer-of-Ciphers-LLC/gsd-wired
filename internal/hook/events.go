package hook

import "encoding/json"

// Hook event name constants.
const (
	EventSessionStart = "SessionStart"
	EventPreToolUse   = "PreToolUse"
	EventPostToolUse  = "PostToolUse"
	EventPreCompact   = "PreCompact"
)

// ValidEvents lists all hook event names recognized by gsdw.
var ValidEvents = []string{EventSessionStart, EventPreToolUse, EventPostToolUse, EventPreCompact}

// IsValidEvent returns true if name is a recognized hook event.
func IsValidEvent(name string) bool {
	for _, v := range ValidEvents {
		if v == name {
			return true
		}
	}
	return false
}

// HookInputBase contains fields common to all hook events.
type HookInputBase struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
}

// SessionStartInput is the JSON payload for SessionStart events.
type SessionStartInput struct {
	HookInputBase
	Source    string `json:"source"`              // "startup"|"resume"|"clear"|"compact"
	Model     string `json:"model,omitempty"`
	AgentType string `json:"agent_type,omitempty"`
}

// PreCompactInput is the JSON payload for PreCompact events.
type PreCompactInput struct {
	HookInputBase
	Trigger            string `json:"trigger"`             // "manual"|"auto"
	CustomInstructions string `json:"custom_instructions"`
}

// PreToolUseInput is the JSON payload for PreToolUse events.
type PreToolUseInput struct {
	HookInputBase
	PermissionMode string          `json:"permission_mode"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	ToolUseID      string          `json:"tool_use_id"`
}

// PostToolUseInput is the JSON payload for PostToolUse events.
type PostToolUseInput struct {
	HookInputBase
	PermissionMode string          `json:"permission_mode"`
	ToolName       string          `json:"tool_name"`
	ToolInput      json.RawMessage `json:"tool_input"`
	ToolResponse   json.RawMessage `json:"tool_response"`
	ToolUseID      string          `json:"tool_use_id"`
}

// HookOutput is the JSON response written to stdout by any hook handler.
// An empty HookOutput ({}) is a valid no-op response for all hooks.
type HookOutput struct {
	Continue           *bool  `json:"continue,omitempty"`
	StopReason         string `json:"stopReason,omitempty"`
	SuppressOutput     bool   `json:"suppressOutput,omitempty"`
	AdditionalContext  string `json:"additionalContext,omitempty"`
	HookSpecificOutput any    `json:"hookSpecificOutput,omitempty"`
}

// PreToolUseHookOutput is the hookSpecificOutput for PreToolUse.
type PreToolUseHookOutput struct {
	HookEventName            string          `json:"hookEventName"`
	PermissionDecision       string          `json:"permissionDecision,omitempty"`
	PermissionDecisionReason string          `json:"permissionDecisionReason,omitempty"`
	UpdatedInput             json.RawMessage `json:"updatedInput,omitempty"`
	AdditionalContext        string          `json:"additionalContext,omitempty"`
}

// PostToolUseHookOutput is the hookSpecificOutput for PostToolUse.
type PostToolUseHookOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}
