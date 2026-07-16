import type { CapacitorConfig } from "@capacitor/cli";

function requiredBuildValue(name: "CLIENT_APPLICATION_ID" | "CLIENT_DISPLAY_NAME") {
  const value = process.env[name]?.trim();
  if (!value) {
    throw new Error(`${name} must be set before Capacitor synchronization.`);
  }
  return value;
}

const appId = requiredBuildValue("CLIENT_APPLICATION_ID");
const appName = requiredBuildValue("CLIENT_DISPLAY_NAME");

const config: CapacitorConfig = {
  appId,
  appName,
  webDir: "dist",
  server: {
    hostname: "localhost",
    androidScheme: "https",
  },
  android: {
    allowMixedContent: false,
  },
};

export default config;
