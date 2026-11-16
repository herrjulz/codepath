// Package service validates, enriches, and stores walkthrough reports.
package service

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"codepath/internal/gitrev"
	"codepath/internal/redact"
	"codepath/internal/reports"
	"codepath/internal/source"
	"codepath/internal/store"
)

// ErrTooLarge is returned when a create body exceeds the size limit.
var ErrTooLarge = errors.New("request body exceeds 1 MiB")

// Service is the report application used by HTTP and the seed command.
type Service struct {
	RepoRoot string
	Files    *source.Reader
	Store    *store.Store
	BaseURL  string
}

// CreateResult is the success payload for a new report.
type CreateResult struct {
	ReportID string   `json:"reportId"`
	URL      string   `json:"url"`
	Warnings []string `json:"warnings"`
}

// Create validates a payload, resolves source spans, and persists the report.
func (s *Service) Create(ctx context.Context, body []byte) (CreateResult, error) {
	if len(body) > reports.MaxBodyBytes {
		return CreateResult{}, ErrTooLarge
	}
	req, err := reports.DecodeCreate(body)
	if err != nil {
		return CreateResult{}, err
	}
	if err := reports.Validate(req); err != nil {
		return CreateResult{}, err
	}
	warnings := make([]string, 0)
	if redactRequest(&req) {
		warnings = append(warnings, "Redacted possible secrets in the submitted text.")
	}
	revision, err := gitrev.Head(ctx, s.RepoRoot)
	if err != nil {
		revision = ""
		warnings = append(warnings, "Git revision unavailable; the report has no recorded HEAD.")
	}
	id, err := reports.NewID()
	if err != nil {
		return CreateResult{}, err
	}
	rep := reports.Build(req, reports.Meta{
		ID:          id,
		DisplayName: filepath.Base(s.RepoRoot),
		Revision:    revision,
		Now:         time.Now(),
	})
	ve := &reports.ValidationError{}
	for i := range rep.Steps {
		for j := range rep.Steps[i].CodeRefs {
			ref := &rep.Steps[i].CodeRefs[j]
			hash, _, err := s.Files.Inspect(ref.Path, ref.StartLine, ref.EndLine)
			if err != nil {
				ve.Add(fmt.Sprintf("steps[%d].codeRefs[%d]", i, j), err.Error())
				continue
			}
			ref.FileSha256 = hash
		}
	}
	if len(ve.Details) > 0 {
		return CreateResult{}, ve
	}
	if err := s.Store.Save(rep); err != nil {
		return CreateResult{}, err
	}
	return CreateResult{ReportID: rep.ID, URL: s.ReportURL(rep.ID), Warnings: warnings}, nil
}

// Get returns a stored report.
func (s *Service) Get(id string) (reports.WalkthroughReport, error) {
	return s.Store.Get(id)
}

// List returns report summaries.
func (s *Service) List() ([]reports.Summary, error) {
	return s.Store.List(s.ReportURL)
}

// Source loads one stored code reference.
func (s *Service) Source(id, stepID string, refIndex int) (source.View, error) {
	rep, err := s.Store.Get(id)
	if err != nil {
		return source.View{}, err
	}
	var step *reports.FlowStep
	for i := range rep.Steps {
		if rep.Steps[i].ID == stepID {
			step = &rep.Steps[i]
			break
		}
	}
	if step == nil || refIndex < 0 || refIndex >= len(step.CodeRefs) {
		return source.View{}, store.ErrNotFound
	}
	ref := step.CodeRefs[refIndex]
	view := s.Files.Load(ref.Path, ref.StartLine, ref.EndLine, ref.Label, ref.FileSha256)
	return view, nil
}

// ReportURL is the browser URL for a report.
func (s *Service) ReportURL(id string) string {
	return strings.TrimRight(s.BaseURL, "/") + "/reports/" + id
}

func redactRequest(req *reports.CreateRequest) bool {
	changed := false
	req.Title, changed = redactOne(req.Title, changed)
	req.Summary, changed = redactOne(req.Summary, changed)
	for i := range req.Overview.Sequence.Participants {
		req.Overview.Sequence.Participants[i].Label, changed = redactOne(req.Overview.Sequence.Participants[i].Label, changed)
	}
	for i := range req.Overview.Sequence.Messages {
		req.Overview.Sequence.Messages[i].Label, changed = redactOne(req.Overview.Sequence.Messages[i].Label, changed)
	}
	if req.Overview.Erd != nil {
		for i := range req.Overview.Erd.Entities {
			ent := &req.Overview.Erd.Entities[i]
			ent.Label, changed = redactOne(ent.Label, changed)
			for j := range ent.Attributes {
				ent.Attributes[j].Name, changed = redactOne(ent.Attributes[j].Name, changed)
				ent.Attributes[j].Type, changed = redactOne(ent.Attributes[j].Type, changed)
			}
		}
		for i := range req.Overview.Erd.Relationships {
			req.Overview.Erd.Relationships[i].Label, changed = redactOne(req.Overview.Erd.Relationships[i].Label, changed)
		}
	}
	for i := range req.Steps {
		step := &req.Steps[i]
		step.Title, changed = redactOne(step.Title, changed)
		step.Summary, changed = redactOne(step.Summary, changed)
		step.Explanation, changed = redactOne(step.Explanation, changed)
		for j := range step.Notes {
			step.Notes[j], changed = redactOne(step.Notes[j], changed)
		}
		for j := range step.CodeRefs {
			step.CodeRefs[j].Label, changed = redactOne(step.CodeRefs[j].Label, changed)
		}
	}
	for i := range req.Edges {
		req.Edges[i].Label, changed = redactOne(req.Edges[i].Label, changed)
	}
	for i := range req.Scenarios {
		sc := &req.Scenarios[i]
		sc.Name, changed = redactOne(sc.Name, changed)
		sc.Description, changed = redactOne(sc.Description, changed)
		for j := range sc.Examples {
			ex := &sc.Examples[j]
			ex.Transformation, changed = redactOne(ex.Transformation, changed)
			ex.Input.Content, changed = redactOne(ex.Input.Content, changed)
			ex.Output.Content, changed = redactOne(ex.Output.Content, changed)
			for k := range ex.FieldChanges {
				ch := &ex.FieldChanges[k]
				ch.Field, changed = redactOne(ch.Field, changed)
				ch.Before, changed = redactOne(ch.Before, changed)
				ch.After, changed = redactOne(ch.After, changed)
				ch.Why, changed = redactOne(ch.Why, changed)
			}
		}
	}
	return changed
}

func redactOne(value string, changed bool) (string, bool) {
	next, did := redact.Text(value)
	return next, changed || did
}
