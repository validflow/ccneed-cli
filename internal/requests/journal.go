// Package requests 保存不含提示词或凭证的提交恢复标识，不建立独立任务身份。
package requests

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/validflow/ccneed-cli/internal/api"
)

// Record 是按 origin 和原始会话隔离的不可变提交记录。
type Record struct {
	ThreadID    string `json:"thread_id"`
	RequestID   string `json:"client_request_id"`
	InputDigest string `json:"input_digest"`
}

// DefaultDirectory 返回操作系统用户配置目录中的非秘密恢复目录。
//
// Returns:
//
//	恢复目录，或无法找到配置根目录的错误。
//
// Errors:
//
//	返回 recovery_unavailable，不退回当前目录或共享临时目录。
func DefaultDirectory() (string, error) {
	directory, err := os.UserConfigDir()
	if err != nil || directory == "" {
		return "", &api.Error{Code: "recovery_unavailable"}
	}
	return filepath.Join(directory, "ccneed", "requests"), nil
}

// NewID 生成可在后续 CLI 调用复用的 UUID 幂等标识。
//
// Returns:
//
//	随机 UUID，或随机源错误。
//
// Errors:
//
//	随机源不可用时返回 recovery_unavailable，不生成弱随机标识。
func NewID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", &api.Error{Code: "recovery_unavailable"}
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}

// Prepare 在 HTTP 写入之前持久化标识；同标识只允许完全相同的输入摘要。
//
// Args:
//
//	directory: 明确的本地恢复根目录。
//	origin: 已规范化的 API origin，不含凭证。
//	threadID: 已校验的原始会话 UUID。
//	requestID: 已校验的幂等 UUID。
//	message: 只用于计算摘要，不写入磁盘或错误。
//
// Returns:
//
//	已同步落盘的恢复记录，或失败关闭的错误。
//
// Errors:
//
//	存储不可用时不提交；同标识不同输入返回 request_id_conflict。
func Prepare(directory, origin, threadID, requestID, message string) (Record, error) {
	record := Record{ThreadID: threadID, RequestID: requestID, InputDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(message)))}
	name := fmt.Sprintf("%x.json", sha256.Sum256([]byte(origin+"\x00"+threadID+"\x00"+requestID)))
	if err := ensureDirectory(directory); err != nil {
		return Record{}, err
	}
	filename := filepath.Join(directory, name)
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		return matchRecord(filename, record)
	}
	if err != nil {
		return Record{}, &api.Error{Code: "recovery_unavailable"}
	}
	if err := persist(file, record); err != nil {
		return Record{}, err
	}
	return record, nil
}

// ensureDirectory 拒绝恢复目录本身是链接或普通文件，避免静默写入替代目录。
//
// Args:
//
//	directory: 用户配置目录下的恢复根目录。
//
// Returns:
//
//	安全普通目录检查结果。
//
// Errors:
//
//	创建或元数据检查失败返回 recovery_unavailable。
func ensureDirectory(directory string) error {
	if directory == "" || os.MkdirAll(directory, 0700) != nil {
		return &api.Error{Code: "recovery_unavailable"}
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return &api.Error{Code: "recovery_unavailable"}
	}
	return nil
}

// persist 同步完整记录后关闭文件，未完成的记录永不触发网络提交。
//
// Args:
//
//	file: 独占创建的普通文件。
//	record: 无秘密的不可变请求记录。
//
// Returns:
//
//	同步与关闭结果。
//
// Errors:
//
//	任一步骤失败返回 recovery_unavailable；残留记录保持失败关闭。
func persist(file *os.File, record Record) error {
	defer func() { _ = file.Close() }()
	if json.NewEncoder(file).Encode(record) != nil || file.Sync() != nil {
		return &api.Error{Code: "recovery_unavailable"}
	}
	if err := file.Close(); err != nil {
		return &api.Error{Code: "recovery_unavailable"}
	}
	return nil
}

// matchRecord 检查有限大小普通文件，只接受原样不可变记录。
//
// Args:
//
//	filename: 当前 origin/thread/request 的摘要文件路径。
//	expected: 当前输入的预期记录。
//
// Returns:
//
//	已存在的相同记录，或冲突错误。
//
// Errors:
//
//	链接、损坏或不可读记录失败关闭，不覆盖恢复证据。
func matchRecord(filename string, expected Record) (Record, error) {
	info, err := os.Lstat(filename)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 2048 {
		return Record{}, &api.Error{Code: "recovery_unavailable"}
	}
	file, err := os.Open(filename)
	if err != nil {
		return Record{}, &api.Error{Code: "recovery_unavailable"}
	}
	defer func() { _ = file.Close() }()
	data, err := io.ReadAll(io.LimitReader(file, 2049))
	var stored Record
	if err != nil || len(data) > 2048 || json.Unmarshal(data, &stored) != nil {
		return Record{}, &api.Error{Code: "recovery_unavailable"}
	}
	if stored != expected {
		return Record{}, &api.Error{Code: "request_id_conflict"}
	}
	return stored, nil
}
