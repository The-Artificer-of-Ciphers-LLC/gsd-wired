package hook

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

// HookInput represents the JSON payload Claude Code sends to a hook binary via stdin.
type HookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	HookEventName  string `json:"hook_event_name"`
}

// HookOutput represents the JSON response written to stdout by a hook binary.
// An empty HookOutput ({}) is a valid no-op response for all hooks.
type HookOutput struct {
	Continue       *bool `json:"continue,omitempty"`
	SuppressOutput bool  `json:"suppressOutput,omitempty"`
}
