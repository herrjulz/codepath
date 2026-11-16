package reports

const (
	SchemaVersion = 1

	MaxBodyBytes       = 1 << 20
	MaxSteps             = 64
	MaxEdges             = 128
	MaxScenarios         = 16
	MaxPathSteps         = 64
	MaxRefsPerStep       = 8
	MaxSpanLines         = 400
	MaxTitle             = 200
	MaxSummary           = 2_000
	MaxStepSummary       = 500
	MaxExplanation       = 8_000
	MaxExampleContent    = 16 << 10
	MaxTransformation    = 4_000
	MaxNotes             = 10
	MaxNoteLen           = 1_000
	MaxLabel             = 200
	MaxName              = 200
	MaxField             = 200
	MaxFieldValue        = 2_000
	MaxWhy               = 2_000
	MaxPathLen           = 240
	MaxFileBytes         = 1 << 20
	ContextRadius        = 20
	MaxValidationItems   = 40
	MinParticipants      = 2
	MaxParticipants      = 16
	MaxSequenceMessages  = 64
	MinErdEntities       = 1
	MaxErdEntities       = 16
	MaxErdAttributes     = 16
	MaxErdRelationships  = 32
)
