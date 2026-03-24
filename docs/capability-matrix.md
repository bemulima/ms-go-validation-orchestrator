# Capability Matrix

This matrix shows what each engine supports today and where support is still intentionally partial.

| Engine | Scope | Live-ready | Final-ready | Composite-ready | Status |
| --- | --- | --- | --- | --- | --- |
| `html.dom` | HTML structure, text, attributes, nested children | Yes | Yes | Yes | Implemented |
| `css.ast` | CSS AST, selectors, declarations, values | Yes | Yes | Yes | Implemented |
| `scss.ast` | SCSS through CSS validator path | Yes | Yes | Yes | Implemented subset |
| `react.ast` | Components, hooks, JSX tree, event handlers | Yes | Yes | Yes | Implemented |
| `js.ast` | JS static checks through node validator | Yes | Yes | Yes | Implemented |
| `ts.ast` | TS static checks through node validator | Yes | Yes | Yes | Implemented |
| `node.express` | Express structure validation | Yes | Yes | Yes | Implemented |
| `node.fastify` | Fastify structure validation | Yes | Yes | Yes | Implemented |
| `node.nest` | NestJS structure validation | Yes | Yes | Yes | Implemented subset |
| `http.runtime` | Runtime HTTP requests/responses | No | Yes | Yes | Implemented |
| `python.django.runtime` | Django runtime boot + HTTP assertions | No | Yes | Yes | Implemented subset |
| `go.gin.runtime` | Gin runtime boot + HTTP assertions | No | Yes | Yes | Implemented subset |
| `go.echo.runtime` | Echo runtime boot + HTTP assertions | No | Yes | Yes | Implemented subset |
| `php.laravel.runtime` | Laravel runtime boot + HTTP assertions | No | Yes | Yes | Implemented subset |
| `php.symfony.runtime` | Symfony runtime boot + HTTP assertions | No | Yes | Yes | Implemented subset |
| `php.yii2.runtime` | Yii2 runtime boot + HTTP assertions | No | Yes | Yes | Implemented subset |
| `php.yii3.runtime` | Yii3 runtime boot + HTTP assertions | No | Yes | Yes | Implemented subset |
| `browser.runtime` | DOM interactions and computed styles in real browser | Task-dependent | Yes | Yes | Implemented subset |
| `git.core` | Branch, clean state, tracked/ignored files, tags, commits | Sandbox-dependent | Yes | Yes | Implemented subset |
| `docker.dockerfile` | Static Dockerfile instruction validation | Yes | Yes | Yes | Implemented |
| `docker.compose` | Static Docker Compose structure validation | Yes | Yes | Yes | Implemented |
| `python.core` | Python AST checks for imports, functions, classes, variables | Yes | Yes | Yes | Implemented |
| `python.django` | Django project structure, settings, urls, models, views, templates | Task-dependent | Yes | Yes | Implemented subset |
| `golang` | Go AST checks for imports, functions, structs, interfaces, methods | Yes | Yes | Yes | Implemented |
| `go.core` | Go AST checks for imports, functions, structs, interfaces, methods | Yes | Yes | Yes | Implemented |
| `go.gin` | Gin route and group structure validation | Task-dependent | Yes | Yes | Implemented subset |
| `go.echo` | Echo route and group structure validation | Task-dependent | Yes | Yes | Implemented subset |
| `db.postgres.schema` | PostgreSQL SQL schema validation from workspace files | Yes | Yes | Yes | Implemented subset |
| `db.postgres.runtime` | PostgreSQL runtime queries with explicit DSN or auto-provisioned ephemeral DB | Task-dependent | Yes | Yes | Implemented |
| `db.mysql.schema` | MySQL SQL schema validation from workspace files | Yes | Yes | Yes | Implemented subset |
| `db.mysql.runtime` | MySQL runtime queries with explicit DSN or auto-provisioned ephemeral DB | Task-dependent | Yes | Yes | Implemented |
| `db.tarantool.schema` | Tarantool spaces, indexes, Lua schema fragments from workspace files | Yes | Yes | Yes | Implemented subset |
| `db.tarantool.runtime` | Tarantool Lua assertions against an explicit address or auto-provisioned temporary instance | Task-dependent | Yes | Yes | Implemented subset |
| `linux.fs` | Files, directories, executable bits, forbidden paths, file content | Yes | Yes | Yes | Implemented |
| `linux.cli` | Command execution with stdout/stderr and exit-code assertions | Task-dependent | Yes | Yes | Implemented |
| `linux.runtime` | Simple command workflows with post-run filesystem assertions | No | Yes | Yes | Implemented subset |
| `cache.redis.config` | Redis config fragments from workspace files | Yes | Yes | Yes | Implemented |
| `cache.redis.runtime` | Redis commands against an explicit connection or auto-provisioned Redis | Task-dependent | Yes | Yes | Implemented |
| `search.elasticsearch.mapping` | Elasticsearch mappings/settings from JSON files | Yes | Yes | Yes | Implemented |
| `search.elasticsearch.runtime` | Elasticsearch HTTP assertions against an explicit base URL or auto-provisioned node | Task-dependent | Yes | Yes | Implemented subset |
| `search.manticore` | Manticore config validation and runtime SphinxQL assertions | Task-dependent | Yes | Yes | Implemented subset |
| `search.sphinx` | Sphinx config validation and runtime SphinxQL assertions | Task-dependent | Yes | Yes | Implemented subset |
| `php.core` | Syntax + simple structural PHP checks | No | Yes | Yes | Implemented |
| `nextjs.app` | App Router pages, layouts, API routes, client/server boundaries | No | Yes | Yes | Implemented subset |
| `php.laravel` | Laravel workspace validation | Task-dependent | Yes | Yes | Implemented subset |
| `php.yii2` | Yii2 workspace validation | Task-dependent | Yes | Yes | Implemented subset |
| `php.yii3` | Yii3 workspace validation | Task-dependent | Yes | Yes | Implemented subset |
| `php.symfony` | Symfony workspace validation | Task-dependent | Yes | Yes | Implemented subset |
| `legacy.generic` | Temporary legacy compatibility wrapper | Yes | Yes | No | Transitional only |

## Notes by engine family

## HTML / CSS / Browser

- `html.dom` and `css.ast` are the main building blocks for frontend composites.
- `browser.runtime` is used when the task depends on actual DOM behavior or computed styles.
- `browser.runtime` should not replace `html.dom` or `css.ast`; it complements them.

## React / Next.js

- `react.ast` is for component-level checks.
- `nextjs.app` is for workspace and framework structure.
- Do not overload `react.ast` with framework-level Next.js rules.

## Node / API

- `js.ast` and `ts.ast` handle lightweight static checks.
- `node.express`, `node.fastify`, and `node.nest` handle framework structure.
- `http.runtime` should be used after structure stages, usually with `depends_on`.

## Git / Docker

- `git.core` requires a real repository path with `.git`; JSON-only workspace snapshots are not sufficient for full history validation.
- `docker.dockerfile` and `docker.compose` are static validators today.
- Docker runtime, build, and daemon-backed checks are intentionally out of scope for the first pass.

## Python / Go

- `python.core`, `golang`, and `go.core` are AST-based validators.
- `python.django`, `go.gin`, and `go.echo` validate framework structure.
- `python.django.runtime`, `go.gin.runtime`, and `go.echo.runtime` add framework-aware runtime execution on top of `ms-go-http-runtime-validator`.

## Databases

- `db.postgres.schema` and `db.mysql.schema` work from SQL files in the workspace snapshot.
- `db.postgres.runtime` and `db.mysql.runtime` execute runtime SQL assertions against an explicit connection DSN or an auto-provisioned ephemeral database.
- `db.tarantool.schema` validates `init.lua`-style schema files for spaces and indexes.
- `db.tarantool.runtime` executes Lua assertions against an explicit Tarantool address or an auto-provisioned temporary Tarantool instance.

## Linux

- `linux.fs` works against either `workspace.root_path` or a materialized JSON workspace.
- `linux.cli` runs command assertions in the workspace root or a stage-level working directory.
- `linux.runtime` is intended for small shell workflows and post-run file assertions, not for long-lived interactive terminal sessions.

## Cache / Search

- `cache.redis.config` validates static Redis config files.
- `cache.redis.runtime` executes Redis commands against an explicit connection address or an auto-provisioned Redis container.
- `search.elasticsearch.mapping` validates JSON mapping/settings files in the workspace.
- `search.elasticsearch.runtime` executes HTTP assertions against an explicit Elasticsearch base URL or an auto-provisioned node.
- `search.manticore` and `search.sphinx` validate config/index/source definitions and can additionally run SphinxQL assertions when `checks` are present.

## PHP

- `php.core` is still the most mature PHP engine.
- `php.laravel`, `php.yii2`, `php.yii3`, and `php.symfony` validate framework structure.
- `php.laravel.runtime`, `php.symfony.runtime`, `php.yii2.runtime`, and `php.yii3.runtime` provide framework-aware runtime execution, but coverage is still a subset rather than full framework lifecycle validation.

## Authoring guidance

- Prefer implemented engines for production tasks.
- Treat `Implemented subset` rows as production-capable only when your task stays within the documented subset.
