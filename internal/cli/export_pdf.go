package cli

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/artifacts"
)

// artifactExportPDFCommand exports an owned report through Club's PDF renderer.
//
// Args:
//
//	connection: Credential bound to the explicitly selected service origin.
//
// Returns:
//
//	Separate PDF export command; raw artifact download remains unchanged.
func artifactExportPDFCommand(connection *connectionConfig) *cobra.Command {
	var threadID, artifactID, output string
	command := &cobra.Command{
		Use:   "export-pdf",
		Short: "Render an owned report to PDF on the server and save it locally",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := artifacts.ValidateIDs(threadID, artifactID); err != nil {
				return err
			}
			if _, err := artifacts.ValidateOutput(output); err != nil {
				return err
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			result, err := artifacts.New(client).ExportPDF(cmd.Context(), threadID, artifactID, output)
			if err != nil {
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		},
	}
	command.Flags().StringVar(&threadID, "thread-id", "", "Root conversation UUID")
	command.Flags().StringVar(&artifactID, "artifact-id", "", "Ready report artifact UUID")
	command.Flags().StringVar(&output, "output", "", "New local PDF path; existing files are never overwritten")
	return command
}
