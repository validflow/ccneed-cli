package update

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

// FileCache 将公开版本元数据保存在指定用户缓存目录，不存储凭证。
type FileCache struct{ path string }

// NewFileCache 使用调用方确定的缓存文件路径。
//
// Args:
//
//	path: 用户缓存目录内的绝对文件路径，不得来自服务器响应。
//
// Returns:
//
//	不含认证信息的文件缓存。
func NewFileCache(path string) *FileCache { return &FileCache{path: path} }

// Load 读取上次检查，损坏或缺失由检查器作为缓存未命中处理。
//
// Returns:
//
//	上次结果，或缓存错误。
//
// Errors:
//
//	返回文件读取或 JSON 格式错误。
func (c *FileCache) Load() (Record, error) {
	file, err := os.Open(c.path)
	if err != nil {
		return Record{}, err
	}
	defer func() { _ = file.Close() }()
	var record Record
	err = json.NewDecoder(io.LimitReader(file, 2048)).Decode(&record)
	return record, err
}

// Save 以临时文件加 rename 更新非敏感元数据。
//
// Args:
//
//	record: 上次检查的公开版本信息。
//
// Returns:
//
//	保存失败，或 nil。
//
// Errors:
//
//	返回缓存目录、编码或原子替换错误；不影响产品命令。
func (c *FileCache) Save(record Record) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	directory := filepath.Dir(c.path)
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".update-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(file.Name()) }()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), c.path)
}
