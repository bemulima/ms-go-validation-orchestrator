# Repository Guidelines

## Agent bootstrap

Read `.ai/rules/common.md`, `.ai/service.yaml`, `docs/README.md`, and the affected contracts before changing files. Code, tests, examples, and repository-owned documentation are authoritative.

## Architecture invariants

- Domain types and ports live in `internal/domain`; orchestration belongs in `internal/usecase`; engine integrations belong in `internal/adapters/engines`; HTTP transport and wiring stay at the edges.
- This service coordinates validators. It must not absorb language-, framework-, browser-, database-, or infrastructure-specific validation logic.
- `ValidationContractV1`, normalized results, stage ordering/mode filtering, engine IDs, link semantics, and outbound validator payloads are versioned contracts.
- Stages execute sequentially in deterministic topological order. Do not claim parallelism, per-stage timeout enforcement, strict contract-schema validation, authentication, or workspace path containment unless the implementation adds and tests them.
- Legacy payloads are adapted to `legacy.generic`, but that engine intentionally returns `LEGACY_CONTRACT_NOT_MIGRATED`; adaptation is not successful legacy execution.
- `workspace.selector_exists` and `workspace.file_contains` currently use literal substring matching. `workspace.required_files` is declared but not enforced here.
- Preserve the current optional-stage edge case in documentation: validation failure can be optional, but an engine/transport execution error currently fails the aggregate result.

## Verification and delivery

- Use `.ai/commands.yaml`; run policy, tracked-file `gofmt`, `go vet ./...`, `go test ./... -count=1`, and build verification.
- Contract, mode/filtering, dependency, aggregation, adapter normalization, or engine-registration changes require focused tests and synchronized repository docs/examples.
- Do not start Docker/Foundation stacks, contact validators, create networks, or run commands marked `requires_approval: true` without approval.
