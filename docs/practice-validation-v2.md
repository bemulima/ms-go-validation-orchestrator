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
reference, and a versioned validation specification. Specification V1
contains the ordered executable contracts for that scope. Specification V2
contains one original full contract for FINAL without a milestone identity.
Review policy,
mentor rubrics, AI review contracts, and unrelated private context are not
accepted by the strict schema.

The result echoes every correlation identity and the full snapshot reference.
`outcome` is `PASS`, `FAIL`, or `ERROR`. Stage summaries retain execution
order and contain only `stage_id`, `required`, `outcome`, and bounded
`{code,message}` issues. Link criteria are represented with a `link:` prefix in
`stage_id`. Raw validator results, evidence, selectors, routes, symbols,
properties, commands, fixtures, and private evaluation assets are not returned.

## Specification versions

`practice-validation-specification.v1` remains unchanged for existing
MILESTONE and FINAL callers. Every `contracts[]` entry retains its authored
`milestone_id`, explicit `required` flag, V1 `validation_contract` and
`evaluation_assets`. MILESTONE requires exactly one contract whose ID matches
the requested milestone. Existing FINAL callers may continue sending their
ordered authored contract array.

`practice-validation-specification.v2` is an additive internal extension for
an imported task whose original full contract has no authored milestone ID:

```json
{
  "schema": "practice-validation-specification.v2",
  "runtime_profile": {},
  "final_contract": {
    "required": true,
    "validation_contract": {
      "version": 1,
      "kind": "workspace_contract",
      "stages": [
        {
          "id": "html",
          "engine": "html.dom",
          "mode": "final",
          "rules": {
            "version": 2,
            "head": {"doctype": true, "htmlLang": "en"},
            "body": {"roots": [{"tag": {"value": "h1"}, "text": {"value": "Hello World!"}}]}
          }
        }
      ]
    },
    "evaluation_assets": {}
  }
}
```

The example illustrates the envelope; the producer supplies its complete
immutable original `validation_contract`, including all authored workspace
requirements, stages, dependencies and links. It must not partition the full
contract, borrow a teaching-step ID, or synthesize a milestone. The existing
FINAL mode filter and deterministic dependency ordering execute that contract
once against the exact pinned snapshot. Original stage IDs and `link:` IDs
remain the only criterion identities in the safe result.

V2 accepts only outer `scope: "FINAL"`, with outer `milestone_id` omitted.
`contracts` and `milestone_id` are not fields of the V2 specification;
`final_contract` likewise has no milestone field. Unknown fields, including
mixed V1/V2 fields, are rejected as invalid requests. The final contract,
specification and evaluation-assets objects also reject duplicate members and
case aliases; opaque engine-rule and runtime-profile members retain their
existing interpretation. The final contract,
`required: true`, an object-valued `validation_contract` and an object-valued
`evaluation_assets` are mandatory; absent or invalid immutable values return
`ERROR/AUTHORING_DEFECT`. `runtime_profile` must remain a JSON object. An
optional stage or link retains its normal behavior, but the contract as a
whole cannot be optional and cannot have zero required final criteria.

V2 uses the same private evaluation-assets schema and provider prerequisites
as V1. Empty `evaluation_assets: {}` is valid for engines without private
asset requirements, including `browser.runtime`. It does not supply the
frozen Go test bundle/runner/fixture binding required by `go.tests.v1`, nor an
execution root for providers that require one. No fixture, suite reference or
runner binding is inferred. Configured capabilities and passing executable
authoring verification remain separate prerequisites.

Roll out owner support before an imported-task producer emits V2. Authored
MILESTONE producers continue emitting V1 with their exact published IDs; the
outer request, result and digest schema versions do not change. This extension
does not enable Start, deploy services, or certify cross-service execution.

## Digest and snapshot checks

Validation recomputes `practice-validation-contract-digest.v1` over canonical
JSON containing the schema, scope, optional milestone ID, and complete
validation specification. Object keys are sorted recursively, arrays retain
their order, insignificant whitespace is removed, and the UTF-8 bytes are
SHA-256 hashed. Strings use the owner's JSON encoding without HTML escaping.
JSON numbers retain their supplied lexemes using `json.Number`: `1e-7` is not
rewritten as `0.0000001`, and integers above 2^53 are not rounded through a
floating-point intermediate. Producers and persistence must preserve those
values. These rules apply to both specification versions.

The owner-generated
[digest vectors](../internal/usecase/testdata/practice_validation_digest_vectors.json)
pin V1 MILESTONE and V2 FINAL values containing `<main>&`, `1e-7` and
`9007199254740993`. They were computed using the existing owner digest function
at commit `30091f3c6d99a66e7e5b088042b20866e56901ad`, before the specification
extension; their constants are shared with Runtime compatibility tests.

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

LC-07 provider binding: Go capability/definition preparation transport, timeout, HTTP availability and response protocol failures reuse the existing typed validator ERROR classifications. Malformed immutable profile/binding/definition or digest drift remains AUTHORING_DEFECT. No new wire enum or receipt contract; provider causes are private and never learner diagnostics.
