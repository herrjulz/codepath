import { z } from "zod";

export const exampleValueSchema = z.object({
  format: z.enum(["json", "go", "http", "text"]),
  content: z.string(),
});

export const fieldChangeSchema = z.object({
  field: z.string(),
  before: z.string().optional(),
  after: z.string().optional(),
  why: z.string(),
});

export const stepExampleSchema = z.object({
  stepId: z.string(),
  kind: z.literal("synthetic"),
  input: exampleValueSchema,
  transformation: z.string(),
  output: exampleValueSchema,
  fieldChanges: z.array(fieldChangeSchema).optional(),
});

export const codeRefSchema = z.object({
  path: z.string(),
  startLine: z.number(),
  endLine: z.number(),
  label: z.string().optional(),
  fileSha256: z.string().optional(),
});

export const flowStepSchema = z.object({
  id: z.string(),
  title: z.string(),
  summary: z.string(),
  explanation: z.string(),
  codeRefs: z.array(codeRefSchema),
  notes: z.array(z.string()).optional(),
});

export const flowEdgeSchema = z.object({
  id: z.string(),
  from: z.string(),
  to: z.string(),
  label: z.string().optional(),
  kind: z.enum(["next", "condition", "error", "async"]),
});

export const scenarioSchema = z.object({
  id: z.string(),
  name: z.string(),
  description: z.string().optional(),
  path: z.array(z.string()),
  examples: z.array(stepExampleSchema),
});

export const participantSchema = z.object({
  id: z.string(),
  label: z.string(),
});

export const sequenceMessageSchema = z.object({
  id: z.string(),
  from: z.string(),
  to: z.string(),
  label: z.string(),
  kind: z.enum(["call", "return", "note"]),
  stepId: z.string().optional(),
});

export const sequenceDiagramSchema = z.object({
  participants: z.array(participantSchema),
  messages: z.array(sequenceMessageSchema),
});

export const erdAttributeSchema = z.object({
  name: z.string(),
  type: z.string().optional(),
  key: z.enum(["pk", "fk", "unique"]).optional(),
});

export const erdEntitySchema = z.object({
  id: z.string(),
  label: z.string(),
  attributes: z.array(erdAttributeSchema).optional(),
});

export const erdRelationshipSchema = z.object({
  id: z.string(),
  from: z.string(),
  to: z.string(),
  label: z.string().optional(),
  kind: z.enum(["one_to_one", "one_to_many", "many_to_one", "many_to_many"]),
});

export const erdDiagramSchema = z.object({
  entities: z.array(erdEntitySchema),
  relationships: z.array(erdRelationshipSchema),
});

export const overviewSchema = z.object({
  sequence: sequenceDiagramSchema,
  erd: erdDiagramSchema.optional(),
});

export const reportSchema = z.object({
  schemaVersion: z.literal(1),
  id: z.string(),
  title: z.string(),
  summary: z.string(),
  language: z.literal("go"),
  repository: z.object({
    displayName: z.string(),
    revision: z.string().optional(),
  }),
  createdAt: z.string(),
  overview: overviewSchema,
  steps: z.array(flowStepSchema),
  edges: z.array(flowEdgeSchema),
  scenarios: z.array(scenarioSchema),
});

export const summarySchema = z.object({
  id: z.string(),
  title: z.string(),
  summary: z.string(),
  language: z.string(),
  repository: z.string(),
  revision: z.string().optional(),
  createdAt: z.string(),
  url: z.string(),
});

export const sourceSchema = z.object({
  path: z.string(),
  label: z.string().optional(),
  startLine: z.number(),
  endLine: z.number(),
  content: z.string(),
  contextStart: z.number(),
  contextEnd: z.number(),
  context: z.string(),
  language: z.string(),
  storedSha256: z.string().optional(),
  currentSha256: z.string().optional(),
  stale: z.boolean(),
  missing: z.boolean(),
  redacted: z.boolean(),
  message: z.string().optional(),
});

export type WalkthroughReport = z.infer<typeof reportSchema>;
export type FlowStep = z.infer<typeof flowStepSchema>;
export type FlowEdge = z.infer<typeof flowEdgeSchema>;
export type Scenario = z.infer<typeof scenarioSchema>;
export type StepExample = z.infer<typeof stepExampleSchema>;
export type SourceView = z.infer<typeof sourceSchema>;
export type ReportSummary = z.infer<typeof summarySchema>;
export type CodeRef = z.infer<typeof codeRefSchema>;
export type Overview = z.infer<typeof overviewSchema>;
export type SequenceDiagram = z.infer<typeof sequenceDiagramSchema>;
export type Participant = z.infer<typeof participantSchema>;
export type SequenceMessage = z.infer<typeof sequenceMessageSchema>;
export type ErdDiagram = z.infer<typeof erdDiagramSchema>;
export type ErdEntity = z.infer<typeof erdEntitySchema>;
export type ErdRelationship = z.infer<typeof erdRelationshipSchema>;
