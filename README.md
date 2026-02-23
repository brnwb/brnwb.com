# brnwb.com

Personal blog site at [www.brnwb.com](https://www.brnwb.com), deployed to GitHub Pages.

## Local development

Run the local dev server (watch + serve):

```bash
go run ./cmd/sitegen -in src -out html -watch -serve 8080
```

This builds from `src/` into `html/` and serves on port `8080`.

## Build

Create a production build:

```bash
go run ./cmd/sitegen -in src -out html -clean
```

Output is written to `html/`.
