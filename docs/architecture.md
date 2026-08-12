# Architecture

`ms-go-validation-orchestrator` owns coordination, not technology-specific validation. It parses a validation request, chooses the V1 or transitional legacy path, filters and orders stages, invokes configured engine adapters, evaluates simple links, and returns one normalized result.

## Boundaries

- `internal/domain` defines contracts, requests, results, issues, and engine ports.
- `internal/usecase` owns parsing, legacy adaptation, stage selection and ordering, execution, aggregation, and links.
- `internal/adapters/engines` maps orchestration inputs to validator-specific HTTP payloads and normalizes responses.
- `transport/http`, `dto`, `mapper`, and `public` implement the inbound API and reusable client.
- `internal/app`, `config`, and `cmd` form the composition root and register engines from environment configuration.

Engine services own HTML, CSS, language, framework, browser, Git, Docker, database, Linux, cache, and search validation. Adding such validation directly to this repository would violate the service boundary; add an adapter here and the implementation in the owning validator.

## Execution flow

1. Decode `POST /api/v1/validate` and map its DTO.
2. Parse `code_structure`. A payload with positive `version`, non-empty `kind`, and at least one stage is treated as the new format; other JSON is sent to the legacy adapter.
3. Filter stages by request mode. With an empty request mode, no filtering occurs.
4. Validate retained stage identities and dependencies, then topologically sort them. Ready stages are ordered by ID for deterministic execution.
5. Execute stages one at a time. A failed dependency skips its dependent stage.
6. Retain only links whose dependencies remain after stage filtering, evaluate them in authored order, and aggregate reports and errors.

## Trust boundaries

The HTTP API currently has no authentication, authorization, request-body limit, or strict unknown-field rejection. Outbound engine clients have a 30-second HTTP timeout, but response bodies are unbounded. The public client has no default timeout when the caller supplies `nil`.

`workspace.root_path` is forwarded to validators without containment checks. Deployments must establish the trusted shared `/workspaces/<sandbox-id>` namespace and must not accept arbitrary host paths from untrusted callers.

See [Implementation limits](implementation-limits.md) before changing contract authoring or promising validation guarantees.
