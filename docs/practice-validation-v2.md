# Practice Validation V2

This endpoint implements the Validation Orchestrator side of the frozen
[Practice Runtime V2 contract](../../ms-go-practice-runtime/contracts/practice-runtime-v2.md).
It evaluates immutable V1 stage contracts for one TaskInstance submission and
does not persist official results or modify Practice or Student state.

## Request and response

`POST /api/v2/practice-validations` requires `X-Internal-Token`. JSON is capped
at 2 MiB and rejects unknown fields and trailing JSON. Invalid command
envelopes return `400`; oversized bodies return `413`. A completed evaluation,
including an evaluation `ERROR`, returns `200` with
`practice-validation-result.v2`.

The request carries the TaskInstance, Submission, ValidationRun, revision
digest, validation-contract digest, scope, the exact Sandbox snapshot
reference, and a `practice-validation-specification.v1`. The specification
contains the ordered executable contracts for that scope. Review policy,
mentor rubrics, AI review contracts, and unrelated private context are not
accepted by the strict schema.

The result echoes every correlation identity and the full snapshot reference.
`outcome` is `PASS`, `FAIL`, or `ERROR`. Stage summaries retain execution
order and contain only `stage_id`, `required`, `outcome`, and bounded
`{code,message}` issues. Link criteria are represented with a `link:` prefix in
`stage_id`. Raw validator results, evidence, selectors, routes, symbols,
properties, commands, fixtures, and private evaluation assets are not returned.

## Digest and snapshot checks

Validation recomputes `practice-validation-contract-digest.v1` over canonical
JSON containing the schema, scope, optional milestone ID, and complete
validation specification. Object keys are sorted recursively, arrays retain
their order, insignificant whitespace is removed, and the UTF-8 bytes are
SHA-256 hashed.

Workspace content is fetched only from Sandbox's exact pinned snapshot route:

```text
GET /internal/v2/sandboxes/{sandboxID}/snapshots/{snapshotID}
    ?generation_id={generationID}&workspace_digest={workspaceDigest}
```

The reader sends `X-Internal-Token`, verifies all returned identities and the
`practice-pinned-snapshot-content.v2` envelope, and recomputes the workspace
digest as SHA-256 over JSON-encoded `{path,content}` files sorted by path. A
missing pin, corrupt response, identity mismatch, or digest mismatch becomes
an `ERROR`; it never reads the current workspace head.

The retained `revision` must be present and a non-negative integer. Explicit
revision `0` is a valid bootstrap generation. Missing, null, negative or
malformed revisions are corrupt responses; accepting genesis does not relax
identity, pin metadata, file-count or digest checks.

## Provider input prerequisites

Official validation passes the verified snapshot's `{path,content}` files to
engines without a local `root_path`. HTML consumes these files directly.
The outbound Foundation adapter rejects missing root prerequisites for
`git.core` and the dedicated HTTP runtime engines (`http.runtime`,
`python.django.runtime`, `go.gin.runtime`, `go.echo.runtime`,
`php.laravel.runtime`, `php.symfony.runtime`, `php.yii2.runtime`, and
`php.yii3.runtime`) before contacting those providers. Such an execution
failure becomes `ERROR` with `UNSUPPORTED_CONFIGURATION`, including when
the affected stage is optional. It does not become a learner semantic failure.

The HTTP dispatcher applies this prerequisite only when it selects the
dedicated provider; its Node branch continues accepting inline files.
V1 requests carrying their existing portable root continue forwarding that
root and their opaque rules/checks. No host root or current workspace is
substituted into official validation.

Linux can materialize inline text files and inferred parent directories, so
it is not rejected merely for lacking a root. This text snapshot does not
represent executable modes, independently existing empty directories, Git
history or a reproducible execution environment. Faithful validation of
those richer artifact criteria remains blocked on a reviewed artifact and
prerequisite contract; this scoped implementation does not certify them.

## Outcome rules

- Required semantic criteria all pass: `PASS`.
- A required semantic criterion fails: `FAIL`.
- Optional semantic failure does not block `PASS`.
- A zero-required-criteria contract, malformed immutable contract, or digest
  mismatch returns `ERROR` with `AUTHORING_DEFECT`.
- Every stage execution error, including an optional stage, is `ERROR`.
- Stage `timeout_seconds` is enforced with a per-stage context deadline.
- Validator timeout, unavailable transport/5xx, malformed validator response,
  unsupported engine configuration, and Sandbox failures remain `ERROR`, never
  learner `FAIL`.

Stable error classifications are `VALIDATOR_TIMEOUT`,
`VALIDATOR_UNAVAILABLE`, `VALIDATOR_PROTOCOL`, `STAGE_EXECUTION_ERROR`,
`SNAPSHOT_UNAVAILABLE`, `SNAPSHOT_CORRUPT`, `SNAPSHOT_DIGEST_MISMATCH`,
`UNSUPPORTED_CONFIGURATION`, and `AUTHORING_DEFECT`.

## Configuration

`SANDBOX_SERVICE_BASE_URL` selects the Sandbox base URL, and
`SANDBOX_SERVICE_INTERNAL_TOKEN` supplies its internal service token. If either
is unset, the endpoint returns `ERROR` with `SNAPSHOT_UNAVAILABLE` after
validating the request and contract digest.
