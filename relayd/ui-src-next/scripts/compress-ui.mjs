// Emit precompressed (.br/.gz) siblings for dashboard assets. relayd serves
// these when the browser advertises support, cutting transfer size ~70% with
// zero runtime CPU cost on the host.
import { readdirSync, statSync, readFileSync, writeFileSync } from "node:fs";
import { join, extname } from "node:path";
import { gzipSync, brotliCompressSync, constants } from "node:zlib";

const root = process.argv[2] || "../ui";
const exts = new Set([".js", ".css", ".html", ".svg", ".json", ".txt", ".xml", ".map"]);
const APPLEDOUBLE_MAGIC = Buffer.from([0x00, 0x05, 0x16, 0x07]);

let count = 0;
let before = 0;
let after = 0;

function walk(dir) {
  for (const name of readdirSync(dir)) {
    if (name.startsWith("._") || name.startsWith(".__")) continue; // macOS AppleDouble junk
    const p = join(dir, name);
    const st = statSync(p);
    if (st.isDirectory()) {
      walk(p);
      continue;
    }
    const ext = extname(name);
    if (!exts.has(ext)) continue;
    const buf = readFileSync(p);
    if (buf.length >= APPLEDOUBLE_MAGIC.length && buf.subarray(0, APPLEDOUBLE_MAGIC.length).equals(APPLEDOUBLE_MAGIC)) {
      continue;
    }
    if (buf.length < 1024) continue; // not worth the embed overhead
    const br = brotliCompressSync(buf, {
      params: { [constants.BROTLI_PARAM_QUALITY]: 10 },
    });
    const gz = gzipSync(buf, { level: 9 });
    if (br.length < buf.length) writeFileSync(`${p}.br`, br);
    if (gz.length < buf.length) writeFileSync(`${p}.gz`, gz);
    count += 1;
    before += buf.length;
    after += Math.min(br.length, buf.length);
  }
}

walk(root);
console.log(
  `compress-ui: ${count} assets, ${Math.round(before / 1024)} KB -> ${Math.round(after / 1024)} KB (brotli)`,
);
