// Package config resolves the repository root, the per-repository data
// directory, and the local API token shared by serve and mcp.
package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CanonicalRepo resolves path to an absolute directory with symlinks evaluated.
func CanonicalRepo(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("--repo is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("repository path: %w", err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repository path is not a directory: %s", resolved)
	}
	return resolved, nil
}

// DataDir returns the storage directory for a canonical repository.
// An override is used as-is after it is made absolute. The default directory
// lives in the user config dir and is keyed by the repository path.
func DataDir(repo, override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		return filepath.Abs(override)
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(repo))
	return filepath.Join(base, "codepath", "repos", hex.EncodeToString(sum[:8])), nil
}

// EnsureToken returns the bearer token, creating a mode-0600 file when needed.
func EnsureToken(dataDir string) (string, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dataDir, "token")
	if data, err := os.ReadFile(path); err == nil {
		tok := strings.TrimSpace(string(data))
		if tok == "" {
			return "", errors.New("token file is empty")
		}
		_ = os.Chmod(path, 0o600)
		return tok, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	tok := hex.EncodeToString(buf)
	if err := os.WriteFile(path, []byte(tok+"\n"), 0o600); err != nil {
		return "", err
	}
	return tok, nil
}

// ReadToken reads the token created by serve. It does not create one.
func ReadToken(dataDir string) (string, error) {
	path := filepath.Join(dataDir, "token")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("no local token found; start codepath serve once so it can create the token")
		}
		return "", err
	}
	tok := strings.TrimSpace(string(data))
	if len(tok) < 32 {
		return "", errors.New("local token file is empty or invalid; start codepath serve again after removing it")
	}
	return tok, nil
}
