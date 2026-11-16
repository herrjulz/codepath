package reports

import (
	"strings"
	"testing"
)

func baseRequest() CreateRequest {
	return CreateRequest{
		SchemaVersion: SchemaVersion,
		Title:         "Login",
		Summary:       "How login works.",
		Language:      "go",
		Overview: CreateOverview{
			Sequence: CreateSequence{
				Participants: []CreateParticipant{
					{ID: "client", Label: "Client"},
					{ID: "server", Label: "Server"},
				},
				Messages: []CreateSequenceMessage{
					{ID: "m1", From: "client", To: "server", Label: "request", Kind: "call", StepID: "parse"},
					{ID: "m2", From: "server", To: "client", Label: "response", Kind: "return", StepID: "check"},
				},
			},
		},
		Steps: []CreateStep{
			step("parse", "main.go", 1, 2),
			step("check", "main.go", 3, 4),
		},
		Edges: []CreateEdge{{
			ID:   "next",
			From: "parse",
			To:   "check",
			Kind: "next",
		}},
		Scenarios: []CreateScenario{{
			ID:   "ok",
			Name: "Success",
			Path: []string{"parse", "check"},
			Examples: []CreateStepExample{
				example("parse"),
				example("check"),
			},
		}},
	}
}

func step(id, path string, start, end int) CreateStep {
	return CreateStep{
		ID:          id,
		Title:       id,
		Summary:     "Summary for " + id,
		Explanation: "Explanation for " + id,
		CodeRefs:    []CreateCodeRef{{Path: path, StartLine: start, EndLine: end}},
	}
}

func example(stepID string) CreateStepExample {
	return CreateStepExample{
		StepID:         stepID,
		Kind:           "synthetic",
		Input:          ExampleValue{Format: "text", Content: "in"},
		Transformation: "changes the value",
		Output:         ExampleValue{Format: "text", Content: "out"},
	}
}

func TestValidateAcceptsBase(t *testing.T) {
	if err := Validate(baseRequest()); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := []struct {
		name string
		edit func(*CreateRequest)
		want string
	}{
		{"schema", func(r *CreateRequest) { r.SchemaVersion = 2 }, "schemaVersion"},
		{"language", func(r *CreateRequest) { r.Language = "python" }, "language"},
		{"title", func(r *CreateRequest) { r.Title = strings.Repeat("a", MaxTitle+1) }, "title"},
		{"duplicate step", func(r *CreateRequest) { r.Steps[1].ID = "parse" }, "duplicate step"},
		{"path traversal", func(r *CreateRequest) { r.Steps[0].CodeRefs[0].Path = "../secret.go" }, "parent"},
		{"absolute", func(r *CreateRequest) { r.Steps[0].CodeRefs[0].Path = "/etc/passwd" }, "repository-relative"},
		{"nul", func(r *CreateRequest) { r.Steps[0].CodeRefs[0].Path = "a\x00b.go" }, "NUL"},
		{"line order", func(r *CreateRequest) { r.Steps[0].CodeRefs[0].EndLine = 0 }, "line range"},
		{"span", func(r *CreateRequest) { r.Steps[0].CodeRefs[0].EndLine = MaxSpanLines + 5 }, "span"},
		{"missing edge", func(r *CreateRequest) { r.Edges = nil }, "no edge"},
		{"observed", func(r *CreateRequest) { r.Scenarios[0].Examples[0].Kind = "observed" }, "synthetic"},
		{"missing example", func(r *CreateRequest) {
			r.Scenarios[0].Examples = r.Scenarios[0].Examples[:1]
		}, "exactly one example"},
		{"duplicate example", func(r *CreateRequest) {
			r.Scenarios[0].Examples[1].StepID = "parse"
		}, "duplicate example"},
		{"unknown step", func(r *CreateRequest) { r.Scenarios[0].Path[1] = "missing" }, "existing step"},
		{"missing overview participants", func(r *CreateRequest) {
			r.Overview.Sequence.Participants = r.Overview.Sequence.Participants[:1]
		}, "at least 2 participants"},
		{"bad message participant", func(r *CreateRequest) {
			r.Overview.Sequence.Messages[0].From = "ghost"
		}, "existing participant"},
		{"bad message step", func(r *CreateRequest) {
			r.Overview.Sequence.Messages[0].StepID = "ghost"
		}, "existing step"},
		{"note must be self", func(r *CreateRequest) {
			r.Overview.Sequence.Messages[0].Kind = "note"
		}, "same from and to"},
		{"bad erd entity", func(r *CreateRequest) {
			r.Overview.Erd = &CreateErd{
				Entities: []CreateErdEntity{{ID: "user", Label: "User"}},
				Relationships: []CreateErdRelationship{{
					ID: "r1", From: "user", To: "ghost", Kind: "one_to_many",
				}},
			}
		}, "existing entity"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := baseRequest()
			tc.edit(&req)
			err := Validate(req)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestNewID(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 20; i++ {
		id, err := NewID()
		if err != nil {
			t.Fatal(err)
		}
		if !ValidID(id) {
			t.Fatalf("invalid id %s", id)
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id %s", id)
		}
		seen[id] = struct{}{}
	}
	if ValidID("../etc/passwd") || ValidID("not-an-id") {
		t.Fatal("accepted unsafe id")
	}
}
