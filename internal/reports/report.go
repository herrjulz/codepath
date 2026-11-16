// Package reports defines the walkthrough schema, validation, and IDs.
package reports

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// WalkthroughReport is a stored walkthrough. Server-owned fields are filled
// at creation time and are not accepted from the model.
type WalkthroughReport struct {
	SchemaVersion int        `json:"schemaVersion"`
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	Summary       string     `json:"summary"`
	Language      string     `json:"language"`
	Repository    Repository `json:"repository"`
	CreatedAt     string     `json:"createdAt"`
	Overview      Overview   `json:"overview"`
	Steps         []FlowStep `json:"steps"`
	Edges         []FlowEdge `json:"edges"`
	Scenarios     []Scenario `json:"scenarios"`
}

// Overview is the summary-entry projection authored with the walkthrough.
type Overview struct {
	Sequence SequenceDiagram `json:"sequence"`
	Erd      *ErdDiagram     `json:"erd,omitempty"`
}

// ErdDiagram is an optional authored entity-relationship diagram for the summary view.
type ErdDiagram struct {
	Entities      []ErdEntity       `json:"entities"`
	Relationships []ErdRelationship `json:"relationships"`
}

type ErdEntity struct {
	ID         string         `json:"id"`
	Label      string         `json:"label"`
	Attributes []ErdAttribute `json:"attributes,omitempty"`
}

type ErdAttribute struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
	Key  string `json:"key,omitempty"`
}

type ErdRelationship struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
	Kind  string `json:"kind"`
}

// SequenceDiagram is an authored participant/message sequence for the summary view.
type SequenceDiagram struct {
	Participants []Participant     `json:"participants"`
	Messages     []SequenceMessage `json:"messages"`
}

type Participant struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type SequenceMessage struct {
	ID     string `json:"id"`
	From   string `json:"from"`
	To     string `json:"to"`
	Label  string `json:"label"`
	Kind   string `json:"kind"`
	StepID string `json:"stepId,omitempty"`
}

type Repository struct {
	DisplayName string `json:"displayName"`
	Revision    string `json:"revision,omitempty"`
}

type FlowStep struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Summary     string    `json:"summary"`
	Explanation string    `json:"explanation"`
	CodeRefs    []CodeRef `json:"codeRefs"`
	Notes       []string  `json:"notes,omitempty"`
}

type CodeRef struct {
	Path       string `json:"path"`
	StartLine  int    `json:"startLine"`
	EndLine    int    `json:"endLine"`
	Label      string `json:"label,omitempty"`
	FileSha256 string `json:"fileSha256,omitempty"`
}

type FlowEdge struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label,omitempty"`
	Kind  string `json:"kind"`
}

type Scenario struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Path        []string      `json:"path"`
	Examples    []StepExample `json:"examples"`
}

type StepExample struct {
	StepID         string        `json:"stepId"`
	Kind           string        `json:"kind"`
	Input          ExampleValue  `json:"input"`
	Transformation string        `json:"transformation"`
	Output         ExampleValue  `json:"output"`
	FieldChanges   []FieldChange `json:"fieldChanges,omitempty"`
}

type FieldChange struct {
	Field  string `json:"field"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
	Why    string `json:"why"`
}

type ExampleValue struct {
	Format  string `json:"format"`
	Content string `json:"content"`
}

// CreateRequest is the model-supplied walkthrough. It deliberately omits
// server-owned fields so a submission cannot choose its id, revision, or hashes.
type CreateRequest struct {
	SchemaVersion int              `json:"schemaVersion" jsonschema:"Must be 1"`
	Title         string           `json:"title" jsonschema:"Short title of the walkthrough"`
	Summary       string           `json:"summary" jsonschema:"One paragraph describing the flow"`
	Language      string           `json:"language" jsonschema:"Programming language. Only go is accepted"`
	Overview      CreateOverview   `json:"overview" jsonschema:"Summary-entry sequence diagram for the high-level flow"`
	Steps         []CreateStep     `json:"steps" jsonschema:"Ordered flow nodes. Ids must be unique"`
	Edges         []CreateEdge     `json:"edges" jsonschema:"Directed edges between step ids"`
	Scenarios     []CreateScenario `json:"scenarios" jsonschema:"Named paths through the graph, each with synthetic examples"`
}

type CreateOverview struct {
	Sequence CreateSequence `json:"sequence" jsonschema:"Authored sequence diagram shown on the summary entry view"`
	Erd      *CreateErd     `json:"erd,omitempty" jsonschema:"Optional entity-relationship diagram when data shapes matter"`
}

type CreateErd struct {
	Entities      []CreateErdEntity       `json:"entities" jsonschema:"Entities or structs shown in the ERD"`
	Relationships []CreateErdRelationship `json:"relationships" jsonschema:"Relationships between entity ids"`
}

type CreateErdEntity struct {
	ID         string               `json:"id" jsonschema:"Stable entity id, starting with a letter"`
	Label      string               `json:"label" jsonschema:"Display name for the entity"`
	Attributes []CreateErdAttribute `json:"attributes,omitempty" jsonschema:"Optional fields shown inside the entity box"`
}

type CreateErdAttribute struct {
	Name string `json:"name" jsonschema:"Field or column name"`
	Type string `json:"type,omitempty" jsonschema:"Optional type label, such as string or int"`
	Key  string `json:"key,omitempty" jsonschema:"Optional key marker: pk, fk, or unique"`
}

type CreateErdRelationship struct {
	ID    string `json:"id" jsonschema:"Stable relationship id, starting with a letter"`
	From  string `json:"from" jsonschema:"Source entity id"`
	To    string `json:"to" jsonschema:"Destination entity id"`
	Label string `json:"label,omitempty" jsonschema:"Optional relationship label"`
	Kind  string `json:"kind" jsonschema:"One of one_to_one, one_to_many, many_to_one, many_to_many"`
}

type CreateSequence struct {
	Participants []CreateParticipant     `json:"participants" jsonschema:"Actors in the sequence, left to right"`
	Messages     []CreateSequenceMessage `json:"messages" jsonschema:"Ordered messages between participants"`
}

type CreateParticipant struct {
	ID    string `json:"id" jsonschema:"Stable participant id, starting with a letter"`
	Label string `json:"label" jsonschema:"Display name shown above the lifeline"`
}

type CreateSequenceMessage struct {
	ID     string `json:"id" jsonschema:"Stable message id, starting with a letter"`
	From   string `json:"from" jsonschema:"Source participant id"`
	To     string `json:"to" jsonschema:"Destination participant id"`
	Label  string `json:"label" jsonschema:"Short message label"`
	Kind   string `json:"kind" jsonschema:"One of call, return, note"`
	StepID string `json:"stepId,omitempty" jsonschema:"Optional step id this message jumps to in the detail view"`
}

type CreateStep struct {
	ID          string          `json:"id" jsonschema:"Stable step id, starting with a letter"`
	Title       string          `json:"title" jsonschema:"Short step title"`
	Summary     string          `json:"summary" jsonschema:"One sentence summary"`
	Explanation string          `json:"explanation" jsonschema:"Plain-language explanation of this step"`
	CodeRefs    []CreateCodeRef `json:"codeRefs" jsonschema:"Repository-relative file spans. At least one"`
	Notes       []string        `json:"notes,omitempty" jsonschema:"Optional assumptions or caveats"`
}

type CreateCodeRef struct {
	Path      string `json:"path" jsonschema:"Repository-relative path using forward slashes"`
	StartLine int    `json:"startLine" jsonschema:"First line of the span, 1-based and inclusive"`
	EndLine   int    `json:"endLine" jsonschema:"Last line of the span, 1-based and inclusive"`
	Label     string `json:"label,omitempty" jsonschema:"Short tab label for this span"`
}

type CreateEdge struct {
	ID    string `json:"id" jsonschema:"Stable edge id, starting with a letter"`
	From  string `json:"from" jsonschema:"Source step id"`
	To    string `json:"to" jsonschema:"Destination step id"`
	Label string `json:"label,omitempty" jsonschema:"Optional branch label, such as password valid"`
	Kind  string `json:"kind" jsonschema:"One of next, condition, error, async"`
}

type CreateScenario struct {
	ID          string              `json:"id" jsonschema:"Stable scenario id, starting with a letter"`
	Name        string              `json:"name" jsonschema:"Human name, such as Successful login"`
	Description string              `json:"description,omitempty" jsonschema:"Optional scenario description"`
	Path        []string            `json:"path" jsonschema:"Ordered step ids from first to last"`
	Examples    []CreateStepExample `json:"examples" jsonschema:"One synthetic example for each step on the path"`
}

type CreateStepExample struct {
	StepID         string              `json:"stepId" jsonschema:"Step id this example illustrates"`
	Kind           string              `json:"kind" jsonschema:"Must be synthetic. Observed examples are rejected"`
	Input          ExampleValue        `json:"input" jsonschema:"Illustrative input value"`
	Transformation string              `json:"transformation" jsonschema:"What this step does to the data"`
	Output         ExampleValue        `json:"output" jsonschema:"Illustrative output value"`
	FieldChanges   []CreateFieldChange `json:"fieldChanges,omitempty" jsonschema:"Meaningful field changes, not a runtime trace"`
}

type CreateFieldChange struct {
	Field  string `json:"field" jsonschema:"Field or binding that changed"`
	Before string `json:"before,omitempty" jsonschema:"Previous value, when useful"`
	After  string `json:"after,omitempty" jsonschema:"New value, when useful"`
	Why    string `json:"why" jsonschema:"Why the value changed"`
}

// Summary is the list-view projection of a report.
type Summary struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Summary    string `json:"summary"`
	Language   string `json:"language"`
	Repository string `json:"repository"`
	Revision   string `json:"revision,omitempty"`
	CreatedAt  string `json:"createdAt"`
	URL        string `json:"url"`
}

// FieldError points at one invalid field so a model can correct and resubmit.
type FieldError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ValidationError is a collection of actionable field errors.
type ValidationError struct {
	Details []FieldError
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Details) == 0 {
		return "validation failed"
	}
	var b strings.Builder
	b.WriteString("validation failed:")
	for _, d := range e.Details {
		b.WriteString("\n- ")
		b.WriteString(d.Path)
		b.WriteString(": ")
		b.WriteString(d.Message)
	}
	return b.String()
}

func (e *ValidationError) Add(path, message string) {
	if len(e.Details) >= MaxValidationItems {
		return
	}
	e.Details = append(e.Details, FieldError{Path: path, Message: message})
}

// DecodeCreate parses a create payload and rejects unknown fields, including
// server-owned ones such as id, repository, and fileSha256.
func DecodeCreate(body []byte) (CreateRequest, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var req CreateRequest
	if err := dec.Decode(&req); err != nil {
		return CreateRequest{}, fmt.Errorf("invalid JSON: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return CreateRequest{}, errors.New("invalid JSON: trailing data")
	}
	return req, nil
}

// Meta holds the server-owned values stamped onto a new report.
type Meta struct {
	ID          string
	DisplayName string
	Revision    string
	Now         time.Time
}

// Build copies a validated request into a stored report. Callers fill file hashes.
func Build(req CreateRequest, meta Meta) WalkthroughReport {
	participants := make([]Participant, len(req.Overview.Sequence.Participants))
	for i, p := range req.Overview.Sequence.Participants {
		participants[i] = Participant{
			ID:    p.ID,
			Label: strings.TrimSpace(p.Label),
		}
	}
	messages := make([]SequenceMessage, len(req.Overview.Sequence.Messages))
	for i, m := range req.Overview.Sequence.Messages {
		messages[i] = SequenceMessage{
			ID:     m.ID,
			From:   m.From,
			To:     m.To,
			Label:  strings.TrimSpace(m.Label),
			Kind:   m.Kind,
			StepID: m.StepID,
		}
	}
	rep := WalkthroughReport{
		SchemaVersion: SchemaVersion,
		ID:            meta.ID,
		Title:         strings.TrimSpace(req.Title),
		Summary:       strings.TrimSpace(req.Summary),
		Language:      req.Language,
		Repository: Repository{
			DisplayName: meta.DisplayName,
			Revision:    meta.Revision,
		},
		CreatedAt: meta.Now.UTC().Format(time.RFC3339),
		Overview: Overview{
			Sequence: SequenceDiagram{
				Participants: participants,
				Messages:     messages,
			},
			Erd: buildErd(req.Overview.Erd),
		},
		Steps:     make([]FlowStep, len(req.Steps)),
		Edges:     make([]FlowEdge, len(req.Edges)),
		Scenarios: make([]Scenario, len(req.Scenarios)),
	}
	for i, step := range req.Steps {
		notes := append([]string(nil), step.Notes...)
		refs := make([]CodeRef, len(step.CodeRefs))
		for j, ref := range step.CodeRefs {
			refs[j] = CodeRef{
				Path:      ref.Path,
				StartLine: ref.StartLine,
				EndLine:   ref.EndLine,
				Label:     strings.TrimSpace(ref.Label),
			}
		}
		rep.Steps[i] = FlowStep{
			ID:          step.ID,
			Title:       strings.TrimSpace(step.Title),
			Summary:     strings.TrimSpace(step.Summary),
			Explanation: strings.TrimSpace(step.Explanation),
			CodeRefs:    refs,
			Notes:       notes,
		}
	}
	for i, edge := range req.Edges {
		rep.Edges[i] = FlowEdge{
			ID:    edge.ID,
			From:  edge.From,
			To:    edge.To,
			Label: strings.TrimSpace(edge.Label),
			Kind:  edge.Kind,
		}
	}
	for i, sc := range req.Scenarios {
		byStep := make(map[string]CreateStepExample, len(sc.Examples))
		for _, ex := range sc.Examples {
			byStep[ex.StepID] = ex
		}
		examples := make([]StepExample, 0, len(sc.Path))
		for _, stepID := range sc.Path {
			ex := byStep[stepID]
			changes := make([]FieldChange, len(ex.FieldChanges))
			for j, ch := range ex.FieldChanges {
				changes[j] = FieldChange{
					Field:  strings.TrimSpace(ch.Field),
					Before: ch.Before,
					After:  ch.After,
					Why:    strings.TrimSpace(ch.Why),
				}
			}
			examples = append(examples, StepExample{
				StepID:         ex.StepID,
				Kind:           ex.Kind,
				Input:          ex.Input,
				Transformation: strings.TrimSpace(ex.Transformation),
				Output:         ex.Output,
				FieldChanges:   changes,
			})
		}
		rep.Scenarios[i] = Scenario{
			ID:          sc.ID,
			Name:        strings.TrimSpace(sc.Name),
			Description: strings.TrimSpace(sc.Description),
			Path:        append([]string(nil), sc.Path...),
			Examples:    examples,
		}
	}
	return rep
}

func buildErd(src *CreateErd) *ErdDiagram {
	if src == nil {
		return nil
	}
	entities := make([]ErdEntity, len(src.Entities))
	for i, ent := range src.Entities {
		attrs := make([]ErdAttribute, len(ent.Attributes))
		for j, attr := range ent.Attributes {
			attrs[j] = ErdAttribute{
				Name: strings.TrimSpace(attr.Name),
				Type: strings.TrimSpace(attr.Type),
				Key:  attr.Key,
			}
		}
		entities[i] = ErdEntity{
			ID:         ent.ID,
			Label:      strings.TrimSpace(ent.Label),
			Attributes: attrs,
		}
	}
	rels := make([]ErdRelationship, len(src.Relationships))
	for i, rel := range src.Relationships {
		rels[i] = ErdRelationship{
			ID:    rel.ID,
			From:  rel.From,
			To:    rel.To,
			Label: strings.TrimSpace(rel.Label),
			Kind:  rel.Kind,
		}
	}
	return &ErdDiagram{Entities: entities, Relationships: rels}
}

// SummaryOf projects a report for the list endpoint.
func SummaryOf(rep WalkthroughReport, url string) Summary {
	return Summary{
		ID:         rep.ID,
		Title:      rep.Title,
		Summary:    rep.Summary,
		Language:   rep.Language,
		Repository: rep.Repository.DisplayName,
		Revision:   rep.Repository.Revision,
		CreatedAt:  rep.CreatedAt,
		URL:        url,
	}
}
