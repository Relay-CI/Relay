import { copyFile, mkdir, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import esbuild from "esbuild";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const srcDir = path.join(__dirname, "src");
const outDir = path.join(__dirname, "..", "ui");

await mkdir(outDir, { recursive: true });

await esbuild.build({
  entryPoints: [path.join(srcDir, "index.jsx")],
  bundle: true,
  entryNames: "dashboard",
  outdir: outDir,
  jsx: "automatic",
  format: "iife",
  target: ["es2020"],
  minify: false,
  sourcemap: false,
  loader: {
    ".css": "css",
  },
  define: {
    "process.env.NODE_ENV": '"production"',
  },
});

const html = await readFile(path.join(srcDir, "index.html"), "utf8");
await writeFile(path.join(outDir, "index.html"), html, "utf8");
await copyFile(path.join(srcDir, "favicon.svg"), path.join(outDir, "favicon.svg"));
