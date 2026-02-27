/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare const __APP_GIT_BRANCH__: string
declare const __APP_GIT_COMMIT__: string
