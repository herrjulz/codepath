package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathGuards(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ok.go"), []byte("package ok\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "leak.go")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "ok.go"), filepath.Join(root, "inside.go")); err != nil {
		t.Fatal(err)
	}
	reader, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}

	if _, _, err := reader.Inspect("../ok.go", 1, 1); err == nil {
		t.Fatal("accepted parent path")
	}
	if _, _, err := reader.Inspect("/etc/passwd", 1, 1); err == nil {
		t.Fatal("accepted absolute path")
	}
	if _, _, err := reader.Inspect("leak.go", 1, 1); err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("symlink escape error: %v", err)
	}
	view := reader.Load("leak.go", 1, 1, "", "")
	if !view.Missing || strings.Contains(view.Content, "secret") || strings.Contains(view.Context, "secret") {
		t.Fatalf("leaked symlink view: %+v", view)
	}
	if strings.Contains(view.Message, outside) {
		t.Fatalf("message leaked outside path: %s", view.Message)
	}

	hash, lines, err := reader.Inspect("inside.go", 1, 1)
	if err != nil || lines != 1 || hash == "" {
		t.Fatalf("internal symlink: hash=%s lines=%d err=%v", hash, lines, err)
	}
	if _, _, err := reader.Inspect("ok.go", 1, 9); err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("line range error: %v", err)
	}
	if _, _, err := reader.Inspect("missing.go", 1, 1); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing file error: %v", err)
	}

	missing := reader.Load("missing.go", 1, 1, "", hash)
	if !missing.Missing || missing.Content != "" {
		t.Fatalf("missing view: %+v", missing)
	}

	if err := os.WriteFile(filepath.Join(root, "ok.go"), []byte("package ok\n\nfunc changed() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stale := reader.Load("ok.go", 1, 1, "", hash)
	if !stale.Stale || !strings.Contains(stale.Content, "package ok") {
		t.Fatalf("stale view: %+v", stale)
	}
}
