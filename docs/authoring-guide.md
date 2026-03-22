# Admin Authoring Guide

This guide describes how to author tasks with the new validation contract in admin UI.

## Modes

`admin-nextjs` now supports three authoring modes for `code_structure`.

## Basic

Use Basic mode when the task is:

- single-file
- beginner-oriented
- close to one of the built-in presets

Basic mode is intentionally narrow. It edits:

- primary profile
- profiles
- required files
- simple target files
- simple entrypoint fields

Use it for:

- HTML
- CSS
- JavaScript
- TypeScript
- simple React

## Advanced

Use Advanced mode when the task is:

- multi-stage
- multi-file
- composite
- dependent on explicit stage ordering

Advanced mode edits:

- `profile`
- `profiles`
- `workspace.required_files`
- `stages[]`
- `stage.depends_on`
- `stage.targets`
- `stage.timeout_seconds`
- `links[]`
- `link.config`

Use it for:

- `html + css`
- `html + css + js`
- `express + http.runtime`
- `php + css + js`

Complex engine-specific `rules` and `checks` still belong in Expert JSON.

## Expert JSON

Use Expert JSON when the task needs:

- rich engine-specific rules
- exact control over checks
- foundation-only engines
- hand-authored contract tuning

This is the canonical editing mode when you need full control.

## Presets

Use presets to create a clean stage-based starting point. Current built-ins include:

- HTML single-file
- CSS single-file
- JavaScript single-file
- TypeScript single-file
- HTML + CSS
- HTML + CSS + JS
- React component
- Express route
- Next.js foundation
- Browser runtime foundation
- PHP core

## Legacy tasks

Legacy tasks remain supported, but they should be considered migration candidates.

Admin UI now provides:

- legacy detection
- migration action to `ValidationContractV1`
- migration warnings

Always review the migrated contract before saving. Some legacy semantics are best-effort only.

## Authoring rules of thumb

- Put cheap static checks in `live` or `both`.
- Put runtime-heavy checks in `final`.
- Use `depends_on` to keep runtime stages behind structural ones.
- Use links for simple workspace assertions, not for deep semantic validation.
- Keep `code_structure_type` aligned with the dominant task profile, even though `code_structure` is now the main truth source.
