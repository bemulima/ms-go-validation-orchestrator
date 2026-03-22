# Rollout Plan

This plan describes how to enable the new validation platform without breaking legacy tasks.

It assumes the following architecture is already present:

- `ms-go-course` stores both legacy and `ValidationContractV1` payloads
- `ms-go-task-answer` routes new contracts through `ms-go-validation-orchestrator`
- `ms-go-sandbox` routes new live validation through `ms-go-validation-orchestrator`
- admin and student frontends understand the new contract/report shape

## Core rollout principles

- Do not mass-migrate tasks without course-level review.
- Keep legacy fallback available until each stack family has passed E2E checks.
- Enable expensive runtime engines only after static and structural stages are stable.
- Prefer new authoring for new tasks before migrating old tasks.

## Phase 0. Platform wiring verification

Goal:

- confirm service connectivity and fallback behavior

Required checks:

- `ms-getway` proxies orchestrator routes
- `VALIDATION_ORCHESTRATOR_URL` is set in `ms-go-task-answer` and `ms-go-sandbox`
- orchestrator health endpoint is reachable
- legacy tasks still pass through old validation path

Success criteria:

- no regression for legacy tasks
- new contract tasks can be created and stored
- orchestrator failures do not silently fake success

## Phase 1. Authoring-first enablement

Goal:

- make `ValidationContractV1` the preferred format for new tasks

Required checks:

- admin Basic, Advanced, and Expert modes work in create/edit flows
- legacy migration helper is available but optional
- canonical presets and examples are visible to content authors

Success criteria:

- at least one new task per supported stack is authored without legacy JSON
- editors can save and re-open new contracts without data loss

## Phase 2. Reference seeds and fixtures

Goal:

- establish stable reference tasks and starter workspaces

Required checks:

- `validation-contracts-lab` seeds import correctly
- starter archives for new contracts contain only intended files
- no starter workspace auto-passes validation

Success criteria:

- reference tasks exist for HTML, CSS, JS, TS, React, Express, Next.js, PHP foundation, and composites
- seed validation scripts pass in CI/local verification

## Phase 3. Final validation rollout

Goal:

- use orchestrator in final submission flow for selected stacks

Recommended first stacks:

- `html.dom`
- `css.ast`
- `js.ast`
- `ts.ast`
- `react.ast`
- `node.express`
- `html.dom` + `css.ast`
- `html.dom` + `css.ast` + `js.ast`

Required checks:

- final submission stores normalized `ValidationResultV1`
- student result pages render structured stage reports
- auto-review behaves correctly for pass/fail and allowed-error thresholds

Success criteria:

- selected stacks use orchestrator in production-like environments
- rollback to legacy path remains possible by config

## Phase 4. Live validation rollout

Goal:

- enable orchestrator in sandbox live validation for cheap stages

Recommended live stage families:

- `html.dom`
- `css.ast`
- `scss.ast`
- `react.ast`
- `js.ast`
- `ts.ast`
- `node.express`
- `node.fastify`
- simple `workspace.*` links

Keep final-only by default:

- `http.runtime`
- `php.core`
- `nextjs.app`
- `browser.runtime`

Required checks:

- live validation latency is acceptable
- stage mode filtering works
- student workspace shows structured issues without blocking editor flow

Success criteria:

- supported live stages feel responsive in sandbox
- runtime-heavy stages do not execute on each file update

## Phase 5. Extended engines

Goal:

- roll out heavier or narrower engines after baseline success

Scope:

- `nextjs.app`
- `browser.runtime`
- `node.nest`
- `php.core` composite tasks

Required checks:

- dedicated engine service is reachable
- environment-specific dependencies are installed
- runtime cost and failure modes are understood

Success criteria:

- each extended engine is enabled only for task families that have stable reference fixtures

## Phase 6. Framework foundations

Goal:

- prepare future framework-aware PHP rollout without false production claims

Scope:

- `php.laravel`
- `php.yii2`
- `php.yii3`
- `php.symfony`

Required checks:

- dedicated framework validator service exists
- contracts and examples are validated against a real runtime
- admin presets stop being marked as foundation-only only after that runtime exists

Success criteria:

- framework tasks are enabled only when execution is real, not simulated

## Recommended stack order

1. HTML
2. CSS / SCSS
3. JS / TS
4. React
5. Express
6. HTML + CSS
7. HTML + CSS + JS
8. Express + HTTP runtime
9. Next.js
10. Browser runtime tasks
11. PHP core
12. PHP framework families

## Rollback strategy

If a rollout causes regressions:

1. Stop authoring new tasks for the affected stack family.
2. Disable engine endpoint configuration if the failure is engine-specific.
3. Keep legacy tasks on the old path.
4. Re-run reference tasks from `validation-contracts-lab`.
5. Restore the stack family only after the failing scenario has a deterministic fixture and test.

## Operational checklist

Before enabling a stack family in production-like environments:

- engine URL is configured
- gateway route is reachable
- at least one seed/reference task passes
- student workspace renders live issues correctly
- student result page renders final report correctly
- admin can create and edit the contract without JSON loss
- fallback behavior is documented for the team

## Minimal release gates

Use these gates before calling the platform rollout-complete:

- all implemented engines have at least one canonical example contract
- all supported stacks have at least one E2E reference task
- orchestrator path is used in both sandbox and final validation for supported stacks
- admin migration helper is available for legacy tasks
- student UI can open files from validation report issues
- docs package and capability matrix are up to date
