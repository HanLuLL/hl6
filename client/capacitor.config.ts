import type { CapacitorConfig } from "@capacitor/cli";

const config: CapacitorConfig = {
  appId: process.env.CLIENT_APP_ID || "cloud.houlang.hl6",
  appName: process.env.CLIENT_APP_NAME || "HL6",
  webDir: "www",
  server: {
    androidScheme: "https",
  },
};

export default config;
