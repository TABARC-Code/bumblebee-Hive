package readlimit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadBoundedSmallFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "small.txt")
	if err := os.WriteFile(p, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := ReadBounded(p, 1024, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("data = %q, want %q", data, "hello")
	}
}

func TestReadBoundedExceedsMax(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(p, []byte("hello world"), 0o644); err != nil {
		t.Fatal(err)
	}
	var diagLevel, diagMsg string
	diag := func(level, path, msg string) { diagLevel = level; diagMsg = msg }
	_, err := ReadBounded(p, 5, diag)
	if err == nil {
		t.Fatal("expected error for file exceeding max size")
	}
	if !strings.Contains(err.Error(), "max size") {
		t.Fatalf("error = %q, want max size message", err)
	}
	if diagLevel != "warn" {
		t.Fatalf("diag level = %q, want warn", diagLevel)
	}
	if diagMsg == "" {
		t.Fatal("expected non-empty diag message")
	}
}

func TestReadBoundedExactlyMax(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "exact.txt")
	content := []byte("12345")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	// max == len(content): should succeed.
	data, err := ReadBounded(p, int64(len(content)), nil)
	if err != nil {
		t.Fatalf("unexpected error at exact max: %v", err)
	}
	if string(data) != string(content) {
		t.Fatalf("data = %q, want %q", data, content)
	}
}

func TestReadBoundedUnbounded(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "any.txt")
	if err := os.WriteFile(p, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	// max <= 0 means unbounded.
	data, err := ReadBounded(p, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error with unbounded read: %v", err)
	}
	if string(data) != "data" {
		t.Fatalf("data = %q, want %q", data, "data")
	}
}

func TestReadBoundedMissingFile(t *testing.T) {
	_, err := ReadBounded(filepath.Join(t.TempDir(), "missing.txt"), 1024, nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadBoundedNonRegularFile(t *testing.T) {
	// Directories are not regular files.
	_, err := ReadBounded(t.TempDir(), 1024, nil)
	if err == nil {
		t.Fatal("expected error for non-regular file (directory)")
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("error = %q, want 'not a regular file'", err)
	}
}

func TestReadBoundedNilDiagOnExceed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(p, []byte("too long"), 0o644); err != nil {
		t.Fatal(err)
	}
	// nil diag must not panic.
	_, err := ReadBounded(p, 3, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
