package store

import (
	"os"
	"strings"
	"testing"

	"codepath/internal/reports"
)

func TestSaveReopenAndList(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	id, err := reports.NewID()
	if err != nil {
		t.Fatal(err)
	}
	rep := reports.WalkthroughReport{
		SchemaVersion: reports.SchemaVersion,
		ID:            id,
		Title:         "Persisted",
		Summary:       "Still here.",
		Language:      "go",
		Repository:    reports.Repository{DisplayName: "demo"},
		CreatedAt:     "2026-01-02T03:04:05Z",
		Steps: []reports.FlowStep{{
			ID: "only", Title: "Only", Summary: "s", Explanation: "e",
			CodeRefs: []reports.CodeRef{{Path: "main.go", StartLine: 1, EndLine: 1, FileSha256: "abc"}},
		}},
		Edges: []reports.FlowEdge{},
		Scenarios: []reports.Scenario{{
			ID: "one", Name: "One", Path: []string{"only"},
			Examples: []reports.StepExample{{
				StepID: "only", Kind: "synthetic",
				Input:          reports.ExampleValue{Format: "text", Content: "a"},
				Transformation: "t",
				Output:         reports.ExampleValue{Format: "text", Content: "b"},
			}},
		}},
	}
	if err := st.Save(rep); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(st.Dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp") {
			t.Fatalf("temp file left behind: %s", entry.Name())
		}
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != rep.Title || got.Steps[0].CodeRefs[0].FileSha256 != "abc" {
		t.Fatalf("round trip: %+v", got)
	}
	list, err := reopened.List(func(id string) string { return "http://127.0.0.1/reports/" + id })
	if err != nil || len(list) != 1 || list[0].URL != "http://127.0.0.1/reports/"+id {
		t.Fatalf("list: %+v %v", list, err)
	}
	if _, err := reopened.Get("../" + id); !errorsIsNotFound(err) {
		t.Fatalf("unsafe id error: %v", err)
	}
	if err := st.Save(reports.WalkthroughReport{ID: "../evil"}); err == nil {
		t.Fatal("saved unsafe id")
	}
}

func errorsIsNotFound(err error) bool {
	return err == ErrNotFound
}
