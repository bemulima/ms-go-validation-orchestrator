# ms-go-validation-orchestrator

Validation orchestrator for stage-based task validation.

## Scope

The service accepts a unified validation contract, plans validation stages, invokes engine adapters, evaluates simple cross-stage links, and returns a normalized validation report.

This initial foundation is intentionally conservative:

- new `ValidationContractV1` is supported;
- legacy `code_structure` is adapted into a temporary `legacy.generic` stage;
- legacy execution itself is not migrated yet, so adapted legacy contracts return an explicit compatibility error instead of fake success;
- advanced PHP framework validation is not fully implemented yet, but the orchestrator already has foundation hooks for dedicated future engine services.
- `browser.runtime` is now available when `BROWSER_RUNTIME_VALIDATOR_URL` points to `ms-ts-browser-runtime-validator`.
- `nextjs.app` is now available when `NEXTJS_VALIDATOR_URL` points to `ms-ts-nextjs-validator`.

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
| `PHP_FRAMEWORK_VALIDATOR_URL` | `""` | Base URL of a future PHP framework validator for `php.laravel`, `php.yii2`, `php.yii3`, `php.symfony` |
| `NEXTJS_VALIDATOR_URL` | `""` | Base URL of `ms-ts-nextjs-validator` |
| `BROWSER_RUNTIME_VALIDATOR_URL` | `""` | Base URL of `ms-ts-browser-runtime-validator` |

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
- `php.core`
- `php.laravel` foundation hook
- `php.yii2` foundation hook
- `php.yii3` foundation hook
- `php.symfony` foundation hook
- `nextjs.app`
- `browser.runtime`
- Legacy adapter that wraps old `code_structure` payloads into `legacy.generic`

## Important limitations

- Legacy execution is not migrated yet. `legacy.generic` returns an explicit issue so platform services can keep their old fallback path until migration is complete.
- `html.dom` inline rules are supported, but advanced selector semantics are still limited by the current HTML engine implementation.
- Cross-stage links are intentionally minimal in this first foundation pass.
- `php.core` remains the only implemented PHP engine today. `php.laravel`, `php.yii2`, `php.yii3`, and `php.symfony` are orchestration-ready foundation hooks that require a dedicated framework-aware validator endpoint.
- `nextjs.app` requires `ms-ts-nextjs-validator`.
- `browser.runtime` requires `ms-ts-browser-runtime-validator` and a Playwright-compatible browser.

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
