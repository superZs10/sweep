package clean

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/superZs10/sweep/internal/rules"
	"github.com/superZs10/sweep/internal/scan"
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

func TestApplyDeleteContents(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.bin"), 100)
	writeFile(t, filepath.Join(dir, "sub", "b.bin"), 200)

	r := rules.Rule{ID: "test", Action: rules.DeleteContents, Paths: []string{dir}}
	item, err := scan.Run(r)
	if err != nil {
		t.Fatal(err)
	}

	res := Apply(item, Options{})
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if res.Freed != 300 {
		t.Errorf("freed = %d, want 300", res.Freed)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("目录未清空，还剩 %d 项", len(entries))
	}
}

func TestApplyDryRunKeepsFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.bin"), 10)

	r := rules.Rule{ID: "test", Action: rules.DeleteContents, Paths: []string{dir}}
	item, _ := scan.Run(r)

	res := Apply(item, Options{DryRun: true})
	if res.Freed != 10 {
		t.Errorf("dry-run freed = %d, want 10", res.Freed)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.bin")); err != nil {
		t.Error("dry-run 不应删除文件")
	}
}

func TestApplyDeleteFilesRespectsMinAge(t *testing.T) {
	dir := t.TempDir()
	oldFile := filepath.Join(dir, "old.exe")
	newFile := filepath.Join(dir, "new.exe")
	writeFile(t, oldFile, 100)
	writeFile(t, newFile, 100)

	old := time.Now().Add(-200 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, old, old); err != nil {
		t.Fatal(err)
	}

	r := rules.Rule{
		ID:       "test",
		Action:   rules.DeleteFiles,
		Paths:    []string{dir},
		Patterns: []string{"*.exe"},
		MinAge:   90 * 24 * time.Hour,
	}
	item, _ := scan.Run(r)
	res := Apply(item, Options{})
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("旧文件应被删除")
	}
	if _, err := os.Stat(newFile); err != nil {
		t.Error("新文件不应被删除")
	}
}
