// Package source reads repository-local file spans. It refuses paths that
// escape the configured root, including symlink escapes.
package source

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"codepath/internal/redact"
	"codepath/internal/reports"
)

// Reader reads files under a canonical repository root.
type Reader struct {
	Root string
}

// Open canonicalizes root and requires it to be a directory.
func Open(root string) (*Reader, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, fmt.Errorf("repository root: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("repository root is not a directory")
	}
	return &Reader{Root: resolved}, nil
}

// View is the source payload returned to the viewer.
type View struct {
	Path          string `json:"path"`
	Label         string `json:"label,omitempty"`
	StartLine     int    `json:"startLine"`
	EndLine       int    `json:"endLine"`
	Content       string `json:"content"`
	ContextStart  int    `json:"contextStart"`
	ContextEnd    int    `json:"contextEnd"`
	Context       string `json:"context"`
	Language      string `json:"language"`
	StoredSHA256  string `json:"storedSha256,omitempty"`
	CurrentSHA256 string `json:"currentSha256,omitempty"`
	Stale         bool   `json:"stale"`
	Missing       bool   `json:"missing"`
	Redacted      bool   `json:"redacted"`
	Message       string `json:"message,omitempty"`
}

// Inspect checks that a span exists and returns the file hash and line count.
func (r *Reader) Inspect(rel string, start, end int) (string, int, error) {
	data, err := r.read(rel)
	if err != nil {
		return "", 0, err
	}
	lines := splitLines(data)
	if start < 1 || end < start || end > len(lines) {
		return "", len(lines), fmt.Errorf("line range %d-%d is outside %s (%d lines)", start, end, rel, len(lines))
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), len(lines), nil
}

// Load reads a previously accepted span. Missing files and escapes become a
// warning view instead of a failure, and the recorded location is preserved.
func (r *Reader) Load(rel string, start, end int, label, storedHash string) View {
	view := View{
		Path:         rel,
		Label:        label,
		StartLine:    start,
		EndLine:      end,
		Language:     "go",
		StoredSHA256: storedHash,
	}
	data, err := r.read(rel)
	if err != nil {
		view.Missing = true
		view.Message = safeReadError(rel, err)
		return view
	}
	sum := sha256.Sum256(data)
	view.CurrentSHA256 = hex.EncodeToString(sum[:])
	if storedHash != "" && storedHash != view.CurrentSHA256 {
		view.Stale = true
		view.Message = "This file changed after the walkthrough was created. The highlighted lines may no longer match the explanation."
	}
	lines := splitLines(data)
	if start < 1 || end < start || end > len(lines) {
		view.Stale = true
		view.Message = fmt.Sprintf("The saved line range %d-%d is no longer inside %s (%d lines).", start, end, rel, len(lines))
		return view
	}
	span := lines[start-1 : end]
	from := start - reports.ContextRadius
	if from < 1 {
		from = 1
	}
	to := end + reports.ContextRadius
	if to > len(lines) {
		to = len(lines)
	}
	context := lines[from-1 : to]
	view.Content, view.Redacted = joinRedacted(span)
	var contextRedacted bool
	view.Context, contextRedacted = joinRedacted(context)
	view.Redacted = view.Redacted || contextRedacted
	view.ContextStart = from
	view.ContextEnd = to
	if view.Redacted && view.Message == "" {
		view.Message = "Obvious secret material in this span was redacted for display."
	}
	return view
}

func (r *Reader) read(rel string) ([]byte, error) {
	data, err := r.readResolved(rel)
	if err != nil {
		return nil, publicReadError(rel, err)
	}
	return data, nil
}

func (r *Reader) readResolved(rel string) ([]byte, error) {
	resolved, err := r.resolve(rel)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if info.Size() > reports.MaxFileBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", reports.MaxFileBytes)
	}
	f, err := os.Open(resolved)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, reports.MaxFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > reports.MaxFileBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", reports.MaxFileBytes)
	}
	return data, nil
}

func (r *Reader) resolve(rel string) (string, error) {
	if err := reports.ValidateRelPath(rel); err != nil {
		return "", err
	}
	joined := filepath.Join(r.Root, filepath.FromSlash(rel))
	if err := inside(r.Root, joined); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(joined)
	if err != nil {
		return "", err
	}
	if err := inside(r.Root, resolved); err != nil {
		return "", fmt.Errorf("path escapes the repository")
	}
	return resolved, nil
}

func inside(root, path string) error {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return fmt.Errorf("path escapes the repository")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes the repository")
	}
	return nil
}

func publicReadError(rel string, err error) error {
	if os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", rel)
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return fmt.Errorf("cannot read %s", rel)
	}
	return err
}

func safeReadError(rel string, err error) string {
	if os.IsNotExist(err) || strings.Contains(err.Error(), "does not exist") {
		return fmt.Sprintf("Source file %s no longer exists.", rel)
	}
	msg := err.Error()
	if strings.Contains(msg, "escapes") || strings.Contains(msg, "repository-relative") || strings.Contains(msg, "NUL") || strings.Contains(msg, "drive") {
		return "The stored path is not a safe repository-relative path."
	}
	if strings.Contains(msg, "exceeds") {
		return fmt.Sprintf("Source file %s is too large to display.", rel)
	}
	if strings.Contains(msg, "not a regular file") {
		return fmt.Sprintf("Source path %s is not a regular file.", rel)
	}
	return fmt.Sprintf("Source file %s could not be read.", rel)
}

func splitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	lines := strings.Split(s, "\n")
	if strings.HasSuffix(s, "\n") {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinRedacted(lines []string) (string, bool) {
	redacted := false
	out := make([]string, len(lines))
	for i, line := range lines {
		next, changed := redact.Line(line)
		out[i] = next
		redacted = redacted || changed
	}
	return strings.Join(out, "\n"), redacted
}
