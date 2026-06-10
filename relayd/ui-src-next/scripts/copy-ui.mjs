// Copy the static Next export into relayd/ui while dropping macOS metadata
// sidecars. USB/FAT-style workspaces can materialize AppleDouble files under
// several names, and embedding them bloats the daemon binary with junk.
import { copyFileSync, existsSync, mkdirSync, readFileSync, readdirSync, rmSync, statSync } from "node:fs";
import { dirname, join } from "node:path";

const srcRoot = process.argv[2] || "out";
const dstRoot = process.argv[3] || "../ui";
const APPLEDOUBLE_MAGIC = Buffer.from([0x00, 0x05, 0x16, 0x07]);

function hasAppleDoubleName(name) {
  return name.startsWith("._") || name.startsWith(".__");
}

function isAppleDoubleFile(path) {
  try {
    const buf = readFileSync(path, { encoding: null, flag: "r" });
    return buf.length >= APPLEDOUBLE_MAGIC.length && buf.subarray(0, APPLEDOUBLE_MAGIC.length).equals(APPLEDOUBLE_MAGIC);
  } catch {
    return false;
  }
}

function copyTree(src, dst) {
  const st = statSync(src);
  if (st.isDirectory()) {
    mkdirSync(dst, { recursive: true });
    for (const name of readdirSync(src)) {
      if (hasAppleDoubleName(name)) continue;
      copyTree(join(src, name), join(dst, name));
    }
    return;
  }
  if (isAppleDoubleFile(src)) return;
  mkdirSync(dirname(dst), { recursive: true });
  copyFileSync(src, dst);
}

if (existsSync(dstRoot)) {
  rmSync(dstRoot, { recursive: true });
}
copyTree(srcRoot, dstRoot);
console.log(`Copied ${srcRoot}/ to ${dstRoot}/`);
