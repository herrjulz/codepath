package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"codepath/internal/config"
	"codepath/internal/reports"
	"codepath/internal/service"
	"codepath/internal/source"
	"codepath/internal/store"
)

// ModuleRoot returns the repository root that contains go.mod.
func ModuleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

// RepoWithFile creates a temporary repository containing main.go.
func RepoWithFile(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	rel := "main.go"
	body := "package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n"
	if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, rel
}

// Sample is a structurally valid walkthrough whose code refs point at rel.
func Sample(rel string) reports.CreateRequest {
	return reports.CreateRequest{
		SchemaVersion: reports.SchemaVersion,
		Title:         "Sample flow",
		Summary:       "A small sample used by tests.",
		Language:      "go",
		Overview: reports.CreateOverview{
			Sequence: reports.CreateSequence{
				Participants: []reports.CreateParticipant{
					{ID: "caller", Label: "Caller"},
					{ID: "worker", Label: "Worker"},
				},
				Messages: []reports.CreateSequenceMessage{
					{ID: "m_start", From: "caller", To: "worker", Label: "start", Kind: "call", StepID: "start"},
					{ID: "m_done", From: "worker", To: "caller", Label: "done", Kind: "return", StepID: "finish"},
				},
			},
		},
		Steps: []reports.CreateStep{
			{
				ID:          "start",
				Title:       "Start",
				Summary:     "The first step.",
				Explanation: "It begins here.",
				CodeRefs: []reports.CreateCodeRef{{
					Path:      rel,
					StartLine: 1,
					EndLine:   4,
					Label:     "main",
				}},
			},
			{
				ID:          "finish",
				Title:       "Finish",
				Summary:     "The last step.",
				Explanation: "It ends here.",
				CodeRefs: []reports.CreateCodeRef{{
					Path:      rel,
					StartLine: 3,
					EndLine:   5,
				}},
			},
		},
		Edges: []reports.CreateEdge{{
			ID:    "to_finish",
			From:  "start",
			To:    "finish",
			Label: "ok",
			Kind:  "next",
		}},
		Scenarios: []reports.CreateScenario{{
			ID:   "happy",
			Name: "Happy path",
			Path: []string{"start", "finish"},
			Examples: []reports.CreateStepExample{
				{
					StepID:         "start",
					Kind:           "synthetic",
					Input:          reports.ExampleValue{Format: "text", Content: "in"},
					Transformation: "passes through",
					Output:         reports.ExampleValue{Format: "text", Content: "mid"},
				},
				{
					StepID:         "finish",
					Kind:           "synthetic",
					Input:          reports.ExampleValue{Format: "text", Content: "mid"},
					Transformation: "finishes",
					Output:         reports.ExampleValue{Format: "text", Content: "out"},
				},
			},
		}},
	}
}

// MustJSON marshals v or fails the test.
func MustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// NewService builds a service rooted at repo with an isolated data directory.
// The returned token is the bearer token for POST /api/reports.
func NewService(t *testing.T, repo, baseURL string) (*service.Service, string) {
	t.Helper()
	files, err := source.Open(repo)
	if err != nil {
		t.Fatal(err)
	}
	dataDir := t.TempDir()
	token, err := config.EnsureToken(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	return &service.Service{
		RepoRoot: files.Root,
		Files:    files,
		Store:    st,
		BaseURL:  baseURL,
	}, token
}
