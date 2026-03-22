# Capability Matrix

This matrix shows what each engine supports today and what remains foundation-only.

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
| `browser.runtime` | DOM interactions and computed styles in real browser | Task-dependent | Yes | Yes | Implemented subset |
| `php.core` | Syntax + simple structural PHP checks | No | Yes | Yes | Implemented |
| `nextjs.app` | App Router pages, layouts, API routes, client/server boundaries | No | Yes | Yes | Implemented subset |
| `php.laravel` | Laravel workspace validation | No | No | Yes | Foundation only |
| `php.yii2` | Yii2 workspace validation | No | No | Yes | Foundation only |
| `php.yii3` | Yii3 workspace validation | No | No | Yes | Foundation only |
| `php.symfony` | Symfony workspace validation | No | No | Yes | Foundation only |
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

## PHP

- `php.core` is the production-ready PHP engine today.
- Framework engines exist in platform contracts and authoring, but they still need a dedicated runtime service.

## Authoring guidance

- Prefer implemented engines for production tasks.
- Use foundation-only engines only when you explicitly want a forward-looking contract and you understand that execution still depends on future engine rollout.
