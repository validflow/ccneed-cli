package artifacts

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/validflow/ccneed-cli/internal/api"
)

const maxPDFBytes int64 = 50 * 1024 * 1024

// ExportPDF saves Club's rendered report PDF without changing raw artifact downloads.
//
// Args:
//
//	ctx: Cancellation context for the server-side render request.
//	threadID: Existing root conversation UUID.
//	artifactID: Ready report artifact UUID.
//	output: Explicit, not-yet-existing local PDF destination.
//
// Returns:
//
//	Saved PDF metadata, or an error with no published target file.
//
// Errors:
//
//	Rejects invalid IDs, unsafe destinations, non-PDF or oversized server output.
func (c *Client) ExportPDF(ctx context.Context, threadID, artifactID, output string) (Downloaded, error) {
	if err := ValidateIDs(threadID, artifactID); err != nil {
		return Downloaded{}, err
	}
	target, err := ValidateOutput(output)
	if err != nil {
		return Downloaded{}, err
	}
	if !strings.EqualFold(filepath.Ext(target), ".pdf") {
		return Downloaded{}, &api.Error{Code: "invalid_output"}
	}
	temporary, err := os.CreateTemp(filepath.Dir(target), ".ccneed-export-pdf-*")
	if err != nil {
		return Downloaded{}, &api.Error{Code: "output_write"}
	}
	name := temporary.Name()
	defer func() { _ = temporary.Close(); _ = os.Remove(name) }()
	hash := sha256.New()
	size, err := c.api.CopyReportPDF(ctx, threadID, artifactID, io.MultiWriter(temporary, hash), maxPDFBytes)
	if err != nil {
		return Downloaded{}, err
	}
	if _, err := temporary.Seek(0, io.SeekStart); err != nil {
		return Downloaded{}, &api.Error{Code: "output_write"}
	}
	var signature [5]byte
	if _, err := io.ReadFull(temporary, signature[:]); err != nil || string(signature[:]) != "%PDF-" {
		return Downloaded{}, &api.Error{Code: "invalid_response"}
	}
	if err := finishDownload(temporary, target); err != nil {
		return Downloaded{}, err
	}
	return Downloaded{
		ThreadID: threadID, ArtifactID: artifactID, Path: target,
		Size: size, SHA256: fmt.Sprintf("%x", hash.Sum(nil)),
		ContentType: "application/pdf",
	}, nil
}
