package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

// Dispatch reads a hook event from stdin, validates it, and routes to the
// appropriate handler. Each handler writes JSON to stdout.
// It uses explicit context, reader/writer for testability — it does NOT
// use os.Stdin or os.Stdout directly.
func Dispatch(ctx context.Context, event string, stdin io.Reader, stdout io.Writer) error {
	// Validate the event name against known constants
	if !IsValidEvent(event) {
		return fmt.Errorf("unknown hook event: %s", event)
	}

	// Read all stdin bytes — handlers decode their own per-event type
	rawBytes, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("failed to read hook input: %w", err)
	}

	// Extract common base fields for validation
	var base struct {
		HookEventName string `json:"hook_event_name"`
		SessionID     string `json:"session_id"`
	}
	if err := json.Unmarshal(rawBytes, &base); err != nil {
		return fmt.Errorf("failed to decode hook input: %w", err)
	}

	// Validate that the JSON event name matches the subcommand argument
	if base.HookEventName != event {
		return fmt.Errorf("hook event mismatch: expected %s, got %s", event, base.HookEventName)
	}

	slog.Debug("hook dispatched", "event", event, "session_id", base.SessionID)

	// Create a fresh hookState per invocation — hooks are short-lived processes
	hs := &hookState{}

	raw := json.RawMessage(rawBytes)

	switch event {
	case EventSessionStart:
		return handleSessionStart(ctx, raw, hs, stdout)
	case EventPreCompact:
		return handlePreCompact(ctx, raw, hs, stdout)
	case EventPreToolUse:
		return handlePreToolUse(ctx, raw, hs, stdout)
	case EventPostToolUse:
		return handlePostToolUse(ctx, raw, hs, stdout)
	default:
		// Should not reach here — already validated above
		return fmt.Errorf("unhandled hook event: %s", event)
	}
}
