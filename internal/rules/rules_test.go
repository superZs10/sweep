package rules

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("SWEEP_TEST_DIR", `C:\fake\dir`)
	if got := ExpandEnv(`%SWEEP_TEST_DIR%\sub`); got != `C:\fake\dir\sub` {
		t.Errorf("ExpandEnv = %q", got)
	}
	// 未知变量原样保留
	if got := ExpandEnv(`%NOT_SET_AT_ALL%\x`); got != `%NOT_SET_AT_ALL%\x` {
		t.Errorf("ExpandEnv unknown = %q", got)
	}
}

func TestExpandPaths(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	f2 := filepath.Join(dir, "b.txt")
	for _, f := range []string{f1, f2} {
		if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	got := ExpandPaths([]string{filepath.Join(dir, "*.txt")})
	if len(got) != 2 {
		t.Fatalf("glob matched %d paths, want 2: %v", len(got), got)
	}

	// 不存在的路径被过滤
	if got := ExpandPaths([]string{filepath.Join(dir, "nope")}); len(got) != 0 {
		t.Errorf("expected no match, got %v", got)
	}

	// 重复路径去重
	dup := ExpandPaths([]string{f1, f1})
	if len(dup) != 1 {
		t.Errorf("expected dedup to 1, got %d", len(dup))
	}
}

func TestMatchesAny(t *testing.T) {
	cases := []struct {
		patterns []string
		name     string
		want     bool
	}{
		{[]string{"*.exe"}, "setup.EXE", true},
		{[]string{"*.exe"}, "a.zip", false},
		{[]string{"thumbcache_*.db"}, "thumbcache_96.db", true},
		{nil, "anything", true},
	}
	for _, c := range cases {
		if got := MatchesAny(c.patterns, c.name); got != c.want {
			t.Errorf("MatchesAny(%v, %q) = %v, want %v", c.patterns, c.name, got, c.want)
		}
	}
}

func TestOldEnough(t *testing.T) {
	now := time.Now()
	if !OldEnough(now.Add(-100*24*time.Hour), 90*24*time.Hour) {
		t.Error("100 天前的文件应算旧文件")
	}
	if OldEnough(now.Add(-10*24*time.Hour), 90*24*time.Hour) {
		t.Error("10 天前的文件不应算旧文件")
	}
	if !OldEnough(now, 0) {
		t.Error("MinAge=0 时应恒为真")
	}
}

// TestRulesValid 检查内置规则表本身没有笔误。
func TestRulesValid(t *testing.T) {
	seen := map[string]bool{}
	cats := map[string]bool{}
	for _, c := range Categories() {
		cats[c] = true
	}
	for _, r := range All() {
		if r.ID == "" || r.Name == "" {
			t.Errorf("规则缺少 ID 或名称: %+v", r)
		}
		if seen[r.ID] {
			t.Errorf("重复的规则 ID: %s", r.ID)
		}
		seen[r.ID] = true
		if !cats[r.Category] {
			t.Errorf("规则 %s 使用了未声明的类别 %q", r.ID, r.Category)
		}
		if r.Action == DeleteContents || r.Action == DeleteFiles {
			if len(r.Paths) == 0 {
				t.Errorf("规则 %s 没有路径", r.ID)
			}
		}
		if r.Action == DeleteFiles && len(r.Patterns) == 0 {
			t.Errorf("规则 %s 是 DeleteFiles 但没有 Pattern", r.ID)
		}
	}
}

func TestByID(t *testing.T) {
	if r, ok := ByID("temp.user"); !ok || r.ID != "temp.user" {
		t.Error("ByID(temp.user) 失败")
	}
	if _, ok := ByID("no.such"); ok {
		t.Error("ByID 对未知 ID 应返回 false")
	}
}
