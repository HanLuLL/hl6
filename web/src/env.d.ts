/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL?: string
  readonly VITE_CAPACITOR_APP?: string
  readonly VITE_CLIENT_COMMUNICATION_KEY?: string
  readonly VITE_CLIENT_VERSION?: string
  readonly VITE_NATIVE_REDIRECT_URI?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare const __APP_GIT_BRANCH__: string
declare const __APP_GIT_COMMIT__: string
