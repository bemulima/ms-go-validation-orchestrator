# Repository Guidelines

## Agent bootstrap

Read `.ai/rules/common.md`, `.ai/service.yaml`, `docs/README.md`, and the affected contracts before changing files. Code, tests, examples, and repository-owned documentation are authoritative.

## Architecture invariants

- Domain types and ports live in `internal/domain`; orchestration belongs in `internal/usecase`; outbound engine HTTP integrations belong in `internal/infrastructure/http`; inbound HTTP transport and wiring stay at the edges in `internal/transport/http` and `internal/app`.
- This service coordinates validators. It must not absorb language-, framework-, browser-, database-, or infrastructure-specific validation logic.
- `ValidationContractV1`, normalized results, stage ordering/mode filtering, engine IDs, link semantics, and outbound validator payloads are versioned contracts.
- V1 stages execute sequentially in deterministic topological order. V1 core fields are parsed strictly, but engine-specific `rules` and `checks` remain opaque JSON owned by engines. V1 forwards `timeout_seconds` without enforcing a per-stage context deadline.
- `/api/v1/*` and `/api/v2/*` require `X-Internal-Token`; request bodies are capped at 2 MiB and engine/Sandbox responses at 4 MiB. V1 root-backed workspaces must use the exact portable `/workspaces/<sandbox-id>` namespace.
- New authoring must inspect configured capabilities and obtain a passing executable verification receipt. Inspection alone is not proof that fixtures behave as intended.
- Legacy payloads are adapted to `legacy.generic`, but that engine intentionally returns `LEGACY_CONTRACT_NOT_MIGRATED`; adaptation is not successful legacy execution.
- `workspace.selector_exists` and `workspace.file_contains` currently use literal substring matching. `workspace.required_files` is declared but not enforced here.
- Preserve V1's optional-stage edge case in documentation: semantic validation failure can be optional, but an engine/transport execution error fails the V1 aggregate. Practice V2 treats any stage execution error as typed `ERROR` and treats optional semantic failure as non-blocking.

## Verification and delivery

- Use `.ai/commands.yaml`; run policy, tracked-file `gofmt`, `go vet ./...`, `go test ./... -count=1`, and build verification.
- Contract, mode/filtering, dependency, aggregation, adapter normalization, or engine-registration changes require focused tests and synchronized repository docs/examples.
- Do not start Docker/Foundation stacks, contact validators, create networks, or run commands marked `requires_approval: true` without approval.
