import { watch } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import * as esbuild from "esbuild";

import { generateZepboundData } from "./generate-zepbound-data.mjs";

const thisFile = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(thisFile), "..");

const watchMode = process.argv.includes("--watch");
const csvPath = path.join(repoRoot, "src", "data", "zepbound-weight.csv");

const buildOptions = {
  entryPoints: [path.join(repoRoot, "web", "zepbound", "chart.js")],
  outfile: path.join(repoRoot, "src", "_js", "zepbound", "chart.js"),
  bundle: true,
  format: "iife",
  platform: "browser",
  target: "es2020",
  legalComments: "none",
  minify: !watchMode,
  sourcemap: watchMode ? "inline" : false,
  logLevel: "info",
};

async function runBuild() {
  const generated = await generateZepboundData();
  console.log(`data: generated ${generated.count} rows`);

  if (!watchMode) {
    await esbuild.build(buildOptions);
    console.log("js: built src/_js/zepbound/chart.js");
    return;
  }

  const ctx = await esbuild.context(buildOptions);
  await ctx.watch();
  console.log("js: watching zepbound chart sources");
  console.log(`js: watching ${path.relative(repoRoot, csvPath)}`);

  let pending = false;
  let timer = null;

  const regenerate = async () => {
    if (pending) {
      return;
    }
    pending = true;

    try {
      const regenerated = await generateZepboundData();
      console.log(`data: regenerated ${regenerated.count} rows`);
    } catch (err) {
      console.error("data: regeneration failed");
      console.error(err);
    } finally {
      pending = false;
    }
  };

  const scheduleRegeneration = () => {
    if (timer !== null) {
      clearTimeout(timer);
    }
    timer = setTimeout(() => {
      timer = null;
      void regenerate();
    }, 100);
  };

  const csvWatcher = watch(csvPath, { persistent: true }, () => {
    scheduleRegeneration();
  });

  const shutdown = async () => {
    csvWatcher.close();
    await ctx.dispose();
  };

  process.on("SIGINT", () => {
    void shutdown().finally(() => process.exit(0));
  });
  process.on("SIGTERM", () => {
    void shutdown().finally(() => process.exit(0));
  });
}

runBuild().catch((err) => {
  console.error("js: build failed");
  console.error(err);
  process.exit(1);
});
