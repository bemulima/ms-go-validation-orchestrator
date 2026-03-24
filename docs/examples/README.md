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
- `golang-basic.json`
- `go-core-basic.json`
- `go-gin-basic.json`
- `go-gin-runtime.json`
- `go-echo-basic.json`
- `go-echo-runtime.json`
- `postgres-schema-basic.json`
- `postgres-runtime-autoprovision.json`
- `mysql-runtime-basic.json`
- `tarantool-runtime-autoprovision.json`
- `tarantool-schema-basic.json`
- `linux-fs-basic.json`
- `linux-cli-basic.json`
- `redis-config-basic.json`
- `redis-runtime-autoprovision.json`
- `elasticsearch-runtime-basic.json`
- `elasticsearch-runtime-autoprovision.json`
- `manticore-config-basic.json`
- `manticore-runtime-basic.json`
- `sphinx-config-basic.json`
- `sphinx-runtime-basic.json`
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
- `postgres-schema-basic.json`, `postgres-runtime-autoprovision.json`, and `mysql-runtime-basic.json` show the database validator family, including auto-provisioned runtime.
- `tarantool-schema-basic.json` and `tarantool-runtime-autoprovision.json` show Tarantool schema and runtime patterns inside the DB validator family.
- `linux-fs-basic.json` and `linux-cli-basic.json` show the first-pass Linux validator family.
- `redis-config-basic.json`, `redis-runtime-autoprovision.json`, `elasticsearch-runtime-basic.json`, and `elasticsearch-runtime-autoprovision.json` show the cache/search validator family.
- `manticore-config-basic.json`, `manticore-runtime-basic.json`, `sphinx-config-basic.json`, and `sphinx-runtime-basic.json` show config + runtime patterns for search-engine stages.
- Composite examples intentionally use only link kinds that are implemented in the current orchestrator foundation.
