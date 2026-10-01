import { build } from "esbuild";
import { copyFileSync, mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(fileURLToPath(import.meta.url));
const src = join(root, "src");
const dist = join(root, "..", "dist");

mkdirSync(dist, { recursive: true });

await build({
  entryPoints: [join(src, "main.jsx")],
  bundle: true,
  format: "esm",
  jsxFactory: "h",
  jsxFragment: "Fragment",
  sourcemap: false,
  minify: true,
  target: ["es2020"],
  outfile: join(dist, "app.js"),
});

copyFileSync(join(src, "index.html"), join(dist, "index.html"));
copyFileSync(join(src, "styles.css"), join(dist, "styles.css"));
