package outputs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearDirectoryRemovesChildren(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "old.bin"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ClearDirectory(dir); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty dir, got %d entries", len(entries))
	}
}

func TestClearDirectoriesUniqueTwiceSamePath(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "out")
	if err := os.MkdirAll(filepath.Join(p, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := ClearDirectoriesUnique([]string{p, p}); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(p)
	if len(entries) != 0 {
		t.Fatalf("expected cleared once, got %d", len(entries))
	}
}
