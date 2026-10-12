package requests

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/validflow/ccneed-cli/internal/api"
)

var journalName = regexp.MustCompile(`^[0-9a-f]{64}\.json$`)
var requestIdentifier = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Reference 只公开原始会话和提交标识，不泄露提示词摘要或其他环境。
type Reference struct {
	ThreadID  string `json:"thread_id"`
	RequestID string `json:"client_request_id"`
}

// List 找回终端退出前已落盘的原始提交标识，不代表云端已接收。
//
// Args:
//
//	directory: 操作系统用户配置目录下的恢复目录。
//	origin: 已校验的当前规范化 origin。
//	threadID: 已校验的原始会话 UUID。
//
// Returns:
//
//	当前 origin 和会话的提交引用；没有目录时为 []。
//
// Errors:
//
//	目录、记录损坏或超过一万条扫描边界时返回 recovery_unavailable。
func List(directory, origin, threadID string) ([]Reference, error) {
	entries, err := journalEntries(directory)
	if err != nil {
		return nil, err
	}
	result := []Reference{}
	for _, entry := range entries {
		if !journalName.MatchString(entry.Name()) {
			continue
		}
		record, err := readJournal(filepath.Join(directory, entry.Name()))
		if err != nil {
			return nil, err
		}
		expected := fmt.Sprintf("%x.json", sha256.Sum256([]byte(origin+"\x00"+record.ThreadID+"\x00"+record.RequestID)))
		if record.ThreadID == threadID && entry.Name() == expected {
			result = append(result, Reference{ThreadID: record.ThreadID, RequestID: record.RequestID})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].RequestID < result[j].RequestID })
	return result, nil
}

// journalEntries 拒绝目录链接，不把恢复读取转成目录创建。
//
// Args:
//
//	directory: 当前本地恢复目录。
//
// Returns:
//
//	有限目录项，目录不存在时为空。
//
// Errors:
//
//	不安全目录或超出扫描边界时返回 recovery_unavailable。
func journalEntries(directory string) ([]os.DirEntry, error) {
	info, err := os.Lstat(directory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, &api.Error{Code: "recovery_unavailable"}
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) > 10000 {
		return nil, &api.Error{Code: "recovery_unavailable"}
	}
	return entries, nil
}

// readJournal 只读有限普通文件，并验证打开后的文件身份和不可变结构。
//
// Args:
//
//	filename: 本地恢复目录中的摘要文件。
//
// Returns:
//
//	不含提示词或凭证的原始记录。
//
// Errors:
//
//	链接、竞争替换、损坏或不合法记录返回 recovery_unavailable。
func readJournal(filename string) (Record, error) {
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 2048 {
		return Record{}, &api.Error{Code: "recovery_unavailable"}
	}
	file, err := os.Open(filename)
	if err != nil {
		return Record{}, &api.Error{Code: "recovery_unavailable"}
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return Record{}, &api.Error{Code: "recovery_unavailable"}
	}
	data, err := io.ReadAll(io.LimitReader(file, 2049))
	var record Record
	if err != nil || len(data) > 2048 || json.Unmarshal(data, &record) != nil {
		return Record{}, &api.Error{Code: "recovery_unavailable"}
	}
	if !validRecord(record) {
		return Record{}, &api.Error{Code: "recovery_unavailable"}
	}
	return record, nil
}

// validRecord 验证持久化恢复标识和摘要，不把损坏记录当成新的提交。
//
// Args:
//
//	record: 本地解码记录。
//
// Returns:
//
//	原始 UUID 和 SHA256 摘要均有效时为真。
func validRecord(record Record) bool {
	return requestIdentifier.MatchString(record.ThreadID) &&
		requestIdentifier.MatchString(record.RequestID) &&
		journalName.MatchString(record.InputDigest+".json")
}
