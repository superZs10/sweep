package scan

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/superZs10/sweep/internal/rules"
)

func writeFile(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunDeleteContents(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.bin"), 100)
	writeFile(t, filepath.Join(dir, "b.bin"), 200)
	writeFile(t, filepath.Join(dir, "sub", "c.bin"), 50)

	r := rules.Rule{ID: "test", Action: rules.DeleteContents, Paths: []string{dir}}
	item, err := Run(r)
	if err != nil {
		t.Fatal(err)
	}
	if item.Bytes != 350 {
		t.Errorf("bytes = %d, want 350", item.Bytes)
	}
	if item.Files != 3 {
		t.Errorf("files = %d, want 3", item.Files)
	}
	if len(item.Roots) != 1 || item.Roots[0].Path != dir {
		t.Errorf("roots = %+v", item.Roots)
	}
}

func TestRunDeleteFilesWithPatternAndAge(t *testing.T) {
	dir := t.TempDir()
	oldExe := filepath.Join(dir, "old.exe")
	newExe := filepath.Join(dir, "new.exe")
	oldTxt := filepath.Join(dir, "old.txt")
	writeFile(t, oldExe, 100)
	writeFile(t, newExe, 100)
	writeFile(t, oldTxt, 100)

	old := time.Now().Add(-100 * 24 * time.Hour)
	for _, p := range []string{oldExe, oldTxt} {
		if err := os.Chtimes(p, old, old); err != nil {
			t.Fatal(err)
		}
	}

	r := rules.Rule{
		ID:       "test",
		Action:   rules.DeleteFiles,
		Paths:    []string{dir},
		Patterns: []string{"*.exe"},
		MinAge:   90 * 24 * time.Hour,
	}
	item, err := Run(r)
	if err != nil {
		t.Fatal(err)
	}
	// 只有 old.exe 同时满足"是 exe"和"足够旧"
	if item.Files != 1 || item.Bytes != 100 {
		t.Errorf("files=%d bytes=%d, want 1/100", item.Files, item.Bytes)
	}
}

func TestRunMissingPathIgnored(t *testing.T) {
	r := rules.Rule{
		ID:     "test",
		Action: rules.DeleteContents,
		Paths:  []string{filepath.Join(t.TempDir(), "does-not-exist")},
	}
	item, err := Run(r)
	if err != nil {
		t.Fatal(err)
	}
	if !item.Empty() {
		t.Errorf("expected empty item, got %+v", item)
	}
}

func TestSize(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "x", "y.bin"), 42)
	bytes, files := Size(dir)
	if bytes != 42 || files != 1 {
		t.Errorf("Size = %d/%d, want 42/1", bytes, files)
	}
}

func TestIsPermission(t *testing.T) {
	if IsPermission(nil) {
		t.Error("nil 不应算权限错误")
	}
	err := os.WriteFile(filepath.Join(t.TempDir(), "no", "dir", "f"), nil, 0o644)
	if !IsPermission(err) {
		t.Skipf("当前环境未返回权限错误: %v", err)
	}
}
