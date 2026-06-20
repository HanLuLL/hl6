import { execSync } from "node:child_process"
import path from "node:path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig, loadEnv } from "vite"

const projectRoot = path.resolve(__dirname, "..")

function readGitValue(command: string): string | null {
  try {
    const value = execSync(command, {
      cwd: projectRoot,
      stdio: ["ignore", "pipe", "ignore"],
    })
      .toString()
      .trim()
    return value || null
  } catch {
    return null
  }
}

function readEnvValue(keys: string[]): string | null {
  for (const key of keys) {
    const value = process.env[key]?.trim()
    if (value) {
      return value
    }
  }
  return null
}

const gitBranch =
  readEnvValue(["APP_GIT_BRANCH", "CI_COMMIT_REF_NAME", "CI_COMMIT_BRANCH"]) ??
  readGitValue("git rev-parse --abbrev-ref HEAD") ??
  "unknown"

const gitCommit =
  readEnvValue(["APP_GIT_COMMIT", "CI_COMMIT_SHORT_SHA"]) ??
  readEnvValue(["CI_COMMIT_SHA"])?.slice(0, 8) ??
  readGitValue("git rev-parse --short HEAD") ??
  "unknown"

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, projectRoot, "")
  const serverPort = env.SERVER_PORT || "8081"
  const devPort = Number(env.VITE_DEV_PORT || "5174")

  return {
  envDir: projectRoot,
  define: {
    __APP_GIT_BRANCH__: JSON.stringify(gitBranch),
    __APP_GIT_COMMIT__: JSON.stringify(gitCommit),
  },
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: devPort,
    strictPort: true,
    proxy: {
      "/api": `http://localhost:${serverPort}`,
    },
  },
  }
})
