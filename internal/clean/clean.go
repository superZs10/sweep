// Package clean 执行真正的删除。
//
// 设计原则：
//  1. 单条规则失败不影响其他规则；
//  2. 被占用的文件跳过而不是报错中断；
//  3. 权限不足单独计数，最后统一提示"请用管理员运行"。
package clean

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/superZs10/sweep/internal/rules"
	"github.com/superZs10/sweep/internal/scan"
)

// Options 控制清理行为。
type Options struct {
	// DryRun 只统计不删除。
	DryRun bool
}

// Result 是一条规则的清理结果。
type Result struct {
	RuleID  string
	Freed   int64
	Deleted int
	Skipped int // 被占用或无权限而被跳过
	Denied  int // 明确因权限被拒绝
	// Debug 记录每个失败项的原始错误，仅在 --debug 时展示。
	Debug []string
	Err   error
}

// Apply 清理一条规则。
func Apply(item scan.Item, opt Options) Result {
	res := Result{RuleID: item.Rule.ID}

	if opt.DryRun {
		res.Freed = item.Bytes
		res.Deleted = item.Files
		res.Denied = item.Denied
		return res
	}

	switch item.Rule.Action {
	case rules.EmptyRecycleBin:
		if err := emptyRecycleBin(); err != nil {
			res.Err = err
			return res
		}
		res.Freed = item.Bytes
		res.Deleted = item.Files
		return res

	case rules.DisableHibernate:
		if runtime.GOOS != "windows" {
			res.Err = errUnsupported
			return res
		}
		if out, err := exec.Command("powercfg", "/h", "off").CombinedOutput(); err != nil {
			res.Err = &cmdError{cmd: "powercfg /h off", out: string(out), err: err}
			return res
		}
		res.Freed = item.Bytes
		return res

	case rules.DeleteFiles:
		cleanFiles(&res, item.Rule)

	case rules.DeleteContents:
		cleanContents(&res, item.Rule)
	}
	return res
}

func cleanContents(res *Result, r rules.Rule) {
	for _, root := range rules.ExpandPaths(r.Paths) {
		entries, err := os.ReadDir(root)
		if err != nil {
			res.Skipped++
			res.Debug = append(res.Debug, fmt.Sprintf("%s: %v", root, err))
			if scan.IsPermission(err) {
				res.Denied++
			}
			continue
		}
		for _, e := range entries {
			p := filepath.Join(root, e.Name())
			bytes, files := scan.Size(p)
			if err := os.RemoveAll(p); err != nil {
				res.Skipped++
				res.Debug = append(res.Debug, fmt.Sprintf("%s: %v", p, err))
				if scan.IsPermission(err) {
					res.Denied++
				}
				continue
			}
			res.Freed += bytes
			res.Deleted += files
		}
	}
}

func cleanFiles(res *Result, r rules.Rule) {
	for _, root := range rules.ExpandPaths(r.Paths) {
		info, err := os.Lstat(root)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			if rules.MatchesAny(r.Patterns, info.Name()) && rules.OldEnough(info.ModTime(), r.MinAge) {
				removeOne(res, root, info.Size(), 1)
			}
			continue
		}

		_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
			if err != nil {
				res.Skipped++
				if d != nil && d.IsDir() {
					if scan.IsPermission(err) {
						res.Denied++
					}
					return filepath.SkipDir
				}
				return nil
			}
			if p == root || d.IsDir() {
				return nil
			}
			if !rules.MatchesAny(r.Patterns, d.Name()) {
				return nil
			}
			fi, e := d.Info()
			if e != nil {
				return nil
			}
			if !rules.OldEnough(fi.ModTime(), r.MinAge) {
				return nil
			}
			removeOne(res, p, fi.Size(), 1)
			return nil
		})
	}
}

func removeOne(res *Result, p string, bytes int64, files int) {
	if err := os.Remove(p); err != nil {
		res.Skipped++
		res.Debug = append(res.Debug, fmt.Sprintf("%s: %v", p, err))
		if scan.IsPermission(err) {
			res.Denied++
		}
		return
	}
	res.Freed += bytes
	res.Deleted += files
}
