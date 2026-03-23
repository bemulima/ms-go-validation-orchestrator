# Engine Model

The platform is built around one orchestration layer and many engines.

## Why engines exist

Each technology stack has different validation requirements:

- HTML needs DOM structure validation.
- CSS needs stylesheet AST validation.
- React needs JSX and hook-aware AST checks.
- Node frameworks need static, structure, and HTTP runtime validation.
- Browser tasks need real DOM/runtime execution.
- PHP beginner tasks need simple syntax and structural checks.

Trying to collapse all of that into one validator would create a large, tightly coupled service with weak boundaries. The orchestrator exists to keep those concerns separate.

## Roles

### Orchestrator

`ms-go-validation-orchestrator` is responsible for:

- parsing `ValidationContractV1`
- adapting legacy payloads
- planning stages by `depends_on`
- running the right engine for each stage
- evaluating simple links
- aggregating a normalized result

It does **not** perform deep language or framework parsing itself.

### Engines

Each engine validates one technology concern:

- `html.dom`
- `css.ast`
- `scss.ast`
- `react.ast`
- `js.ast`
- `ts.ast`
- `node.express`
- `node.fastify`
- `node.nest`
- `http.runtime`
- `python.django.runtime`
- `go.gin.runtime`
- `go.echo.runtime`
- `php.laravel.runtime`
- `php.symfony.runtime`
- `php.yii2.runtime`
- `php.yii3.runtime`
- `browser.runtime`
- `git.core`
- `docker.dockerfile`
- `docker.compose`
- `python.core`
- `python.django`
- `go.core`
- `go.gin`
- `go.echo`
- `php.core`
- `php.laravel`
- `php.yii2`
- `php.yii3`
- `php.symfony`
- `nextjs.app`

## Stage model

One task can contain many stages. Typical patterns:

### Single-file task

- one stage

Example:

- `html.dom` only
- `php.core` only

### Composite static task

- multiple AST/structure stages
- optional file/string links

Example:

- `html.dom`
- `css.ast`
- `js.ast`

### Runtime-backed task

- structure/static stage first
- runtime stage after dependency success

Example:

- `node.express`
- `http.runtime`

Or:

- `python.django`
- `python.django.runtime`

### Fullstack task

- backend or framework stage
- optional frontend stages
- optional links

Example:

- `php.laravel`
- `php.laravel.runtime`
- `css.ast`
- `js.ast`

### Repository or infrastructure task

- repository or container stage
- optional file/string links

Example:

- `git.core`
- `docker.dockerfile`
- `docker.compose`

### Language and framework backend task

- language stage
- optional framework structure stage
- optional runtime stage later

Example:

- `python.core`
- `python.django`

Or:

- `go.core`
- `go.gin`

## Live vs final

`stage.mode` controls when a stage may run:

- `live`
- `final`
- `both`

General rule:

- keep cheap static checks in `live` or `both`
- keep runtime-heavy checks in `final`

Examples:

- `html.dom`, `css.ast`, `react.ast` are usually `both`
- `http.runtime` is usually `final`
- `python.django.runtime`, `go.gin.runtime`, `go.echo.runtime`, `php.laravel.runtime`, `php.symfony.runtime`, `php.yii2.runtime`, and `php.yii3.runtime` are usually `final`
- `nextjs.app` is currently used as `final`
- `browser.runtime` can be `final` or `both` depending on task cost
- `git.core` is currently most reliable in sandbox-backed `final` mode because it needs a real `.git`
- `docker.dockerfile` and `docker.compose` are usually `both`
- `python.core` and `go.core` are usually `both`
- `python.django`, `go.gin`, `go.echo`, `php.laravel`, `php.symfony`, `php.yii2`, and `php.yii3` are currently best as structure stages paired with runtime stages when HTTP behavior matters

## Links

Links are lightweight assertions outside engine-specific parsing. Current link support is intentionally narrow:

- `workspace.file_contains`
- `workspace.selector_exists`

Use links when:

- you need a simple workspace-level check
- the assertion does not justify a whole new stage

Do not use links as a substitute for rich structural validation when a real engine already exists.

## Backward compatibility

Legacy `code_structure` payloads still exist. The orchestrator wraps them into a temporary `legacy.generic` stage so platform services can route legacy and new tasks through one entrypoint.

That compatibility path is transitional. New authoring should target `ValidationContractV1`.
