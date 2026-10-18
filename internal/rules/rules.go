// Package rules 定义 sweep 知道的全部清理规则。
//
// 规则是纯数据：路径支持 %ENV% 变量和 * 通配符，扫描和清理阶段
// 各自解释同一份定义，保证 "看到什么" 和 "删什么" 始终一致。
package rules

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Risk 表示一条规则的误删风险。
type Risk int

const (
	// Safe 删了会自动重建，几乎无风险。
	Safe Risk = iota
	// Caution 不可恢复，但通常是"垃圾"。
	Caution
	// Risky 可能删掉用户还需要的东西，需显式 --force。
	Risky
)

func (r Risk) String() string {
	switch r {
	case Safe:
		return "安全"
	case Caution:
		return "谨慎"
	case Risky:
		return "高风险"
	}
	return "未知"
}

// Color 返回风险等级对应的 ANSI 颜色代码。
func (r Risk) Color() string {
	switch r {
	case Safe:
		return "32"
	case Caution:
		return "33"
	case Risky:
		return "31"
	}
	return "0"
}

// Action 表示一条规则如何执行清理。
type Action int

const (
	// DeleteContents 删除目录下的全部内容，保留目录本身。
	DeleteContents Action = iota
	// DeleteFiles 只删除匹配通配符的文件。
	DeleteFiles
	// EmptyRecycleBin 清空回收站。
	EmptyRecycleBin
	// DisableHibernate 关闭休眠并删除 hiberfil.sys。
	DisableHibernate
)

func (a Action) String() string {
	switch a {
	case DeleteContents:
		return "清空目录内容"
	case DeleteFiles:
		return "删除匹配文件"
	case EmptyRecycleBin:
		return "清空回收站"
	case DisableHibernate:
		return "关闭休眠"
	}
	return "未知"
}

// Rule 描述一条清理规则。
type Rule struct {
	// ID 是稳定的机器标识，写脚本时用它。
	ID string
	// Name 是给人看的中文名称。
	Name string
	// Category 是分组，如 temp / cache / system。
	Category string
	Risk     Risk
	Action   Action
	// Paths 支持 %ENV% 变量和 * 通配符。
	Paths []string
	// Patterns 仅在 DeleteFiles 时生效，例如 "*.exe"。
	Patterns []string
	// MinAge 只删除修改时间早于 now-MinAge 的文件；0 表示不限制。
	MinAge time.Duration
	// NeedsAdmin 表示通常需要管理员权限才能完整处理。
	NeedsAdmin bool
	// Note 是一句人话解释，会显示在列表里。
	Note string
}

// Categories 返回全部合法类别，顺序固定。
func Categories() []string {
	return []string{"temp", "cache", "system", "installers", "dev"}
}

// All 返回内置规则表。
//
// 新增规则只需要在这里加一条，扫描、清理、文档会自动跟上。
func All() []Rule {
	return []Rule{
		{
			ID:       "temp.user",
			Name:     "用户临时文件",
			Category: "temp",
			Risk:     Safe,
			Action:   DeleteContents,
			Paths:    []string{"%TEMP%"},
			Note:     "程序运行产生的临时文件，删除后会自动重建",
		},
		{
			ID:         "temp.windows",
			Name:       "Windows 临时文件",
			Category:   "temp",
			Risk:       Safe,
			Action:     DeleteContents,
			Paths:      []string{"%WINDIR%\\Temp"},
			NeedsAdmin: true,
			Note:       "系统级临时文件，部分需要管理员权限",
		},
		{
			ID:       "cache.gpu",
			Name:     "显卡着色器缓存",
			Category: "cache",
			Risk:     Safe,
			Action:   DeleteContents,
			Paths: []string{
				"%LOCALAPPDATA%\\NVIDIA\\DXCache",
				"%LOCALAPPDATA%\\NVIDIA\\GLCache",
				"%LOCALAPPDATA%\\NVIDIA\\NV_Cache",
				"%LOCALAPPDATA%\\AMD\\DxCache",
				"%LOCALAPPDATA%\\AMD\\GLCache",
				"%LOCALAPPDATA%\\D3DSCache",
			},
			Note: "首次启动游戏会重新编译着色器，可能略有卡顿",
		},
		{
			ID:       "cache.thumbnails",
			Name:     "缩略图缓存",
			Category: "cache",
			Risk:     Safe,
			Action:   DeleteFiles,
			Paths:    []string{"%LOCALAPPDATA%\\Microsoft\\Windows\\Explorer"},
			Patterns: []string{"thumbcache_*.db"},
			Note:     "被资源管理器占用时会自动跳过",
		},
		{
			ID:       "cache.crash",
			Name:     "崩溃转储",
			Category: "cache",
			Risk:     Safe,
			Action:   DeleteContents,
			Paths:    []string{"%LOCALAPPDATA%\\CrashDumps"},
		},
		{
			ID:       "cache.browser",
			Name:     "浏览器缓存",
			Category: "cache",
			Risk:     Safe,
			Action:   DeleteContents,
			Paths: []string{
				"%LOCALAPPDATA%\\Microsoft\\Edge\\User Data\\*\\Cache",
				"%LOCALAPPDATA%\\Microsoft\\Edge\\User Data\\*\\Code Cache",
				"%LOCALAPPDATA%\\Microsoft\\Edge\\User Data\\*\\GPUCache",
				"%LOCALAPPDATA%\\Google\\Chrome\\User Data\\*\\Cache",
				"%LOCALAPPDATA%\\Google\\Chrome\\User Data\\*\\Code Cache",
				"%LOCALAPPDATA%\\Google\\Chrome\\User Data\\*\\GPUCache",
			},
			Note: "不影响书签和登录状态，网页首次加载会变慢",
		},
		{
			ID:       "system.recyclebin",
			Name:     "回收站",
			Category: "system",
			Risk:     Caution,
			Action:   EmptyRecycleBin,
			Note:     "清空后无法恢复，请先确认里面的东西都不要了",
		},
		{
			ID:         "system.updatecache",
			Name:       "Windows 更新缓存",
			Category:   "system",
			Risk:       Caution,
			Action:     DeleteContents,
			Paths:      []string{"%WINDIR%\\SoftwareDistribution\\Download"},
			NeedsAdmin: true,
		},
		{
			ID:         "system.hibernate",
			Name:       "休眠文件",
			Category:   "system",
			Risk:       Risky,
			Action:     DisableHibernate,
			NeedsAdmin: true,
			Note:       "会失去休眠与快速启动，可用 powercfg /h on 恢复",
		},
		{
			ID:       "installers.downloads",
			Name:     "下载目录旧安装包",
			Category: "installers",
			Risk:     Risky,
			Action:   DeleteFiles,
			Paths:    []string{"%USERPROFILE%\\Downloads"},
			Patterns: []string{"*.exe", "*.msi", "*.msix", "*.zip", "*.iso"},
			MinAge:   90 * 24 * time.Hour,
			Note:     "只处理 90 天前的安装包，避免误删刚下载的文件",
		},
		{
			ID:       "dev.npm",
			Name:     "npm 缓存",
			Category: "dev",
			Risk:     Safe,
			Action:   DeleteContents,
			Paths:    []string{"%LOCALAPPDATA%\\npm-cache", "%APPDATA%\\npm-cache"},
		},
		{
			ID:       "dev.pnpm",
			Name:     "pnpm 缓存",
			Category: "dev",
			Risk:     Safe,
			Action:   DeleteContents,
			Paths:    []string{"%LOCALAPPDATA%\\pnpm\\store", "%LOCALAPPDATA%\\pnpm-cache"},
		},
		{
			ID:       "dev.pip",
			Name:     "pip 缓存",
			Category: "dev",
			Risk:     Safe,
			Action:   DeleteContents,
			Paths:    []string{"%LOCALAPPDATA%\\pip\\Cache"},
		},
	}
}

// ByID 按 ID 查找规则。
func ByID(id string) (Rule, bool) {
	for _, r := range All() {
		if strings.EqualFold(r.ID, id) {
			return r, true
		}
	}
	return Rule{}, false
}

// ExpandEnv 展开 %VAR% 形式的 Windows 环境变量（也兼容 ${VAR}）。
func ExpandEnv(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '%' {
			if j := strings.IndexByte(s[i+1:], '%'); j > 0 {
				if v, ok := os.LookupEnv(s[i+1 : i+1+j]); ok {
					b.WriteString(v)
					i += j + 2
					continue
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return os.ExpandEnv(b.String())
}

// ExpandPaths 展开变量和通配符，去重后返回真实存在的路径。
func ExpandPaths(paths []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, raw := range paths {
		for _, p := range expandOne(raw) {
			if p == "" {
				continue
			}
			key := strings.ToLower(p)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, p)
		}
	}
	return out
}

func expandOne(raw string) []string {
	p := ExpandEnv(raw)
	if p == "" {
		return nil
	}
	if !strings.ContainsAny(p, "*?[") {
		if _, err := os.Lstat(p); err != nil {
			return nil
		}
		return []string{p}
	}
	matches, err := filepath.Glob(p)
	if err != nil {
		return nil
	}
	var out []string
	for _, m := range matches {
		out = append(out, m)
	}
	return out
}

// MatchesAny 判断文件名是否匹配任一通配符（Windows 下不区分大小写）。
func MatchesAny(patterns []string, name string) bool {
	if len(patterns) == 0 {
		return true
	}
	for _, p := range patterns {
		if ok, err := filepath.Match(strings.ToLower(p), strings.ToLower(name)); err == nil && ok {
			return true
		}
	}
	return false
}

// OldEnough 判断文件是否足够旧（MinAge 为 0 时恒为真）。
func OldEnough(mod time.Time, minAge time.Duration) bool {
	if minAge <= 0 {
		return true
	}
	return time.Since(mod) >= minAge
}
