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
- `engine`: target engine, for example `html.dom`, `css.ast`, `react.ast`, `node.express`, `ts.runtime`, `java.compile`, `java.runtime`, `kotlin.compile`, `kotlin.runtime`, `php.core`.
- `language`: optional language hint.
- `framework`: optional framework hint.
- `mode`: `live`, `final`, or `both`.
- `optional`: whether a failed stage should be non-blocking.
- `depends_on[]`: required predecessor stages.
- `targets.files[]`: stage file targets.
- `targets.entrypoint`: workspace entrypoint where relevant.
- `rules`: engine-specific static/structural rules.
- `checks`: engine-specific runtime checks.

For `ts.runtime`, `checks` uses the node validator's constrained CLI payload: `kind: "cli"`, optional `args` and `timeoutMs`, and `expect` with `exitCode` plus at least one stdout or stderr assertion. The engine is final-only at orchestration time. Pair it with a `ts.ast` stage in `both` mode when live static feedback is required.

For Beginner Java 21 tasks, pair `java.compile` in `both` mode with
`java.runtime` in `final` mode. Both stages target the exact dependency-free
`Main.java` entrypoint. Runtime `checks` uses the same constrained CLI field
shape as above, but commands, classpaths, JVM flags, packages, dependencies,
and compiler plugins are not authorable. The orchestrator hard-gates
`java.runtime` to final requests.

For Beginner Kotlin 2.3.21/JDK 21 tasks, pair `kotlin.compile` in `both`
mode with `kotlin.runtime` in `final` mode. Both stages target the exact
dependency-free `Main.kt` entrypoint and use empty `rules`. Runtime `checks`
uses the constrained CLI shape; packages, Gradle files, dependencies,
compiler flags, custom commands, and custom classpaths are outside this
profile. The orchestrator hard-gates `kotlin.runtime` to final requests.

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
