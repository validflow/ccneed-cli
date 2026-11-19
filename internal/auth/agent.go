package auth

import "github.com/validflow/ccneed-cli/internal/api"

// NormalizeAgent validates the display-only host supplied by a native plugin.
//
// Args:
//
//	value: Canonical host ID, common short alias, or empty for a plain terminal.
//
// Returns:
//
//	The canonical host ID, without treating arbitrary display names as trusted.
//
// Errors:
//
//	Returns invalid_input for an unsupported host.
func NormalizeAgent(value string) (string, error) {
	switch value {
	case "", "cli":
		return "cli", nil
	case "cx", "codex":
		return "codex", nil
	case "cc", "claude", "claude-code":
		return "claude", nil
	case "wb", "workbuddy":
		return "workbuddy", nil
	case "kimi", "openclaw", "pi", "zcode", "qoder", "trae", "dsh":
		return value, nil
	default:
		return "", &api.Error{Code: "invalid_input"}
	}
}
