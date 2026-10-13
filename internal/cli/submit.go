package cli

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/spf13/cobra"
	"github.com/validflow/ccneed-cli/internal/api"
	"github.com/validflow/ccneed-cli/internal/conversations"
	"github.com/validflow/ccneed-cli/internal/requests"
	"github.com/validflow/ccneed-cli/internal/resources"
	"github.com/validflow/ccneed-cli/internal/tasks"
)

// submitCommand 提交已有会话的一轮输入，立即返回回合身份而非阻塞云端执行。
//
// Args:
//
//	connection: 当前 origin 的认证连接。
//
// Returns:
//
//	异步提交命令。
func submitCommand(connection *connectionConfig) *cobra.Command {
	var threadID, requestID string
	var title string
	var selection resources.Selection
	command := &cobra.Command{Use: "run MESSAGE", Short: "Start a conversation or submit another cloud turn", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return submitTurn(cmd, connection, threadID, requestID, title, selection, args[0])
		}}
	command.Flags().StringVar(&threadID, "thread-id", "", "Existing root conversation UUID")
	command.Flags().StringVar(&requestID, "request-id", "", "Original request UUID when explicitly retrying; otherwise generated and recorded before submission")
	command.Flags().Int64SliceVar(&selection.KBIDs, "kb-id", nil, "Existing knowledge-base IDs for a new conversation")
	command.Flags().StringSliceVar(&selection.FileIDs, "file-id", nil, "Uploaded file UUIDs for a new conversation")
	command.Flags().StringVar(&title, "title", "", "Optional title for a new conversation")
	return command
}

// validateRunInput separates first-message selection from existing-thread turns.
//
// Args:
//
//	threadID: Optional existing conversation UUID.
//	requestID: Optional retry UUID for an existing conversation.
//	title: Optional new-conversation title.
//	selection: Initial resources for a new conversation.
//	message: User message for this turn.
//
// Returns:
//
// Nil when the selected mode and message are valid.
//
// Errors:
//
// Returns invalid_input for incompatible flags or malformed input.
func validateRunInput(threadID, requestID, title string, selection resources.Selection, message string) error {
	if threadID == "" {
		if requestID != "" {
			return &api.Error{Code: "invalid_input"}
		}
		if err := tasks.ValidateMessage(message); err != nil {
			return err
		}
		return conversations.ValidateCreate(selection, title)
	}
	if len(selection.KBIDs) != 0 || len(selection.FileIDs) != 0 || title != "" {
		return &api.Error{Code: "invalid_input"}
	}
	return tasks.ValidateSubmission(threadID, requestID, message)
}

// submitTurn creates a conversation only for a valid first message, then submits it.
//
// Args:
//
//	cmd: Cobra invocation and output stream.
//	connection: Authenticated service connection.
//	threadID: Existing conversation UUID, or empty for a new conversation.
//	requestID: Optional existing-turn retry identity.
//	title: Optional new-conversation title.
//	selection: Initial knowledge bases and uploaded files.
//	message: User request to run in the cloud.
//
// Returns:
//
// Nil after encoding the accepted turn acknowledgement.
//
// Errors:
//
// Returns validation, connection, creation, or submission errors without retrying writes.
func submitTurn(cmd *cobra.Command, connection *connectionConfig, threadID, requestID, title string, selection resources.Selection, message string) error {
	if err := validateRunInput(threadID, requestID, title, selection, message); err != nil {
		return err
	}
	directory, err := requests.DefaultDirectory()
	if err != nil {
		return err
	}
	client, err := connection.client(cmd.Context())
	if err != nil {
		return err
	}
	newConversation := threadID == ""
	if newConversation {
		created, createErr := conversations.New(client).Create(cmd.Context(), selection, title)
		if createErr != nil {
			return createErr
		}
		threadID = created.ThreadID
	}
	result, err := tasks.New(client).Submit(cmd.Context(), directory, threadID, requestID, message)
	if err != nil {
		if newConversation && shouldReclaimEmptyThread(err) {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = conversations.New(client).DeleteEmpty(cleanupCtx, threadID)
		}
		return err
	}
	return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
}

// shouldReclaimEmptyThread permits guarded cleanup only when the first turn
// definitely was not accepted. Ambiguous transport outcomes must be retried
// with their original thread and request IDs instead.
//
// Args:
//
//	err: Submission failure after a newly created root was returned.
//
// Returns:
//
// Whether the shell can safely be offered to the server's only_if_empty guard.
func shouldReclaimEmptyThread(err error) bool {
	var submission *tasks.SubmissionError
	if !errors.As(err, &submission) {
		return true
	}
	var remote *api.Error
	return errors.As(err, &remote) && remote.Status >= 400 && remote.Status < 500
}
