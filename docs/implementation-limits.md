# Implementation Limits

This document records behavior visible in the current code. It is a constraint list and backlog input, not a promise that the listed gaps are already planned.

## Contract validation

- V1 detection checks only positive `version`, non-empty `kind`, and at least one stage. Supported versions, kinds, profiles, modes, targets, rules, checks, and unknown fields are not strictly validated.
- Stage ID/engine presence, duplicate stage IDs, missing dependencies, and cycles are checked only after request-mode filtering.
- Link IDs and kinds are not prevalidated; duplicate or empty IDs are accepted until evaluation. Missing or empty substring values can produce misleading success.
- `workspace.required_files` is not enforced by the orchestrator.
- A payload that does not meet the minimal V1 test is adapted to `legacy.generic`. That engine always fails with `LEGACY_CONTRACT_NOT_MIGRATED`; old execution is not implemented here.

## Modes, ordering, and time

- Stages run sequentially in deterministic topological order; there is no parallel execution.
- Only `ts.runtime`, `java.runtime`, `kotlin.runtime`, and `browser.runtime` are hard final-only. Other runtime naming is an authoring convention unless the downstream engine enforces it.
- An empty request mode runs all stages, including final-only stages. An unsupported non-empty request mode can leave only stages authored with empty mode or `both`.
- Dependencies are validated after filtering. A retained stage that depends on a filtered stage makes the request invalid.
- `timeout_seconds` is included in downstream payloads where supported but does not create a per-stage context deadline. The server and outbound HTTP client each use 30-second timeouts, which are not an overall orchestration budget.

## Aggregation and links

- A normal validation failure from an optional stage does not fail the aggregate. An engine or transport execution error always sets the aggregate to failed, even when that stage is optional.
- Stage errors are copied into the top-level error list; execution failures can therefore include a wrapper error and stage-level errors together.
- Links whose stage dependencies were removed by mode filtering are silently omitted.
- `workspace.file_contains` and `workspace.selector_exists` both use literal `strings.Contains`; selector checks do not parse HTML or CSS selectors.

## HTTP and adapters

- Inbound JSON and outbound responses have no explicit body-size limits. Unknown inbound JSON fields are accepted.
- The exported public client has no default timeout when constructed with a nil HTTP client.
- Common validator responses are considered passed when any of `ok`, `isValid`, or `valid` is true; inconsistent response fields are not rejected.
- Engine IDs are stored in a map and a later duplicate registration silently replaces the earlier client.
- `workspace.root_path` is passed through without path-containment validation.

Changes to any item above require focused tests and synchronized updates to the contract, result, engine, authoring, and capability documentation as applicable.
