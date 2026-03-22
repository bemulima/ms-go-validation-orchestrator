# Validation Contract V1

`ValidationContractV1` is the canonical stage-based contract for the validation orchestrator.

## Root fields

- `version`: contract version.
- `kind`: contract kind, for example `workspace_contract`.
- `profile`: optional primary profile.
- `profiles`: optional stack/profile list.
- `workspace.required_files`: files required for the task workspace.
- `stages[]`: ordered-by-dependency validation stages.
- `links[]`: cross-stage or workspace-level assertions.

## Stage fields

- `id`: unique stage identifier.
- `engine`: target engine, for example `html.dom`, `css.ast`, `react.ast`, `node.express`, `php.core`.
- `language`: optional language hint.
- `framework`: optional framework hint.
- `mode`: `live`, `final`, or `both`.
- `optional`: whether a failed stage should be non-blocking.
- `depends_on[]`: required predecessor stages.
- `targets.files[]`: stage file targets.
- `targets.entrypoint`: workspace entrypoint where relevant.
- `rules`: engine-specific static/structural rules.
- `checks`: engine-specific runtime checks.

## Link fields

- `id`: unique link identifier.
- `kind`: link kind.
- `optional`: whether a failed link is non-blocking.
- `depends_on[]`: required predecessor stages.
- `config`: link-specific payload.

## Initial link support

- `workspace.file_contains`
- `workspace.selector_exists`

Advanced cross-stage links will be added incrementally after engines expose stronger evidence.

## Examples

Canonical contract examples live in [docs/examples](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/examples/README.md). They cover single-file tasks, composite stacks, and future-foundation contracts such as Next.js and PHP multi-stack tasks.

## Related docs

- [Validation Result V1](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/validation-result.md)
- [Engine Model](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/engine-model.md)
- [Capability Matrix](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/capability-matrix.md)
- [Admin Authoring Guide](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/authoring-guide.md)
- [Legacy Migration Guide](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/migration-guide.md)
