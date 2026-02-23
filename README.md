# brnwb.com

Personal blog site at [www.brnwb.com](https://www.brnwb.com), deployed to GitHub Pages.

## Local development

Install JavaScript build dependencies:

```bash
pnpm install
```

Run the zepbound JS watcher in one terminal:

```bash
pnpm run watch:js
```

Run the local site dev server (watch + serve) in another terminal:

```bash
go run ./cmd/sitegen -in src -out html -watch -serve 8080
```

This builds from `src/` into `html/` and serves on `http://localhost:8080/`.
Zepbound source data lives in `src/data/zepbound-weight.csv`.

## Build

Create a production build:

```bash
pnpm run build:js
go run ./cmd/sitegen -in src -out html -clean
```

Output is written to `html/`.
