import type { CapacitorConfig } from "@capacitor/cli";

const appId = process.env.CLIENT_APPLICATION_ID?.trim() || "cloud.houlang.hl6";
const appName = process.env.CLIENT_DISPLAY_NAME?.trim() || "HL6";

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
