import { readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";

const version = process.env.VITE_CLIENT_BUILD_VERSION?.trim();
if (!version) throw new Error("VITE_CLIENT_BUILD_VERSION is required");

const numbers = version.match(/\d+/g)?.slice(0, 3).map(Number) ?? [0];
const versionCode = (numbers[0] ?? 0) * 1_000_000 + (numbers[1] ?? 0) * 1_000 + (numbers[2] ?? 0);
const buildGradle = resolve(import.meta.dirname, "..", "android", "app", "build.gradle");
const source = await readFile(buildGradle, "utf8");
const updated = source
  .replace(/versionCode\s+\d+/, `versionCode ${Math.max(versionCode, 1)}`)
  .replace(/versionName\s+"[^"]*"/, `versionName "${version}"`);

if (updated === source) throw new Error("Unable to update Android version metadata");
await writeFile(buildGradle, updated);
