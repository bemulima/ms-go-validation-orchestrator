# Legacy Migration Guide

This guide explains how to move from legacy `code_structure` payloads to `ValidationContractV1`.

## Migration policy

- Legacy tasks remain executable.
- New authoring should use `ValidationContractV1`.
- Migration is incremental, not a flag-day rewrite.

## Migration path

1. Detect legacy payload in admin.
2. Run the admin migration helper.
3. Review generated stages, links, and warnings.
4. Save the migrated contract.
5. Re-test the task through sandbox/final validation.

## Current admin helper coverage

Supported legacy types:

- `html_document`
- `css_stylesheet`
- `javascript_file`
- `typescript_file`
- `react_component`
- `express_server`
- `nextjs_app`
- `php_script`

## Best-effort areas

Some legacy semantics do not map one-to-one.

### HTML

- multiple allowed `lang` values become one `htmlLang` target
- some required-only container flags are softened into explicit node checks

### CSS

- `required_strings` are not converted directly
- legacy selector constraints may need manual review in Expert JSON

### JavaScript / TypeScript

- generic semantic rules are migrated as stage rules plus simple file links
- review generated links for functions, variables, or interfaces

### React

- simple component/hook/tree/event patterns migrate well
- advanced props/state/tree semantics may need manual review

### PHP

- `php.core` migration is intentionally conservative
- complex legacy PHP rules may require manual cleanup after migration

## Old-to-new mapping overview

| Legacy type | New stage engine(s) |
| --- | --- |
| `html_document` | `html.dom` |
| `css_stylesheet` | `css.ast` |
| `javascript_file` | `js.ast` |
| `typescript_file` | `ts.ast` |
| `react_component` | `react.ast` |
| `express_server` | `node.express` + `http.runtime` |
| `nextjs_app` | `nextjs.app` |
| `php_script` | `php.core` |

## Seed migration

`ms-go-course` now contains a dedicated route fixture course:

- [validation-contracts-lab.json](/Users/marat/Developments/microservices/ms-go-course/db/seeds/sids/routes/validation-contracts-lab.json)

Use it as the canonical seed reference for:

- single-file contracts
- composite contracts
- foundation contracts

## Rollout advice

- migrate high-value reference tasks first
- keep `code_structure_type` stable while migrating
- validate starter workspaces so they do not accidentally auto-pass
- prefer course-by-course or lesson-by-lesson rollout over mass rewrite
