# Architecture

`ms-go-validation-orchestrator` owns coordination, not technology-specific validation. It parses a validation request, chooses the V1 or transitional legacy path, filters and orders stages, invokes configured engine adapters, evaluates simple links, and returns one normalized result.

## Boundaries

- `internal/domain` defines contracts, requests, results, issues, and engine ports.
- `internal/usecase` owns parsing, legacy adaptation, stage selection and ordering, execution, aggregation, and links.
- `internal/infrastructure/http` maps orchestration inputs to validator-specific HTTP payloads and normalizes responses.
- `internal/transport/http`, `dto`, `mapper`, and `public` implement the inbound API and reusable client.
- `internal/app`, `config`, and `cmd` form the composition root and register engines from environment configuration.

Engine services own HTML, CSS, language, framework, browser, Git, Docker, database, Linux, cache, and search validation. Adding such validation directly to this repository would violate the service boundary; add an adapter here and the implementation in the owning validator.

## Execution flow

1. Authenticate `POST /api/v1/validate` with `X-Internal-Token`, enforce the 2 MiB body limit, reject unknown transport fields, and map its DTO.
2. Parse `code_structure`. Any V1 envelope field makes the payload a V1 candidate; malformed or incomplete V1 is rejected instead of falling back to legacy. Core V1 fields are strict while engine-owned `rules` and `checks` remain opaque objects.
3. Filter stages by request mode. With an empty request mode, no filtering occurs.
4. Validate retained stage identities and dependencies, then topologically sort them. Ready stages are ordered by ID for deterministic execution.
5. Execute stages one at a time. A failed dependency skips its dependent stage.
6. Retain only links whose dependencies remain after stage filtering, evaluate them in authored order, and aggregate reports and errors.
7. Build additive `teacher-validation-explanation.v1` from the completed
   normalized result. This presentation projection cannot read `RawResult`,
   cannot change `passed`, and contains only bounded blocking issues.

## Trust boundaries

Every `/api/v1/*` endpoint requires `X-Internal-Token`. Inbound bodies are limited to 2 MiB; downstream engine and exported public-client responses are limited to 4 MiB. The exported client supplies a 30-second default timeout when its caller passes a nil HTTP client.

`workspace.root_path` is accepted only in the canonical portable form `/workspaces/<sandbox-id>` and is then forwarded to validators. This prevents callers from selecting arbitrary host paths, but it does not replace sandbox/container isolation, symlink controls, resource limits, or network policy.

Authoring clients use `GET /api/v1/capabilities` and `POST /api/v1/contracts/inspect` before execution. A task revision can become runnable only after `POST /api/v1/contracts/verify` has executed the final contract in distinct starter, reference, and negative-fixture sandboxes and returned a passing receipt.

Validation owns construction of the safe explanation shape, but it does not
own result persistence or workspace revisions. Practice must persist final
results before exposing them to Teacher. A live Sandbox consumer must bind the
same safe projection to a persisted exact workspace revision and timestamp;
browser/WS state alone is not authoritative.

See [Implementation limits](implementation-limits.md) before changing contract authoring or promising validation guarantees.
