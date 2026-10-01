// Command ccneed 连接云端 CCNeed 能力；不会在本地运行 Agent loop。
package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/validflow/ccneed-cli/internal/cli"
	"github.com/validflow/ccneed-cli/internal/update"
	"github.com/validflow/ccneed-cli/internal/version"
)

var buildVersion = "dev"
var buildCommit = "unknown"

// main 将唯一进程退出点与可测试的命令层分开。
func main() { os.Exit(run()) }

// run 建立本地取消上下文、构建身份及非敏感更新缓存。
//
// Returns:
//
//	命令层给出的进程退出码。
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	options := cli.Options{Version: version.Info{Version: buildVersion, Commit: buildCommit}, Stdout: os.Stdout, Stderr: os.Stderr}
	if directory, err := os.UserCacheDir(); err == nil {
		checker := update.New(update.NewGitHubSource(nil), update.NewFileCache(filepath.Join(directory, "ccneed", "update.json")), time.Now)
		options.CheckUpdate = checker.Check
	}
	return cli.Run(ctx, os.Args[1:], options)
}
