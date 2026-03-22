# E2E Compatibility Matrix

This matrix describes end-to-end platform behavior, not just isolated engine capability.

Read it together with:

- [Capability Matrix](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/capability-matrix.md)
- [Legacy Migration Guide](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/migration-guide.md)
- [Example Contracts](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/examples/README.md)

## Legend

- `Supported` means the platform path is wired and should work in normal operation.
- `Partial` means the path exists but still has scope limits or depends on task design.
- `Foundation` means authoring and orchestration are prepared, but real execution still needs a dedicated engine rollout.

## Platform gates

| Gate | Purpose |
| --- | --- |
| `VALIDATION_ORCHESTRATOR_URL` in `ms-go-task-answer` | Routes final validation of `ValidationContractV1` tasks through `ms-go-validation-orchestrator` |
| `VALIDATION_ORCHESTRATOR_URL` in `ms-go-sandbox` | Routes live validation of `ValidationContractV1` tasks through `ms-go-validation-orchestrator` |
| `NEXTJS_VALIDATOR_URL` in orchestrator | Enables `nextjs.app` execution |
| `BROWSER_RUNTIME_VALIDATOR_URL` in orchestrator | Enables `browser.runtime` execution |
| `PHP_FRAMEWORK_VALIDATOR_URL` in orchestrator | Reserved for future framework-aware PHP engines |

## Legacy tasks

| Scenario | Admin authoring | Sandbox live validation | Final validation | Student report | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| Legacy HTML task | Supported | Supported | Supported | Supported | Supported | Uses legacy path unless migrated |
| Legacy CSS task | Supported | Supported | Supported | Supported | Supported | Uses legacy path unless migrated |
| Legacy JS task | Supported | Supported | Supported | Supported | Supported | Uses legacy path unless migrated |
| Legacy TS task | Supported | Supported | Supported | Supported | Supported | Uses legacy path unless migrated |
| Legacy React task | Supported | Supported | Supported | Supported | Supported | Uses legacy path unless migrated |
| Legacy Express task | Supported | Supported | Supported | Supported | Supported | Uses legacy path unless migrated |
| Legacy Next.js task | Supported | Supported | Supported | Supported | Supported | Existing mapping remains available during transition |
| Legacy PHP task | Supported | Supported | Supported | Supported | Supported | Uses legacy path unless migrated |

## New ValidationContractV1 single-file tasks

| Scenario | Admin authoring | Sandbox live validation | Final validation | Student report | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `html.dom` single-file | Supported | Supported | Supported | Supported | Supported | Main happy-path reference |
| `css.ast` single-file | Supported | Supported | Supported | Supported | Supported | `scss.ast` follows same path with subset support |
| `js.ast` single-file | Supported | Supported | Supported | Supported | Supported | Routed through node validator |
| `ts.ast` single-file | Supported | Supported | Supported | Supported | Supported | Routed through node validator |
| `php.core` single-file | Supported | No | Supported | Supported | Partial | Final-ready engine; not intended for live |
| `react.ast` component | Supported | Supported | Supported | Supported | Supported | Best for component-level tasks |
| `nextjs.app` single workspace | Supported | No | Supported | Supported | Partial | Requires `NEXTJS_VALIDATOR_URL` |

## Composite frontend tasks

| Scenario | Admin authoring | Sandbox live validation | Final validation | Student report | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `html.dom` + `css.ast` | Supported | Supported | Supported | Supported | Supported | Recommended first composite rollout |
| `html.dom` + `css.ast` + `js.ast` | Supported | Supported | Supported | Supported | Supported | Use links only for light workspace assertions |
| `html.dom` + `scss.ast` + `ts.ast` | Supported | Supported | Supported | Supported | Partial | Depends on SCSS subset and task rule design |
| `html.dom` + Bootstrap + jQuery | Supported | Supported | Supported | Supported | Partial | Implement with profiles plus browser/runtime only when needed |
| `html.dom` + `css.ast` + `browser.runtime` | Supported | Task-dependent | Supported | Supported | Partial | Requires Playwright-compatible browser and careful runtime scope |

## Backend and framework tasks

| Scenario | Admin authoring | Sandbox live validation | Final validation | Student report | Status | Notes |
| --- | --- | --- | --- | --- | --- | --- |
| `node.express` only | Supported | Supported | Supported | Supported | Supported | Structure validation path is production-ready |
| `node.express` + `http.runtime` | Supported | No | Supported | Supported | Supported | Runtime should remain `final` in most tasks |
| `node.fastify` only | Supported | Supported | Supported | Supported | Supported | Same execution model as Express |
| `node.nest` only | Supported | Supported | Supported | Supported | Partial | Current structure subset is usable but not exhaustive |
| `react.ast` + `nextjs.app` | Supported | Partial | Supported | Supported | Partial | Use `react.ast` for component rules and `nextjs.app` for workspace rules |
| `php.core` + `css.ast` + `js.ast` | Supported | Partial | Supported | Supported | Partial | Good foundation pattern for mixed beginner fullstack tasks |
| `php.laravel` | Supported | No | No | Supported | Foundation | Contract and authoring ready; no real framework engine yet |
| `php.yii2` | Supported | No | No | Supported | Foundation | Contract and authoring ready; no real framework engine yet |
| `php.yii3` | Supported | No | No | Supported | Foundation | Contract and authoring ready; no real framework engine yet |
| `php.symfony` | Supported | No | No | Supported | Foundation | Contract and authoring ready; no real framework engine yet |

## Cross-platform behavior

| Area | Status | Notes |
| --- | --- | --- |
| Admin Basic mode | Supported | Good for simple single-file and starter composite contracts |
| Admin Advanced mode | Supported | Main production path for multi-stage contracts |
| Admin Expert JSON mode | Supported | Source of truth for full contract control |
| Admin legacy migration helper | Supported | Best-effort conversion, then manual review |
| Student workspace stage-based report | Supported | Includes stage grouping, severity filters, search, file navigation |
| Student final result stage-based report | Supported | Reads orchestrator-shaped result data |
| Gateway routing to orchestrator | Supported | Student/admin proxy path exists |
| Seeds with `ValidationContractV1` | Supported | `validation-contracts-lab` is canonical reference |

## Recommended production baseline

Start production rollout with these task families:

- `html.dom`
- `css.ast`
- `js.ast`
- `ts.ast`
- `react.ast`
- `node.express`
- `node.express` + `http.runtime`
- `html.dom` + `css.ast`
- `html.dom` + `css.ast` + `js.ast`

Hold these behind stricter rollout gates:

- `browser.runtime`
- `nextjs.app`
- `node.nest`
- `php.core` composites

Keep these as foundation-only until a dedicated engine exists:

- `php.laravel`
- `php.yii2`
- `php.yii3`
- `php.symfony`

## Exit criteria for legacy shutdown

Legacy fallback can be reduced only when all of the following are true:

1. New reference tasks exist for each major stack in `ValidationContractV1`.
2. Sandbox live validation is stable for the targeted stack family.
3. Final validation results are rendered correctly in student UI.
4. Admin migration helper has been used and manually reviewed for target courses.
5. Course seeds and starter workspaces do not auto-pass after migration.
