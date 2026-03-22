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
- `php-laravel-foundation.json`
- `php-yii2-foundation.json`
- `php-yii3-foundation.json`
- `php-symfony-foundation.json`
- `html-css-composite.json`
- `html-css-js-composite.json`
- `php-css-js-foundation.json`

Notes:

- `php-laravel-foundation.json`, `php-yii2-foundation.json`, `php-yii3-foundation.json`, and `php-symfony-foundation.json` are framework-foundation examples. They are valid contracts and platform-supported engine IDs, but they still require a dedicated future PHP framework validator service.
- `php-css-js-foundation.json` is a foundation example for a future cross-stack task.
- `nextjs-basic-foundation.json` and `browser-runtime-foundation.json` are now backed by dedicated engines, but they still intentionally cover only the currently implemented subset of checks.
- Composite examples intentionally use only link kinds that are implemented in the current orchestrator foundation.
