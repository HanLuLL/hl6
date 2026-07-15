# Android 客户端适配规范

## 核心定位

HL6 Android 客户端由 Capacitor 将 `web/` 的 React/Vite 生产构建产物打包进 APK。客户端 UI、路由、组件、国际化、状态展示和 API 调用必须直接复用 `web/src/`，不维护第二套 Kotlin、Compose、Vue 或 React Native 页面。

APK 必须包含本地构建的 `web/dist`，禁止在启动时加载远程网页作为 UI 来源。这样客户端与网页端使用同一份 DOM、CSS、组件和业务调用，实现视觉与功能同步。Android 容器只承担应用启动、深链、系统浏览器、Android Keystore 安全存储和 APK 签名能力。

## 双端一致性

1. `web/` 是唯一 UI 源码。客户端不得复制、简化或替换网页页面、组件、样式、路由、国际化文案和交互。
2. 所有影响网页 UI 或 API 使用方式的变更，必须检查 Capacitor 构建是否仍可用，并在同一变更中更新 Android 配置、文档和本文件记录。
3. Android 只允许维护启动容器、深链、签名、图标和安全存储等平台层代码；不得出现独立业务 UI。
4. 每次客户端迭代必须验证网页构建、Capacitor 同步、Android Release 编译、签名校验、OIDC 回调、版本更新和主要 API 的兼容性。

## 通讯密钥与会话

1. 通讯密钥只能由管理后台生成、轮换或作废；服务端仅保存其 SHA-256 哈希，明文只返回一次。
2. 客户端构建时将通讯密钥注入 Vite 环境。它会存在于 APK 中，因此它是应用识别凭据，不得被当作用户权限或保密的服务端密钥。
3. Capacitor 客户端的每个 API 请求必须携带 `X-HL6-Client-Key`。受保护请求还必须携带服务端签发的 Bearer 会话令牌。
4. 会话令牌使用 `capacitor-secure-storage-plugin` 存储在 Android Keystore 支持的安全存储中；禁止写入普通 LocalStorage、日志或仓库文件。
5. 服务端作废通讯密钥后，旧客户端的版本检查和受保护 API 请求必须失败，并清除本地会话、要求重新构建和登录。
6. 构建流程、图标下载、GitHub 日志和 Release 说明不得输出通讯密钥、keystore、密码或用户令牌。

## 构建与发布

`.github/workflows/client-build.yml` 是唯一 Android 客户端构建入口，支持以下动态输入：通讯域名、通讯密钥、客户端名称、PNG/WebP 图标、版本号和 Android 包名。

1. 工作流必须验证全部输入和 Android 签名 Secrets：`ANDROID_KEYSTORE_BASE64`、`ANDROID_KEYSTORE_PASSWORD`、`ANDROID_KEY_ALIAS`、`ANDROID_KEY_PASSWORD`。
2. 工作流必须先构建 `web/dist`，再执行 `npx cap sync android`，确保 APK 使用本次提交的网页 UI。
3. 自定义图标只允许 HTTPS PNG/WebP URL 或仓库相对路径；必须拒绝凭据、重定向、超过 2 MiB 的内容和不匹配的文件签名。
4. 包名、版本号、应用名称、深链和图标必须在构建时生成，禁止硬编码具体部署实例。
5. 工作流必须生成已签名 Release APK 并执行 `apksigner verify`。
6. 工作流不得使用 GitHub Actions Artifact 作为交付物。必须将 `.apk` 作为 GitHub Release 资产上传，用户下载到的是直接 APK，不是 ZIP。
7. 临时 keystore、构建配置、图标和网页构建结果必须在工作流结束时清理，且不得提交到仓库。

## 版本与更新

1. 管理后台维护最新客户端版本、强制更新开关、更新公告和 HTTPS 更新链接。
2. 客户端每次启动使用通讯密钥请求 `/api/v1/client/version?current_version=<versionName>`；版本比较只由服务端完成。
3. 普通更新可关闭提示继续使用；强制更新弹窗不可关闭，且不得进入业务页面。
4. 更新链接必须指向工作流发布的 GitHub Release APK 或其他可信 HTTPS APK 地址。

## OIDC 适配

1. OIDC 提供商只配置服务端 HTTPS 回调：`https://<domain>/api/v1/auth/callback`，不配置 Android 深链。
2. 客户端调用受通讯密钥保护的 `/api/v1/auth/native/start`，在系统浏览器完成 OIDC 登录。通讯密钥不得进入浏览器 URL、OIDC 参数或深链。
3. 服务端登录成功后跳转至 `hl6.<applicationId>://auth/callback?code=...`。该代码只能使用一次且有效期为 90 秒。
4. Capacitor 的 `App` 插件接收深链，客户端带通讯密钥调用 `/api/v1/auth/native/exchange` 换取 Bearer 会话令牌。
5. 常见故障：深链不能唤醒通常是包名与 APK 不一致；`invalid native auth code` 通常表示代码过期或重复消费；`invalid client key` 表示密钥已轮换，需要重新构建 APK。

## 变更记录

| 日期 | 变更 |
| --- | --- |
| 2026-07-15 | 放弃 Kotlin + Jetpack Compose 双 UI 方案，改为 Capacitor 打包 `web/` React/Vite 构建产物，客户端 UI 和功能直接复用网页端源码。 |
| 2026-07-15 | 新增 Capacitor 系统浏览器 OIDC 深链回调、Android Keystore 支持的会话存储、每个 API 请求的通讯密钥与 Bearer 会话注入，以及原生版本更新提示。 |
| 2026-07-15 | 构建工作流改为构建网页前端、同步 Android 容器、签名校验 APK，并以 GitHub Release 直接 APK 资产发布，不再上传 ZIP Artifact。 |
| 2026-07-15 | 收紧 Android 包名校验以确保生成合法 OIDC 深链，并为启动版本检查设置超时，避免离线网络无限阻塞客户端启动。 |
| 2026-07-15 | 图标下载失败改为输出稳定且不泄露网络细节的构建错误，便于工作流排查。 |
| 2026-07-15 | 构建工作流直接调用项目安装的 Capacitor CLI，避免跨平台 `pnpm exec` 命令解析差异。 |
| 2026-07-15 | Capacitor 同步阶段也改为读取工作流注入的应用名称与包名，避免生成配置保留部署实例默认值。 |
| 2026-07-15 | GitHub Linux runner 在调用 Gradle wrapper 前必须显式恢复 `gradlew` 执行权限，避免 Windows 提交丢失 Unix 文件模式导致 APK 构建失败。 |

所有客户端相关代码、接口、工作流和文档变更必须在本文件追加记录。
