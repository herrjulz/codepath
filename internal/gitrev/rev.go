// Package gitrev reads the repository HEAD without accepting any caller-supplied
// command text. Failure is expected when git is unavailable.
package gitrev

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var revisionPattern = regexp.MustCompile(`^[0-9a-f]{40}([0-9a-f]{24})?$`)

// Head returns the current commit hash, or an error when it cannot be read.
func Head(ctx context.Context, repo string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", repo, "rev-parse", "HEAD")
	cmd.Env = withoutGitOverrides(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	rev := strings.TrimSpace(string(out))
	if !revisionPattern.MatchString(rev) {
		return "", errors.New("unexpected git revision")
	}
	return rev, nil
}

func withoutGitOverrides(env []string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, "GIT_DIR=") || strings.HasPrefix(entry, "GIT_WORK_TREE=") || strings.HasPrefix(entry, "GIT_NAMESPACE=") {
			continue
		}
		out = append(out, entry)
	}
	return out
}
