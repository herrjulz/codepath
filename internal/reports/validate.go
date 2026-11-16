package reports

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

var (
	edgeKinds     = map[string]struct{}{"next": {}, "condition": {}, "error": {}, "async": {}}
	exampleFormat = map[string]struct{}{"json": {}, "go": {}, "http": {}, "text": {}}
	messageKinds  = map[string]struct{}{"call": {}, "return": {}, "note": {}}
	erdRelKinds   = map[string]struct{}{"one_to_one": {}, "one_to_many": {}, "many_to_one": {}, "many_to_many": {}}
	erdAttrKeys   = map[string]struct{}{"pk": {}, "fk": {}, "unique": {}}
)

// Validate checks the structural rules of a create request. It does not read
// the repository; file existence and line bounds are checked separately.
func Validate(req CreateRequest) error {
	ve := &ValidationError{}
	if req.SchemaVersion != SchemaVersion {
		ve.Add("schemaVersion", fmt.Sprintf("must be %d", SchemaVersion))
	}
	requireText(ve, "title", req.Title, MaxTitle)
	requireText(ve, "summary", req.Summary, MaxSummary)
	if req.Language != "go" {
		ve.Add("language", `must be "go"`)
	}
	if len(req.Steps) == 0 {
		ve.Add("steps", "at least one step is required")
	}
	if len(req.Steps) > MaxSteps {
		ve.Add("steps", fmt.Sprintf("at most %d steps", MaxSteps))
	}
	if len(req.Edges) > MaxEdges {
		ve.Add("edges", fmt.Sprintf("at most %d edges", MaxEdges))
	}
	if len(req.Scenarios) == 0 {
		ve.Add("scenarios", "at least one scenario is required")
	}
	if len(req.Scenarios) > MaxScenarios {
		ve.Add("scenarios", fmt.Sprintf("at most %d scenarios", MaxScenarios))
	}

	steps := make(map[string]struct{}, len(req.Steps))
	for i, step := range req.Steps {
		p := fmt.Sprintf("steps[%d]", i)
		if !validIdent(step.ID) {
			ve.Add(p+".id", "must start with a letter and contain only letters, digits, underscores, or hyphens")
		} else if _, ok := steps[step.ID]; ok {
			ve.Add(p+".id", "duplicate step id "+step.ID)
		} else {
			steps[step.ID] = struct{}{}
		}
		requireText(ve, p+".title", step.Title, MaxTitle)
		requireText(ve, p+".summary", step.Summary, MaxStepSummary)
		requireText(ve, p+".explanation", step.Explanation, MaxExplanation)
		if len(step.CodeRefs) == 0 {
			ve.Add(p+".codeRefs", "at least one code reference is required")
		}
		if len(step.CodeRefs) > MaxRefsPerStep {
			ve.Add(p+".codeRefs", fmt.Sprintf("at most %d code references", MaxRefsPerStep))
		}
		for j, ref := range step.CodeRefs {
			rp := fmt.Sprintf("%s.codeRefs[%d]", p, j)
			if err := ValidateRelPath(ref.Path); err != nil {
				ve.Add(rp+".path", err.Error())
			}
			if ref.StartLine < 1 || ref.EndLine < ref.StartLine {
				ve.Add(rp, "line range must be positive and ordered")
			} else if ref.EndLine-ref.StartLine+1 > MaxSpanLines {
				ve.Add(rp, fmt.Sprintf("span exceeds %d lines", MaxSpanLines))
			}
			if strings.TrimSpace(ref.Label) != "" && len(ref.Label) > MaxLabel {
				ve.Add(rp+".label", fmt.Sprintf("exceeds %d bytes", MaxLabel))
			}
		}
		if len(step.Notes) > MaxNotes {
			ve.Add(p+".notes", fmt.Sprintf("at most %d notes", MaxNotes))
		}
		for j, note := range step.Notes {
			if strings.TrimSpace(note) == "" {
				ve.Add(fmt.Sprintf("%s.notes[%d]", p, j), "must not be empty")
			} else if len(note) > MaxNoteLen {
				ve.Add(fmt.Sprintf("%s.notes[%d]", p, j), fmt.Sprintf("exceeds %d bytes", MaxNoteLen))
			}
		}
	}

	validateOverview(ve, req.Overview, steps)

	edges := make(map[string]struct{}, len(req.Edges))
	type pair struct{ from, to string }
	pairs := map[pair]struct{}{}
	for i, edge := range req.Edges {
		p := fmt.Sprintf("edges[%d]", i)
		if !validIdent(edge.ID) {
			ve.Add(p+".id", "must start with a letter and contain only letters, digits, underscores, or hyphens")
		} else if _, ok := edges[edge.ID]; ok {
			ve.Add(p+".id", "duplicate edge id "+edge.ID)
		} else {
			edges[edge.ID] = struct{}{}
		}
		if _, ok := steps[edge.From]; edge.From == "" || !ok {
			ve.Add(p+".from", "must reference an existing step")
		}
		if _, ok := steps[edge.To]; edge.To == "" || !ok {
			ve.Add(p+".to", "must reference an existing step")
		}
		if _, ok := edgeKinds[edge.Kind]; !ok {
			ve.Add(p+".kind", `must be one of "next", "condition", "error", "async"`)
		}
		if len(edge.Label) > MaxLabel {
			ve.Add(p+".label", fmt.Sprintf("exceeds %d bytes", MaxLabel))
		}
		if edge.From != "" && edge.To != "" {
			pairs[pair{edge.From, edge.To}] = struct{}{}
		}
	}

	seenScenarios := map[string]struct{}{}
	for i, sc := range req.Scenarios {
		p := fmt.Sprintf("scenarios[%d]", i)
		if !validIdent(sc.ID) {
			ve.Add(p+".id", "must start with a letter and contain only letters, digits, underscores, or hyphens")
		} else if _, ok := seenScenarios[sc.ID]; ok {
			ve.Add(p+".id", "duplicate scenario id "+sc.ID)
		} else {
			seenScenarios[sc.ID] = struct{}{}
		}
		requireText(ve, p+".name", sc.Name, MaxName)
		if len(sc.Description) > MaxSummary {
			ve.Add(p+".description", fmt.Sprintf("exceeds %d bytes", MaxSummary))
		}
		if len(sc.Path) == 0 {
			ve.Add(p+".path", "must contain at least one step")
		}
		if len(sc.Path) > MaxPathSteps {
			ve.Add(p+".path", fmt.Sprintf("at most %d steps", MaxPathSteps))
		}
		pathSet := map[string]struct{}{}
		for j, stepID := range sc.Path {
			sp := fmt.Sprintf("%s.path[%d]", p, j)
			if _, ok := steps[stepID]; !ok {
				ve.Add(sp, "must reference an existing step")
				continue
			}
			if _, ok := pathSet[stepID]; ok {
				ve.Add(sp, "duplicate step "+stepID)
			}
			pathSet[stepID] = struct{}{}
			if j > 0 {
				prev := sc.Path[j-1]
				if _, ok := pairs[pair{prev, stepID}]; !ok {
					ve.Add(sp, fmt.Sprintf("no edge from %s to %s", prev, stepID))
				}
			}
		}
		if len(sc.Examples) != len(pathSet) && len(sc.Path) > 0 {
			ve.Add(p+".examples", "must contain exactly one example for each step on the path")
		}
		seenExamples := map[string]struct{}{}
		for j, ex := range sc.Examples {
			ep := fmt.Sprintf("%s.examples[%d]", p, j)
			if _, ok := pathSet[ex.StepID]; !ok {
				ve.Add(ep+".stepId", "must be a step on this scenario path")
			} else if _, ok := seenExamples[ex.StepID]; ok {
				ve.Add(ep+".stepId", "duplicate example for step "+ex.StepID)
			} else {
				seenExamples[ex.StepID] = struct{}{}
			}
			if ex.Kind != "synthetic" {
				ve.Add(ep+".kind", `must be "synthetic"; observed examples are not accepted`)
			}
			validateValue(ve, ep+".input", ex.Input)
			validateValue(ve, ep+".output", ex.Output)
			requireText(ve, ep+".transformation", ex.Transformation, MaxTransformation)
			if len(ex.FieldChanges) > 32 {
				ve.Add(ep+".fieldChanges", "at most 32 field changes")
			}
			for k, ch := range ex.FieldChanges {
				cp := fmt.Sprintf("%s.fieldChanges[%d]", ep, k)
				requireText(ve, cp+".field", ch.Field, MaxField)
				requireText(ve, cp+".why", ch.Why, MaxWhy)
				if len(ch.Before) > MaxFieldValue {
					ve.Add(cp+".before", fmt.Sprintf("exceeds %d bytes", MaxFieldValue))
				}
				if len(ch.After) > MaxFieldValue {
					ve.Add(cp+".after", fmt.Sprintf("exceeds %d bytes", MaxFieldValue))
				}
			}
		}
	}

	if len(ve.Details) == 0 {
		return nil
	}
	return ve
}

func validateOverview(ve *ValidationError, overview CreateOverview, steps map[string]struct{}) {
	seq := overview.Sequence
	if len(seq.Participants) < MinParticipants {
		ve.Add("overview.sequence.participants", fmt.Sprintf("at least %d participants are required", MinParticipants))
	}
	if len(seq.Participants) > MaxParticipants {
		ve.Add("overview.sequence.participants", fmt.Sprintf("at most %d participants", MaxParticipants))
	}
	if len(seq.Messages) == 0 {
		ve.Add("overview.sequence.messages", "at least one message is required")
	}
	if len(seq.Messages) > MaxSequenceMessages {
		ve.Add("overview.sequence.messages", fmt.Sprintf("at most %d messages", MaxSequenceMessages))
	}

	participants := make(map[string]struct{}, len(seq.Participants))
	for i, p := range seq.Participants {
		path := fmt.Sprintf("overview.sequence.participants[%d]", i)
		if !validIdent(p.ID) {
			ve.Add(path+".id", "must start with a letter and contain only letters, digits, underscores, or hyphens")
		} else if _, ok := participants[p.ID]; ok {
			ve.Add(path+".id", "duplicate participant id "+p.ID)
		} else {
			participants[p.ID] = struct{}{}
		}
		requireText(ve, path+".label", p.Label, MaxLabel)
	}

	messages := make(map[string]struct{}, len(seq.Messages))
	for i, m := range seq.Messages {
		path := fmt.Sprintf("overview.sequence.messages[%d]", i)
		if !validIdent(m.ID) {
			ve.Add(path+".id", "must start with a letter and contain only letters, digits, underscores, or hyphens")
		} else if _, ok := messages[m.ID]; ok {
			ve.Add(path+".id", "duplicate message id "+m.ID)
		} else {
			messages[m.ID] = struct{}{}
		}
		if _, ok := participants[m.From]; m.From == "" || !ok {
			ve.Add(path+".from", "must reference an existing participant")
		}
		if _, ok := participants[m.To]; m.To == "" || !ok {
			ve.Add(path+".to", "must reference an existing participant")
		}
		if _, ok := messageKinds[m.Kind]; !ok {
			ve.Add(path+".kind", `must be one of "call", "return", "note"`)
		} else if m.Kind == "note" && m.From != m.To {
			ve.Add(path, "note messages must use the same from and to participant")
		} else if m.Kind != "note" && m.From != "" && m.To != "" && m.From == m.To {
			ve.Add(path, "call and return messages must connect two different participants")
		}
		requireText(ve, path+".label", m.Label, MaxLabel)
		if m.StepID != "" {
			if _, ok := steps[m.StepID]; !ok {
				ve.Add(path+".stepId", "must reference an existing step")
			}
		}
	}

	if overview.Erd != nil {
		validateErd(ve, *overview.Erd)
	}
}

func validateErd(ve *ValidationError, erd CreateErd) {
	if len(erd.Entities) < MinErdEntities {
		ve.Add("overview.erd.entities", "at least one entity is required")
	}
	if len(erd.Entities) > MaxErdEntities {
		ve.Add("overview.erd.entities", fmt.Sprintf("at most %d entities", MaxErdEntities))
	}
	if len(erd.Relationships) > MaxErdRelationships {
		ve.Add("overview.erd.relationships", fmt.Sprintf("at most %d relationships", MaxErdRelationships))
	}

	entities := make(map[string]struct{}, len(erd.Entities))
	for i, ent := range erd.Entities {
		path := fmt.Sprintf("overview.erd.entities[%d]", i)
		if !validIdent(ent.ID) {
			ve.Add(path+".id", "must start with a letter and contain only letters, digits, underscores, or hyphens")
		} else if _, ok := entities[ent.ID]; ok {
			ve.Add(path+".id", "duplicate entity id "+ent.ID)
		} else {
			entities[ent.ID] = struct{}{}
		}
		requireText(ve, path+".label", ent.Label, MaxLabel)
		if len(ent.Attributes) > MaxErdAttributes {
			ve.Add(path+".attributes", fmt.Sprintf("at most %d attributes", MaxErdAttributes))
		}
		for j, attr := range ent.Attributes {
			ap := fmt.Sprintf("%s.attributes[%d]", path, j)
			requireText(ve, ap+".name", attr.Name, MaxField)
			if len(attr.Type) > MaxField {
				ve.Add(ap+".type", fmt.Sprintf("exceeds %d bytes", MaxField))
			}
			if attr.Key != "" {
				if _, ok := erdAttrKeys[attr.Key]; !ok {
					ve.Add(ap+".key", `must be one of "pk", "fk", "unique"`)
				}
			}
		}
	}

	rels := make(map[string]struct{}, len(erd.Relationships))
	for i, rel := range erd.Relationships {
		path := fmt.Sprintf("overview.erd.relationships[%d]", i)
		if !validIdent(rel.ID) {
			ve.Add(path+".id", "must start with a letter and contain only letters, digits, underscores, or hyphens")
		} else if _, ok := rels[rel.ID]; ok {
			ve.Add(path+".id", "duplicate relationship id "+rel.ID)
		} else {
			rels[rel.ID] = struct{}{}
		}
		if _, ok := entities[rel.From]; rel.From == "" || !ok {
			ve.Add(path+".from", "must reference an existing entity")
		}
		if _, ok := entities[rel.To]; rel.To == "" || !ok {
			ve.Add(path+".to", "must reference an existing entity")
		}
		if _, ok := erdRelKinds[rel.Kind]; !ok {
			ve.Add(path+".kind", `must be one of "one_to_one", "one_to_many", "many_to_one", "many_to_many"`)
		}
		if len(rel.Label) > MaxLabel {
			ve.Add(path+".label", fmt.Sprintf("exceeds %d bytes", MaxLabel))
		}
	}
}

func validateValue(ve *ValidationError, path string, v ExampleValue) {
	if _, ok := exampleFormat[v.Format]; !ok {
		ve.Add(path+".format", `must be one of "json", "go", "http", "text"`)
	}
	if strings.TrimSpace(v.Content) == "" {
		ve.Add(path+".content", "must not be empty")
	} else if len(v.Content) > MaxExampleContent {
		ve.Add(path+".content", fmt.Sprintf("exceeds %d bytes", MaxExampleContent))
	}
}

func requireText(ve *ValidationError, path, value string, max int) {
	if strings.TrimSpace(value) == "" {
		ve.Add(path, "must not be empty")
		return
	}
	if len(value) > max {
		ve.Add(path, fmt.Sprintf("exceeds %d bytes", max))
	}
}

func validIdent(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for i, r := range id {
		if !utf8.ValidRune(r) {
			return false
		}
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case i > 0 && (r >= '0' && r <= '9' || r == '_' || r == '-'):
		default:
			return false
		}
	}
	return true
}

// ValidateRelPath rejects absolute paths, parent segments, and other forms
// that could escape the configured repository.
func ValidateRelPath(p string) error {
	if p == "" {
		return fmt.Errorf("path is required")
	}
	if len(p) > MaxPathLen {
		return fmt.Errorf("path exceeds %d bytes", MaxPathLen)
	}
	if strings.ContainsRune(p, 0) {
		return fmt.Errorf("path contains a NUL byte")
	}
	if strings.Contains(p, "\\") {
		return fmt.Errorf("path must use forward slashes")
	}
	if strings.Contains(p, ":") {
		return fmt.Errorf("path must not contain a drive prefix")
	}
	if strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~") {
		return fmt.Errorf("path must be repository-relative")
	}
	parts := strings.Split(p, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("path must not contain empty, current, or parent segments")
		}
	}
	return nil
}
