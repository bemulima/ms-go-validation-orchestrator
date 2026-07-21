# Admin Authoring Guide

This guide describes how to author tasks with the new validation contract in admin UI.

## Modes

`admin-nextjs` now supports three authoring modes for `code_structure`.

## Basic

Use Basic mode when the task is:

- single-file
- beginner-oriented
- close to one of the built-in presets

Basic mode is intentionally narrow. It edits:

- primary profile
- profiles
- required files
- simple target files
- simple entrypoint fields

Use it for:

- HTML
- CSS
- JavaScript
- TypeScript
- simple React

## Advanced

Use Advanced mode when the task is:

- multi-stage
- multi-file
- composite
- dependent on explicit stage ordering

Advanced mode edits:

- `profile`
- `profiles`
- `workspace.required_files`
- `stages[]`
- `stage.depends_on`
- `stage.targets`
- `stage.timeout_seconds`
- `links[]`
- `link.config`

Use it for:

- `html + css`
- `html + css + js`
- `express + http.runtime`
- `php + css + js`

Complex engine-specific `rules` and `checks` still belong in Expert JSON.

## Expert JSON

Use Expert JSON when the task needs:

- rich engine-specific rules
- exact control over checks
- foundation-only engines
- hand-authored contract tuning
- final TypeScript CLI behavior through `ts.runtime`
- Java 21 compile/runtime behavior through `java.compile` and `java.runtime`
- Kotlin 2.3/JDK 21 compile/runtime behavior through `kotlin.compile` and `kotlin.runtime`

This is the canonical editing mode when you need full control.

## Presets

Use presets to create a clean stage-based starting point. Current built-ins include:

- HTML single-file
- CSS single-file
- JavaScript single-file
- TypeScript single-file
- HTML + CSS
- HTML + CSS + JS
- React component
- Express route
- Next.js foundation
- Browser runtime foundation
- PHP core

## Legacy tasks

Legacy tasks remain supported, but they should be considered migration candidates.

Admin UI now provides:

- legacy detection
- migration action to `ValidationContractV1`
- migration warnings

Always review the migrated contract before saving. Some legacy semantics are best-effort only.

## Authoring rules of thumb

- Put cheap static checks in `live` or `both`.
- Put runtime-heavy checks in `final`.
- Use `depends_on` to keep runtime stages behind structural ones.
- Use links for simple workspace assertions, not for deep semantic validation.
- Keep `code_structure_type` aligned with the dominant task profile, even though `code_structure` is now the main truth source.

## TypeScript CLI runtime

For a TypeScript sandbox task that needs immediate static feedback and authoritative behavioral validation:

1. Add a `ts.ast` stage with `mode: "both"`.
2. Add a `ts.runtime` stage with `mode: "final"` and `depends_on` pointing to the static stage.
3. Put the relative `.ts`, `.mts`, or `.cts` entrypoint in `targets.entrypoint`.
4. Put the constrained CLI contract in `checks`: `kind`, optional `args` and `timeoutMs`, then expected `exitCode` plus at least one stdout or stderr assertion.

The orchestrator never runs `ts.runtime` during live validation, even if the authored stage mode is accidentally broader. The node validator uses a fixed command and does not install task dependencies. See `docs/examples/ts-cli-runtime.json` for the canonical contract.

## Java 21 CLI runtime

For a Beginner Java sandbox task that needs compiler feedback while editing and
authoritative behavioral validation on Submit:

1. Add `java.compile` with `mode: "both"` and the exact `Main.java` target and entrypoint.
2. Add `java.runtime` with `mode: "final"` and `depends_on` pointing to the compile stage.
3. Use empty `rules`; the Beginner profile does not expose packages, dependencies, classpaths, or compiler plugins.
4. Put only constrained CLI fields in `checks`: `kind`, optional `args` and `timeoutMs`, then `expect.exitCode` plus at least one stdout or stderr assertion.

The orchestrator never runs `java.runtime` during live validation, even when a
bad contract marks it as `both`. The code validator owns the fixed `javac` and
`java` commands. See `docs/examples/java-cli-runtime.json`.

## Kotlin 2.3.21/JDK 21 CLI runtime

For a Beginner Kotlin sandbox task that needs compiler feedback while editing
and authoritative behavioral validation on Submit:

1. Add `kotlin.compile` with `mode: "both"` and the exact `Main.kt` target and entrypoint.
2. Add `kotlin.runtime` with `mode: "final"` and `depends_on` pointing to the compile stage.
3. Use empty `rules` and a top-level `fun main()` or `fun main(args: Array<String>)`; packages, Gradle files, dependencies, classpaths, and compiler plugins are not supported.
4. Put only constrained CLI fields in `checks`: `kind`, optional `args` and `timeoutMs`, then `expect.exitCode` plus at least one stdout or stderr assertion.

The orchestrator never runs `kotlin.runtime` during live validation, even when
a bad contract marks it as `both`. See
`docs/examples/kotlin-cli-runtime.json` for the canonical contract.

## Browser input and form runtime

Browser behavior is final-only. Pair `browser.runtime` with `html.dom` plus
`js.ast` or `ts.ast` in `both` mode so Live can return static feedback without
setting the authoritative behavioral result.

The supported interaction actions are `click`, `fill`, `type`, `submit`, and
the legacy `input` alias for `fill`. Every interaction must assert an
observable class, text, exact input value, or ordered text transition. Text
input tasks should exercise both non-empty and empty values.

Form tasks declare exact offline `networkMocks` with a method, deterministic
response sequence, optional short delay, and `expectedRequests`. Use
`expectTextSequence` to observe pending followed by success or error. The
request count rejects missing and duplicate submission handlers; undeclared
external HTTP(S) requests are blocked by the validator.

For TypeScript browser tasks, add the constrained build only:

```json
{"kind":"typescript","entrypoint":"app.ts","output":"app.js"}
```

The generated JavaScript file is not a student file. Custom shell commands,
compiler flags, imports, packages, and dependency installation are outside the
Beginner browser profile. See `docs/examples/browser-runtime-foundation.json`
and `docs/examples/browser-form-runtime.json`.

## PHP static and runtime split

For a PHP sandbox task that needs immediate syntax or structural feedback and
authoritative behavioral validation on Submit:

1. Add `php.core` with `mode: "both"` so the same prerequisite is available during Live and Submit.
2. Target the exact PHP file and keep syntax, token, and structural requirements in `rules`.
3. Put output, HTTP, or other behavioral execution in a separate runtime stage with `mode: "final"` and `depends_on` pointing to the static stage.

`php.core` runs `php -l` and static analyzers but never executes student code.
This makes it suitable for live editor feedback; it is not a replacement for a
final runtime assertion. See `docs/examples/php-single-file.json` for the
single-stage form and `docs/examples/php-css-js-foundation.json` for a
composite static contract. A standalone static stage may use `mode: "live"`,
but a final stage cannot depend on a live-only prerequisite.
