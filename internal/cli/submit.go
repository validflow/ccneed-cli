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
			newConversation := threadID == ""
			if newConversation {
				if requestID != "" {
					return &api.Error{Code: "invalid_input"}
				}
				if err := tasks.ValidateMessage(args[0]); err != nil {
					return err
				}
				if err := conversations.ValidateCreate(selection, title); err != nil {
					return err
				}
			} else {
				if len(selection.KBIDs) != 0 || len(selection.FileIDs) != 0 || title != "" {
					return &api.Error{Code: "invalid_input"}
				}
				if err := tasks.ValidateSubmission(threadID, requestID, args[0]); err != nil {
					return err
				}
			}
			directory, err := requests.DefaultDirectory()
			if err != nil {
				return err
			}
			client, err := connection.client(cmd.Context())
			if err != nil {
				return err
			}
			if newConversation {
				created, err := conversations.New(client).Create(cmd.Context(), selection, title)
				if err != nil {
					return err
				}
				threadID = created.ThreadID
			}
			result, err := tasks.New(client).Submit(cmd.Context(), directory, threadID, requestID, args[0])
			if err != nil {
				if newConversation && shouldReclaimEmptyThread(err) {
					cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_ = conversations.New(client).DeleteEmpty(cleanupCtx, threadID)
				}
				return err
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
		}}
	command.Flags().StringVar(&threadID, "thread-id", "", "Existing root conversation UUID")
	command.Flags().StringVar(&requestID, "request-id", "", "Original request UUID when explicitly retrying; otherwise generated and recorded before submission")
	command.Flags().Int64SliceVar(&selection.KBIDs, "kb-id", nil, "Existing knowledge-base IDs for a new conversation")
	command.Flags().StringSliceVar(&selection.FileIDs, "file-id", nil, "Uploaded file UUIDs for a new conversation")
	command.Flags().StringVar(&title, "title", "", "Optional title for a new conversation")
	return command
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
