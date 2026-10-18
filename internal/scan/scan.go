// Package scan 负责"只读地"测量每条规则能清出多少空间。
//
// 扫描阶段绝不删除任何东西，清理阶段复用它算出的结果。
package scan

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/superZs10/sweep/internal/rules"
)

// Root 是一个被展开后的具体路径及其占用。
type Root struct {
	Path  string
	Bytes int64
	Files int
}

// Item 是一条规则的扫描结果。
type Item struct {
	Rule   rules.Rule
	Roots  []Root
	Bytes  int64
	Files  int
	Denied int // 遇到权限拒绝的次数
}

// Empty 表示这条规则在当前机器上没有任何可清理内容。
func (i Item) Empty() bool { return i.Bytes == 0 && i.Files == 0 }

// Run 扫描一条规则；返回的 error 只在规则本身无法解释时出现。
func Run(r rules.Rule) (Item, error) {
	item := Item{Rule: r}

	switch r.Action {
	case rules.EmptyRecycleBin:
		b, n := RecycleBinStats()
		item.Bytes = b
		item.Files = int(n)
		if b > 0 || n > 0 {
			item.Roots = append(item.Roots, Root{Path: "回收站", Bytes: b, Files: int(n)})
		}
		return item, nil

	case rules.DisableHibernate:
		b, ok := measureHibernate()
		if ok {
			item.Bytes = b
			item.Roots = append(item.Roots, Root{Path: "hiberfil.sys", Bytes: b})
		}
		return item, nil
	}

	for _, p := range rules.ExpandPaths(r.Paths) {
		info, err := os.Lstat(p)
		if err != nil {
			if IsPermission(err) {
				item.Denied++
			}
			continue
		}

		if !info.IsDir() {
			if r.Action == rules.DeleteFiles &&
				rules.MatchesAny(r.Patterns, info.Name()) &&
				rules.OldEnough(info.ModTime(), r.MinAge) {
				item.Roots = append(item.Roots, Root{Path: p, Bytes: info.Size(), Files: 1})
				item.Bytes += info.Size()
				item.Files++
			}
			continue
		}

		b, f, d := walk(p, r)
		if b == 0 && f == 0 {
			item.Denied += d
			continue
		}
		item.Roots = append(item.Roots, Root{Path: p, Bytes: b, Files: f})
		item.Bytes += b
		item.Files += f
		item.Denied += d
	}
	return item, nil
}

// walk 统计目录下符合规则的文件总大小与数量。
func walk(root string, r rules.Rule) (bytes int64, files int, denied int) {
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if IsPermission(err) {
				denied++
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if p == root || d.IsDir() {
			return nil
		}
		if r.Action == rules.DeleteFiles && !rules.MatchesAny(r.Patterns, d.Name()) {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		if r.Action == rules.DeleteFiles && !rules.OldEnough(info.ModTime(), r.MinAge) {
			return nil
		}
		bytes += info.Size()
		files++
		return nil
	})
	if err != nil {
		if IsPermission(err) {
			denied++
		}
		return bytes, files, denied
	}
	return bytes, files, denied
}

// Size 计算一个路径（文件或目录）的总大小和文件数。
func Size(p string) (int64, int) {
	info, err := os.Lstat(p)
	if err != nil {
		return 0, 0
	}
	if !info.IsDir() {
		return info.Size(), 1
	}
	var total int64
	var files int
	_ = filepath.WalkDir(p, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if fi, e := d.Info(); e == nil {
			total += fi.Size()
			files++
		}
		return nil
	})
	return total, files
}

// IsPermission 判断错误是否是权限拒绝。
//
// 单独抽出来是为了让 Windows 上的"需要管理员"提示更准确：
// 除了 fs.ErrPermission，还要兼容 ERROR_ACCESS_DENIED 的错误文本。
func IsPermission(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, fs.ErrPermission) || os.IsPermission(err) {
		return true
	}
	msg := err.Error()
	for _, s := range []string{"拒绝访问", "Access is denied", "access denied"} {
		if contains(msg, s) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// measureHibernate 统计 hiberfil.sys 的大小。
func measureHibernate() (int64, bool) {
	if runtime.GOOS != "windows" {
		return 0, false
	}
	info, err := os.Stat(`C:\hiberfil.sys`)
	if err != nil {
		return 0, false
	}
	return info.Size(), true
}
