import { access, mkdir, readFile, rename, rm, writeFile } from "node:fs/promises";
import { basename, extname, isAbsolute, relative, resolve } from "node:path";

const clientRoot = resolve(import.meta.dirname, "..");
const projectRoot = resolve(clientRoot, "..");
const drawableRoot = resolve(clientRoot, "app", "src", "main", "res", "drawable");
const localPropertiesPath = resolve(clientRoot, "local.properties");
const temporaryPropertiesPath = resolve(clientRoot, "local.properties.tmp");
const customIconPaths = [
  resolve(drawableRoot, "client_icon_custom.png"),
  resolve(drawableRoot, "client_icon_custom.webp"),
];
const maxIconBytes = 2 * 1024 * 1024;

const required = (name) => {
  const value = process.env[name]?.trim();
  if (!value) throw new Error(`Missing required build parameter: ${name}`);
  return value;
};

const escapeProperty = (value) => value.replace(/\\/g, "\\\\").replace(/\n/g, "\\n").replace(/\r/g, "\\r");
const supportedIconTypes = new Map([
  [".png", "png"],
  [".webp", "webp"],
]);
const normalizeApiBaseUrl = (value) => {
  const withScheme = /^https:\/\//i.test(value) ? value : `https://${value}`;
  const url = new URL(withScheme);
  if (url.protocol !== "https:" || url.username || url.password || url.pathname !== "/" || url.search || url.hash) {
    throw new Error("CLIENT_COMMUNICATION_DOMAIN must be an HTTPS domain without a path, credentials, query, or fragment.");
  }
  return `${url.origin}/api/v1`;
};
const iconTypeFromBytes = (bytes) => {
  const isPng = bytes.length >= 8 && [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a].every((value, index) => bytes[index] === value);
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

  const response = await fetch(url, {
    headers: { Accept: "image/png,image/webp" },
    redirect: "error",
    signal: AbortSignal.timeout(15_000),
  });
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

  const iconBytes = new Uint8Array(totalBytes);
  let offset = 0;
  for (const chunk of chunks) {
    iconBytes.set(chunk, offset);
    offset += chunk.length;
  }
  const iconType = iconTypeFromBytes(iconBytes);
  if (!iconType) throw new Error("CLIENT_ICON_PATH remote image must be a valid PNG or WebP file.");
  return { bytes: iconBytes, type: iconType, source: "remote HTTPS URL" };
};
const readRepositoryIcon = async (value) => {
  const source = resolve(projectRoot, value);
  const relativeSource = relative(projectRoot, source);
  const expectedType = supportedIconTypes.get(extname(source).toLowerCase());
  if (relativeSource.startsWith("..") || isAbsolute(relativeSource) || !expectedType) {
    throw new Error("CLIENT_ICON_PATH must be a repository-relative PNG or WebP file.");
  }
  try {
    await access(source);
  } catch {
    throw new Error("CLIENT_ICON_PATH does not reference a readable repository image.");
  }

  const iconBytes = await readFile(source);
  if (iconBytes.length > maxIconBytes) throw new Error("CLIENT_ICON_PATH repository image exceeds the 2 MiB size limit.");
  const iconType = iconTypeFromBytes(iconBytes);
  if (iconType !== expectedType) throw new Error("CLIENT_ICON_PATH image content does not match its PNG or WebP extension.");
  return { bytes: iconBytes, type: iconType, source: basename(source) };
};

const communicationDomain = required("CLIENT_COMMUNICATION_DOMAIN");
const communicationKey = required("CLIENT_COMMUNICATION_KEY");
const displayName = required("CLIENT_DISPLAY_NAME");
const versionName = required("CLIENT_VERSION_NAME");
const applicationId = required("CLIENT_APPLICATION_ID");
const keystoreFile = required("CLIENT_KEYSTORE_FILE");
const keystorePassword = required("CLIENT_KEYSTORE_PASSWORD");
const keystoreType = required("CLIENT_KEYSTORE_TYPE");
const keyAlias = required("CLIENT_KEY_ALIAS");
const keyPassword = required("CLIENT_KEY_PASSWORD");

if (!/^(?:0|[1-9][0-9]{0,2})\.(?:0|[1-9][0-9]{0,2})\.(?:0|[1-9][0-9]{0,2})$/.test(versionName)) {
  throw new Error("CLIENT_VERSION_NAME must use major.minor.patch semantic versioning.");
}
if (!/^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$/.test(applicationId)) throw new Error("CLIENT_APPLICATION_ID must be a valid lowercase Android package name.");
if (applicationId.length > 60) throw new Error("CLIENT_APPLICATION_ID is too long for the native OIDC redirect scheme.");
if (!/^[A-Za-z0-9_-]{43}$/.test(communicationKey)) throw new Error("CLIENT_COMMUNICATION_KEY must be a 32-byte URL-safe key generated by HL6.");
if (displayName.length > 80 || /[\u0000-\u001F\u007F]/.test(displayName)) throw new Error("CLIENT_DISPLAY_NAME must be at most 80 printable characters.");
if (!/^[A-Za-z0-9._-]+$/.test(keystoreType)) throw new Error("CLIENT_KEYSTORE_TYPE must be a valid Java keystore type.");

const versionParts = (versionName.match(/\d+/g) ?? []).slice(0, 3).map(Number);
const versionCode = Math.max(1, (versionParts[0] ?? 0) * 1_000_000 + (versionParts[1] ?? 0) * 1_000 + (versionParts[2] ?? 0));
const iconInput = process.env.CLIENT_ICON_PATH?.trim();
const icon = iconInput
  ? /^https:\/\//i.test(iconInput)
    ? await readRemoteIcon(iconInput)
    : await readRepositoryIcon(iconInput)
  : null;
const customIconPath = icon ? resolve(drawableRoot, `client_icon_custom.${icon.type}`) : null;
const temporaryCustomIconPath = customIconPath ? `${customIconPath}.tmp` : null;
const properties = {
  "client.applicationId": applicationId,
  "client.versionCode": String(versionCode),
  "client.versionName": versionName,
  "client.displayName": displayName,
  "client.apiBaseUrl": normalizeApiBaseUrl(communicationDomain),
  "client.communicationKey": communicationKey,
  "client.iconResource": icon ? "@drawable/client_icon_custom" : "@drawable/client_icon",
  "client.nativeRedirectUri": `hl6.${applicationId}://auth/callback`,
  "client.keystoreFile": keystoreFile,
  "client.keystorePassword": keystorePassword,
  "client.keystoreType": keystoreType,
  "client.keyAlias": keyAlias,
  "client.keyPassword": keyPassword,
};

const propertyContents = `${Object.entries(properties).map(([key, value]) => `${key}=${escapeProperty(value)}`).join("\n")}\n`;

try {
  await writeFile(temporaryPropertiesPath, propertyContents, "utf8");
  if (icon && temporaryCustomIconPath) {
    await mkdir(drawableRoot, { recursive: true });
    await writeFile(temporaryCustomIconPath, icon.bytes);
  }

  await Promise.all(customIconPaths.map((path) => rm(path, { force: true })));
  if (customIconPath && temporaryCustomIconPath) await rename(temporaryCustomIconPath, customIconPath);
  await rm(localPropertiesPath, { force: true });
  await rename(temporaryPropertiesPath, localPropertiesPath);

  if (icon) {
    console.log(`Applied native Android ${icon.type.toUpperCase()} icon from ${icon.source}.`);
  }
} finally {
  await Promise.all([
    rm(temporaryPropertiesPath, { force: true }),
    ...(temporaryCustomIconPath ? [rm(temporaryCustomIconPath, { force: true })] : []),
  ]);
}

console.log(`Configured native Android build: ${applicationId} ${versionName} (${versionCode}).`);
