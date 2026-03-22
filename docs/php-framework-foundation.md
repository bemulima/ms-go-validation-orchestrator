# PHP Framework Foundation

`php.core` remains the only implemented PHP engine today. It is intended for single-file and simple structural validation in [ms-go-php-validator](/Users/marat/Developments/microservices/ms-go-php-validator).

Framework-aware PHP validation is a separate capability family:

- `php.laravel`
- `php.yii2`
- `php.yii3`
- `php.symfony`

## What foundation means here

The platform now supports these engine IDs in:

- `ValidationContractV1`
- orchestrator engine registry
- admin contract authoring
- task capability inference
- example contracts and docs

This does **not** mean that Laravel, Yii2, Yii3, or Symfony validation is already implemented. It means the architecture is ready for a dedicated framework-aware engine.

## Why a separate engine family

The current PHP validator is token-based and single-file-oriented. Framework tasks need:

- multi-file workspace analysis
- routing and controller convention checks
- template/view resolution
- service/config wiring checks
- framework-specific semantics that do not belong in `php.core`

## Expected future contract shape

The framework adapters are expected to consume stage `rules` like:

```json
{
  "requiredRoutes": [
    {
      "file": "routes/web.php",
      "method": "GET",
      "path": "/dashboard",
      "uses": "DashboardController@index"
    }
  ],
  "requiredControllers": [
    {
      "path": "app/Http/Controllers/DashboardController.php",
      "className": "DashboardController",
      "extends": "Controller",
      "methods": ["index"]
    }
  ],
  "requiredViews": ["resources/views/dashboard.blade.php"]
}
```

Framework-specific extensions are expected, but the top-level stage shape stays the same:

- `engine`
- `framework`
- `targets`
- `rules`

## Current rollout rule

- Use `php.core` for production PHP tasks that must validate today.
- Use `php.laravel`, `php.yii2`, `php.yii3`, `php.symfony` only for foundation contracts, examples, or behind a dedicated future validator endpoint configured through `PHP_FRAMEWORK_VALIDATOR_URL`.
