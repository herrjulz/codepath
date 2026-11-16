# Create a CodePath walkthrough

Use this when asked to explain a flow in the current repository and produce a CodePath report.

1. Understand the requested flow with your existing file-search and file-reading tools. Do not guess paths or line numbers.
2. Identify and verify exact repository-relative Go file paths and inclusive 1-based line ranges. Use forward slashes. Do not use absolute paths or `..`.
3. Build a coherent graph of the important success and failure branches. Step, edge, and scenario ids must be unique and start with a letter.
4. Author `overview.sequence`: participants (actors left to right) and ordered messages (`call`, `return`, or `note`). Link messages to step ids with `stepId` when a click should open that step. Notes must use the same `from` and `to` participant.
4b. Optionally author `overview.erd` when data shapes matter (structs, records, tokens). Include entities with useful attributes and relationships (`one_to_one`, `one_to_many`, `many_to_one`, `many_to_many`). Skip the ERD for pure control-flow walkthroughs.
5. Write short, precise explanations that refer to the source you read. Keep explanation text separate from the code references.
6. Pick one or more realistic synthetic scenarios, such as success and a specific failure. Do not claim that the program was executed.
7. For every step on a scenario path, provide `input`, `transformation`, and `output`. Keep ids and example values consistent across adjacent steps. Set `kind` to `synthetic`.
8. Never copy secrets, environment values, passwords, tokens, or user data from the repository. Use fake emails, dummy ids, and obvious demo values such as `demo-only`.
9. Call `create_walkthrough`. If it returns validation errors, correct the named field and submit again.
10. Return the local report URL to the user so they can open the walkthrough. The report opens on a Summary page with the sequence diagram before the detailed walkthrough.

The repository root is configured on the CodePath MCP process. Do not send a repository path in the tool arguments. `language` must be `go` and `schemaVersion` must be `1`. `overview.sequence` is required. `overview.erd` is optional.
