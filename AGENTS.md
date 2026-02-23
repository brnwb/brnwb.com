# AGENTS.md

## Purpose

Development notes for contributors working on this repository.

## Stack

- Static site generator implemented in Go
- Source files in `src/`
- Generated output in `html/`

## Commands

- Build: `go run ./cmd/sitegen -in src -out html -clean`
- Dev (watch + serve): `go run ./cmd/sitegen -in src -out html -watch -serve 8080`
- Test: `go test ./...`

## Source Layout

- `cmd/sitegen/`: CLI entrypoint
- `internal/build/`: main build orchestration and HTML rendering
- `internal/assets/`: CSS/JS bundling and manifest generation
- `internal/watch/`: debounced file watching
- `internal/serve/`: static HTTP serving
- `src/_assets/bundles.json`: asset bundle definitions
- `src/_css/`: modular CSS inputs for `style.css` bundle
- `src/_js/`: modular JS inputs for future bundles

## Template and Asset Rules

- HTML files are template-rendered.
- Use `{{ asset "name" }}` in HTML to reference built assets from the manifest.
- CSS and JS should be built through bundle definitions, not template includes.
- Keep builds deterministic and avoid timestamp-based output changes.

## Project Guardrails

- Do not reintroduce Node/pnpm as required build tooling.
- Keep output paths stable unless explicitly migrating URLs.
- If adding a new bundle output, update `src/_assets/bundles.json` and reference it via `asset`.
- Ensure `go test ./...` and a full build pass before opening a PR.

