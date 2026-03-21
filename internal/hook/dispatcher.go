package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
)

// Dispatch reads a hook event from stdin, validates it, and writes a no-op JSON
// response to stdout. It uses explicit reader/writer for testability — it does NOT
// use os.Stdin or os.Stdout directly.
func Dispatch(event string, stdin io.Reader, stdout io.Writer) error {
	// Validate the event name against known constants
	if !IsValidEvent(event) {
		return fmt.Errorf("unknown hook event: %s", event)
	}

	// Decode JSON from stdin
	var input HookInput
	if err := json.NewDecoder(stdin).Decode(&input); err != nil {
		return fmt.Errorf("failed to decode hook input: %w", err)
	}

	// Validate that the JSON event name matches the subcommand argument
	if input.HookEventName != event {
		return fmt.Errorf("hook event mismatch: expected %s, got %s", event, input.HookEventName)
	}

	slog.Debug("hook dispatched", "event", event, "session_id", input.SessionID)

	// Encode no-op response to stdout — empty HookOutput is valid for all hooks
	if err := json.NewEncoder(stdout).Encode(HookOutput{}); err != nil {
		return fmt.Errorf("failed to encode hook output: %w", err)
	}

	return nil
}
