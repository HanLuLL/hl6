import { access, mkdir, readFile, rename, rm, writeFile } from "node:fs/promises";
import { basename, extname, isAbsolute, relative, resolve } from "node:path";
import sharp from "sharp";

const webRoot = resolve(import.meta.dirname, "..");
const repositoryRoot = resolve(webRoot, "..");
const androidRoot = resolve(webRoot, "android");
const drawableRoot = resolve(androidRoot, "app", "src", "main", "res", "drawable");
const valuesRoot = resolve(androidRoot, "app", "src", "main", "res", "values");
const iconOutputPath = resolve(drawableRoot, "hl6_client_icon.png");
const iconTemporaryPath = `${iconOutputPath}.tmp`;
const valuesOutputPath = resolve(valuesRoot, "hl6_client.xml");
const valuesTemporaryPath = `${valuesOutputPath}.tmp`;
const propertiesOutputPath = resolve(androidRoot, "hl6-client-build.properties");
const propertiesTemporaryPath = `${propertiesOutputPath}.tmp`;
const defaultIconPath = resolve(webRoot, "resources", "default-client-icon.svg");
const maxIconBytes = 2 * 1024 * 1024;

const required = (name) => {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`Missing required build parameter: ${name}`);
  return value;
};

const escapeXml = (value) => value
  .replace(/&/g, "&amp;")
  .replace(/</g, "&lt;")
  .replace(/>/g, "&gt;")
  .replace(/"/g, "&quot;")
  .replace(/'/g, "&apos;");

const normalizeApiBaseUrl = (value) => {
  const withScheme = /^https:\/\//i.test(value) ? value : `https://${value}`;
  const url = new URL(withScheme);
  if (url.protocol !== "https:" || url.username || url.password || url.pathname !== "/" || url.search || url.hash) {
    throw new Error("CLIENT_COMMUNICATION_DOMAIN must be an HTTPS domain without a path, credentials, query, or fragment.");
  }
  return `${url.origin}/api/v1`;
};

const detectImageType = (bytes) => {
  const isPng = bytes.length >= 8 && [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]
    .every((value, index) => bytes[index] === value);
  if (isPng) return "png";

  const isWebp = bytes.length >= 12
    && [0x52, 0x49, 0x46, 0x46].every((value, index) => bytes[index] === value)
    && [0x57, 0x45, 0x42, 0x50].every((value, index) => bytes[index + 8] === value);
  return isWebp ? "webp" : null;
};

const readRemoteIcon = async (value) => {
  let url;
  try {
    url = new URL(value);
  } catch {
    throw new Error("CLIENT_ICON_PATH must be a valid HTTPS icon URL or repository-relative file path.");
  }
  if (url.protocol !== "https:" || url.username || url.password) {
    throw new Error("CLIENT_ICON_PATH remote URLs must use HTTPS without embedded credentials.");
  }

  let response;
  try {
    response = await fetch(url, {
      headers: { Accept: "image/png,image/webp" },
      redirect: "error",
      signal: AbortSignal.timeout(15_000),
    });
  } catch {
    throw new Error("CLIENT_ICON_PATH could not be downloaded from the configured HTTPS URL.");
  }
  if (!response.ok || !response.body) throw new Error("CLIENT_ICON_PATH could not be downloaded from the configured HTTPS URL.");

  const contentLength = Number(response.headers.get("content-length"));
  if (Number.isFinite(contentLength) && contentLength > maxIconBytes) {
    throw new Error("CLIENT_ICON_PATH remote image exceeds the 2 MiB size limit.");
  }

  const chunks = [];
  let totalBytes = 0;
  for await (const chunk of response.body) {
    const bytes = chunk instanceof Uint8Array ? chunk : new Uint8Array(chunk);
    totalBytes += bytes.length;
    if (totalBytes > maxIconBytes) throw new Error("CLIENT_ICON_PATH remote image exceeds the 2 MiB size limit.");
    chunks.push(bytes);
  }

  const bytes = new Uint8Array(totalBytes);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.length;
  }
  if (!detectImageType(bytes)) throw new Error("CLIENT_ICON_PATH remote image must be a valid PNG or WebP file.");
  return { bytes, source: "remote HTTPS URL" };
};

const readRepositoryIcon = async (value) => {
  const source = resolve(repositoryRoot, value);
  const relativeSource = relative(repositoryRoot, source);
  const extension = extname(source).toLowerCase();
  if (relativeSource.startsWith("..") || isAbsolute(relativeSource) || ![".png", ".webp"].includes(extension)) {
    throw new Error("CLIENT_ICON_PATH must be a repository-relative PNG or WebP file.");
  }
  try {
    await access(source);
  } catch {
    throw new Error("CLIENT_ICON_PATH does not reference a readable repository image.");
  }

  const bytes = await readFile(source);
  if (bytes.length > maxIconBytes) throw new Error("CLIENT_ICON_PATH repository image exceeds the 2 MiB size limit.");
  const imageType = detectImageType(bytes);
  if (!imageType || `.${imageType}` !== extension) {
    throw new Error("CLIENT_ICON_PATH image content does not match its PNG or WebP extension.");
  }
  return { bytes, source: basename(source) };
};

const communicationDomain = required("CLIENT_COMMUNICATION_DOMAIN");
const communicationKey = required("CLIENT_COMMUNICATION_KEY");
const displayName = required("CLIENT_DISPLAY_NAME");
const versionName = required("CLIENT_VERSION_NAME");
const applicationId = required("CLIENT_APPLICATION_ID");

if (!/^(?:0|[1-9][0-9]{0,2})\.(?:0|[1-9][0-9]{0,2})\.(?:0|[1-9][0-9]{0,2})$/.test(versionName)) {
  throw new Error("CLIENT_VERSION_NAME must use major.minor.patch semantic versioning.");
}
if (!/^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)+$/.test(applicationId)) {
  throw new Error("CLIENT_APPLICATION_ID must be a lowercase Android package name compatible with the native OIDC redirect scheme.");
}
if (applicationId.length > 60) throw new Error("CLIENT_APPLICATION_ID is too long for the native OIDC redirect scheme.");
if (!/^[A-Za-z0-9_-]{43}$/.test(communicationKey)) {
  throw new Error("CLIENT_COMMUNICATION_KEY must be a 32-byte URL-safe key generated by HL6.");
}
if (displayName.length > 80 || /[\u0000-\u001F\u007F]/.test(displayName)) {
  throw new Error("CLIENT_DISPLAY_NAME must be at most 80 printable characters.");
}

const iconInput = process.env.CLIENT_ICON_PATH?.trim();
const icon = iconInput
  ? /^https:\/\//i.test(iconInput)
    ? await readRemoteIcon(iconInput)
    : await readRepositoryIcon(iconInput)
  : { bytes: await readFile(defaultIconPath), source: "default client icon" };
const versionParts = versionName.split(".").map(Number);
const versionCode = versionParts[0] * 1_000_000 + versionParts[1] * 1_000 + versionParts[2];
const apiBaseUrl = normalizeApiBaseUrl(communicationDomain);

try {
  await mkdir(drawableRoot, { recursive: true });
  await mkdir(valuesRoot, { recursive: true });
  await sharp(icon.bytes, { limitInputPixels: 16_000_000 })
    .resize(512, 512, { fit: "contain", background: { r: 0, g: 0, b: 0, alpha: 0 } })
    .png()
    .toFile(iconTemporaryPath);
  await writeFile(
    valuesTemporaryPath,
    `<?xml version="1.0" encoding="utf-8"?>\n<resources>\n    <string name="hl6_client_name">${escapeXml(displayName)}</string>\n</resources>\n`,
    "utf8",
  );
  await writeFile(
    propertiesTemporaryPath,
    [
      `hl6.applicationId=${applicationId}`,
      `hl6.versionCode=${versionCode}`,
      `hl6.versionName=${versionName}`,
      `hl6.apiBaseUrl=${apiBaseUrl}`,
      `hl6.nativeRedirectUri=hl6.${applicationId}://auth/callback`,
    ].join("\n") + "\n",
    "utf8",
  );

  await Promise.all([
    rm(iconOutputPath, { force: true }),
    rm(valuesOutputPath, { force: true }),
    rm(propertiesOutputPath, { force: true }),
  ]);
  await rename(iconTemporaryPath, iconOutputPath);
  await rename(valuesTemporaryPath, valuesOutputPath);
  await rename(propertiesTemporaryPath, propertiesOutputPath);
} finally {
  await Promise.all([
    rm(iconTemporaryPath, { force: true }),
    rm(valuesTemporaryPath, { force: true }),
    rm(propertiesTemporaryPath, { force: true }),
  ]);
}

console.log(`Configured Capacitor Android build: ${applicationId} ${versionName} (${versionCode}) using ${icon.source}.`);
