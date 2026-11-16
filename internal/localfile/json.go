package localfile

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/validflow/ccneed-cli/internal/api"
)

// SaveJSON publishes a complete bounded JSON result without replacing an existing file.
//
// Args:
//
//	value: An already filtered public command result.
//	output: Explicit destination path that does not exist.
//
// Returns:
//
//	The encoding or publication error, or nil.
//
// Errors:
//
//	Returns invalid_response above 4 MiB and safe output errors for invalid paths or writes.
func SaveJSON(value any, output string) error {
	target, err := ValidateOutput(output)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil || len(encoded)+1 > 4*1024*1024 {
		return &api.Error{Code: "invalid_response"}
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".ccneed-snapshot-*")
	if err != nil {
		return &api.Error{Code: "output_write"}
	}
	defer func() { _ = temporary.Close(); _ = os.Remove(temporary.Name()) }()
	if _, err := temporary.Write(append(encoded, '\n')); err != nil {
		return &api.Error{Code: "output_write"}
	}
	return Publish(temporary, target)
}
