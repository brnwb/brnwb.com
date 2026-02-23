# AGENTS.md

## Purpose

Development notes for contributors working on this repository.

## Stack

- Static site generator implemented in Go
- JavaScript build step for the zepbound chart (`pnpm` + `esbuild`)
- Source files in `src/`
- Generated output in `html/`

## Commands

- Install JS deps: `pnpm install`
- Build JS bundles: `pnpm run build:js`
- Dev JS watcher: `pnpm run watch:js`
- Build site: `go run ./cmd/sitegen -in src -out html -clean`
- Dev site (watch + serve): `go run ./cmd/sitegen -in src -out html -watch -serve 8080`
- Test: `go test ./...`

## Source Layout

- `cmd/sitegen/`: CLI entrypoint
- `internal/build/`: main build orchestration and HTML rendering
- `internal/assets/`: CSS/JS bundling and manifest generation
- `internal/watch/`: debounced file watching
- `internal/serve/`: static HTTP serving
- `scripts/`: JS build/data generation scripts
- `src/_assets/bundles.json`: asset bundle definitions
- `src/_css/`: modular CSS inputs for `style.css` bundle
- `src/_js/`: JS bundle inputs consumed by Go asset pipeline
- `src/data/zepbound-weight.csv`: source-of-truth zepbound chart data
- `web/zepbound/`: zepbound chart source modules (including generated data module)

## Template and Asset Rules

- HTML files are template-rendered.
- Use `{{ asset "name" }}` in HTML to reference built assets from the manifest.
- CSS and JS should be built through bundle definitions, not hardcoded/CDN includes.
- Zepbound data flow: `src/data/zepbound-weight.csv` -> `scripts/generate-zepbound-data.mjs` -> `web/zepbound/weights.generated.js` -> `scripts/build-zepbound.mjs` -> `src/_js/zepbound/chart.js`.
- Do not hand-edit generated files (`web/zepbound/weights.generated.js`, `src/_js/zepbound/chart.js`).
- Keep builds deterministic and avoid timestamp-based output changes.

## Project Guardrails

- Keep Node/pnpm usage limited to zepbound JS/data bundling and keep `pnpm-lock.yaml` committed.
- Keep output paths stable unless explicitly migrating URLs.
- If adding a new bundle output, update `src/_assets/bundles.json` and reference it via `asset`.
- Ensure `pnpm run build:js`, `go test ./...`, and a full site build pass before opening a PR.
