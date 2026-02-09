# brnwb.com

Personal blog site at [www.brnwb.com](https://www.brnwb.com), built with [blargh](https://github.com/badlogic/blargh) and deployed to GitHub Pages.

## Local development

Install dependencies:

```bash
pnpm install
```

Run the local dev server (watch mode):

```bash
pnpm run dev
```

This builds from `src/` into `html/` and serves on port `8080`.

## Build

Create a production build:

```bash
pnpm run build
```

Output is written to `html/`.
