import { cp, mkdir, rm } from "node:fs/promises";
import { resolve } from "node:path";

const clientRoot = resolve(import.meta.dirname, "..");
const webDist = resolve(clientRoot, "..", "web", "dist");
const webDir = resolve(clientRoot, "www");

await rm(webDir, { recursive: true, force: true });
await mkdir(webDir, { recursive: true });
await cp(webDist, webDir, { recursive: true });
