# Validation Contract Examples

These examples show canonical `ValidationContractV1` payloads for the main task shapes that the platform needs to support.

- `html-single-file.json`
- `css-single-file.json`
- `js-single-file.json`
- `ts-single-file.json`
- `php-single-file.json`
- `react-component.json`
- `express-route.json`
- `nextjs-basic-foundation.json`
- `browser-runtime-foundation.json`
- `git-core-basic.json`
- `docker-dockerfile-basic.json`
- `docker-compose-basic.json`
- `python-core-basic.json`
- `python-django-basic.json`
- `python-django-runtime.json`
- `go-core-basic.json`
- `go-gin-basic.json`
- `go-gin-runtime.json`
- `go-echo-basic.json`
- `go-echo-runtime.json`
- `php-laravel-foundation.json`
- `php-laravel-runtime.json`
- `php-yii2-foundation.json`
- `php-yii2-runtime.json`
- `php-yii3-foundation.json`
- `php-yii3-runtime.json`
- `php-symfony-foundation.json`
- `php-symfony-runtime.json`
- `html-css-composite.json`
- `html-css-js-composite.json`
- `php-css-js-foundation.json`

Notes:

- `php-laravel-foundation.json`, `php-yii2-foundation.json`, `php-yii3-foundation.json`, and `php-symfony-foundation.json` are backed by a dedicated static framework validator. The matching `*-runtime.json` examples show how each framework is now composed with dedicated runtime engines on top of the same runtime service.
- `php-css-js-foundation.json` is a foundation example for a future cross-stack task.
- `nextjs-basic-foundation.json` and `browser-runtime-foundation.json` are now backed by dedicated engines, but they still intentionally cover only the currently implemented subset of checks.
- `git-core-basic.json` requires a real repository path at runtime. JSON file snapshots alone are not enough for full Git validation.
- `python-django-runtime.json`, `go-gin-runtime.json`, and `go-echo-runtime.json` show how to compose framework structure validation with generic `http.runtime`.
- Composite examples intentionally use only link kinds that are implemented in the current orchestrator foundation.
