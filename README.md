# ms-go-validation-orchestrator

Validation orchestrator for stage-based task validation.

## Scope

The service accepts a unified validation contract, plans validation stages, invokes engine adapters, evaluates simple cross-stage links, and returns a normalized validation report.

This initial platform pass is intentionally conservative:

- new `ValidationContractV1` is supported;
- legacy `code_structure` is adapted into a temporary `legacy.generic` stage;
- legacy execution itself is not migrated yet, so adapted legacy contracts return an explicit compatibility error instead of fake success;
- advanced framework validators are intentionally split into structure and runtime stages rather than one monolithic engine.
- `browser.runtime` is now available when `BROWSER_RUNTIME_VALIDATOR_URL` points to `ms-ts-browser-runtime-validator`.
- `nextjs.app` is now available when `NEXTJS_VALIDATOR_URL` points to `ms-ts-nextjs-validator`.
- `git.core` is now available when `GIT_VALIDATOR_URL` points to `ms-go-git-validator`.
- `docker.dockerfile` and `docker.compose` are now available when `DOCKER_VALIDATOR_URL` points to `ms-go-docker-validator`.
- `python.core` and `python.django` are now available when `PYTHON_VALIDATOR_URL` points to `ms-py-validator`.
- `golang`, `go.core`, `go.gin`, and `go.echo` are now available when `GO_CODE_VALIDATOR_URL` points to `ms-go-code-validator`.
- `db.postgres.schema`, `db.postgres.runtime`, `db.mysql.schema`, and `db.mysql.runtime` are now available when `DB_VALIDATOR_URL` points to `ms-go-db-validator`.
- `db.tarantool.schema` and `db.tarantool.runtime` are now available when `DB_VALIDATOR_URL` points to `ms-go-db-validator`.
- `linux.fs`, `linux.cli`, and `linux.runtime` are now available when `LINUX_VALIDATOR_URL` points to `ms-go-linux-validator`.
- `cache.redis.config`, `cache.redis.runtime`, `search.elasticsearch.mapping`, and `search.elasticsearch.runtime` are now available when `CACHE_SEARCH_VALIDATOR_URL` points to `ms-go-cache-search-validator`.
- `search.manticore` and `search.sphinx` are now available when `CACHE_SEARCH_VALIDATOR_URL` points to `ms-go-cache-search-validator`.
- generic backend `http.runtime` is now available when `HTTP_RUNTIME_VALIDATOR_URL` points to `ms-go-http-runtime-validator`.

## Configuration

Environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `HOST` | `0.0.0.0` | HTTP listen host |
| `PORT` | `8080` | HTTP listen port |
| `SERVICE_NAME` | `ms-go-validation-orchestrator` | Service name for logs and ops |
| `HTML_VALIDATOR_URL` | `""` | Base URL of `ms-ts-html-validator` |
| `CSS_VALIDATOR_URL` | `""` | Base URL of `ms-ts-css-validator` |
| `REACT_VALIDATOR_URL` | `""` | Base URL of `ms-ts-react-validator` |
| `NODE_VALIDATOR_URL` | `""` | Base URL of `ms-node-validator` |
| `PHP_VALIDATOR_URL` | `""` | Base URL of `ms-go-php-validator` |
| `PHP_FRAMEWORK_VALIDATOR_URL` | `""` | Base URL of `ms-go-php-framework-validator` for `php.laravel`, `php.yii2`, `php.yii3`, `php.symfony` |
| `NEXTJS_VALIDATOR_URL` | `""` | Base URL of `ms-ts-nextjs-validator` |
| `BROWSER_RUNTIME_VALIDATOR_URL` | `""` | Base URL of `ms-ts-browser-runtime-validator` |
| `GIT_VALIDATOR_URL` | `""` | Base URL of `ms-go-git-validator` |
| `DOCKER_VALIDATOR_URL` | `""` | Base URL of `ms-go-docker-validator` |
| `PYTHON_VALIDATOR_URL` | `""` | Base URL of `ms-py-validator` |
| `GO_CODE_VALIDATOR_URL` | `""` | Base URL of `ms-go-code-validator` |
| `DB_VALIDATOR_URL` | `""` | Base URL of `ms-go-db-validator` |
| `LINUX_VALIDATOR_URL` | `""` | Base URL of `ms-go-linux-validator` |
| `CACHE_SEARCH_VALIDATOR_URL` | `""` | Base URL of `ms-go-cache-search-validator` |
| `HTTP_RUNTIME_VALIDATOR_URL` | `""` | Base URL of `ms-go-http-runtime-validator` |

## Run

```bash
go run ./cmd/ms-go-validation-orchestrator
```

## Test

```bash
go test ./... -count=1
```

`ms-go-validation-orchestrator` is the new validation platform entrypoint for rich task contracts.

## Documentation

- [Docs Index](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/README.md)
- [Validation Contract V1](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/validation-contract.md)
- [Validation Result V1](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/validation-result.md)
- [Engine Model](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/engine-model.md)
- [Capability Matrix](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/capability-matrix.md)
- [E2E Compatibility Matrix](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/e2e-compatibility-matrix.md)
- [Rollout Plan](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/rollout-plan.md)
- [Admin Authoring Guide](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/authoring-guide.md)
- [Legacy Migration Guide](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/migration-guide.md)
- [Example Contracts](/Users/marat/Developments/microservices/ms-go-validation-orchestrator/docs/examples/README.md)

## Current scope

- Canonical `ValidationContractV1`
- Canonical `ValidationResultV1`
- Stage planning with `depends_on`
- Engine registry and adapters for:
  - `html.dom`
  - `css.ast`
  - `scss.ast`
  - `react.ast`
  - `js.ast`
  - `ts.ast`
  - `node.express`
  - `node.nest`
  - `node.fastify`
- `http.runtime`
- `python.django.runtime`
- `go.gin.runtime`
- `go.echo.runtime`
- `php.laravel.runtime`
- `php.symfony.runtime`
- `php.yii2.runtime`
- `php.yii3.runtime`
- `php.core`
- `php.laravel`
- `php.yii2`
- `php.yii3`
- `php.symfony`
- `nextjs.app`
- `browser.runtime`
- `git.core`
- `docker.dockerfile`
- `docker.compose`
- `python.core`
- `python.django`
- `golang`
- `go.core`
- `go.gin`
- `go.echo`
- `db.postgres.schema`
- `db.postgres.runtime`
- `db.mysql.schema`
- `db.mysql.runtime`
- `db.tarantool.schema`
- `db.tarantool.runtime`
- `linux.fs`
- `linux.cli`
- `linux.runtime`
- `cache.redis.config`
- `cache.redis.runtime`
- `search.elasticsearch.mapping`
- `search.elasticsearch.runtime`
- `search.manticore`
- `search.sphinx`
- Legacy adapter that wraps old `code_structure` payloads into `legacy.generic`

## Important limitations

- Legacy execution is not migrated yet. `legacy.generic` returns an explicit issue so platform services can keep their old fallback path until migration is complete.
- `html.dom` inline rules are supported, but advanced selector semantics are still limited by the current HTML engine implementation.
- Cross-stage links are intentionally minimal in this platform pass.
- `php.laravel`, `php.yii2`, `php.yii3`, and `php.symfony` are now backed by a dedicated static workspace validator, but they still cover a limited subset of framework checks.
- `nextjs.app` requires `ms-ts-nextjs-validator`.
- `browser.runtime` requires `ms-ts-browser-runtime-validator` and a Playwright-compatible browser.
- `git.core` requires `workspace.root_path` with a real `.git` repository; JSON file snapshots alone are not enough.
- `docker.dockerfile` and `docker.compose` are static-only in this first pass; they do not run Docker builds or compose stacks.
- `db.postgres.runtime` and `db.mysql.runtime` can auto-provision ephemeral databases when the DB validator has Docker access. Explicit DSN is still supported.
- `linux.cli` and `linux.runtime` execute commands inside the validator environment and should be kept to deterministic, short-lived checks.
- `cache.redis.runtime`, `search.elasticsearch.runtime`, `search.manticore`, and `search.sphinx` can auto-provision ephemeral runtime containers when the cache/search validator has Docker access. Explicit endpoints are still supported.
- `db.tarantool.runtime` can auto-provision a temporary Tarantool instance when the DB validator has Docker access and a runtime `init.lua` is available in the workspace.
- `python.django.runtime`, `go.gin.runtime`, `go.echo.runtime`, `php.laravel.runtime`, `php.symfony.runtime`, `php.yii2.runtime`, and `php.yii3.runtime` are framework-aware runtime wrappers around `ms-go-http-runtime-validator`.
- `http.runtime` remains available as the generic escape hatch when a task must provide an explicit `checks.command`.

## HTTP API

### `POST /api/v1/validate`

Request:

```json
{
  "task_id": "task-1",
  "code_structure_type_code": "HTML_BASIC_DOCUMENT",
  "code_structure": {
    "version": 1,
    "kind": "workspace_contract",
    "stages": [
      {
        "id": "styles",
        "engine": "css.ast",
        "targets": { "files": ["style.css"] },
        "rules": {
          "rules": [
            {
              "selector": { "value": ".card" },
              "declarations": [
                {
                  "property": { "value": "background" },
                  "value": { "value": "#fff" }
                }
              ]
            }
          ]
        }
      }
    ]
  },
  "workspace": {
    "root_path": "/workspace",
    "files": [
      { "path": "style.css", "content": ".card{background:#fff}" }
    ]
  }
}
```

## Local commands

```bash
task fmt
task test
task run
```
