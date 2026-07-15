# Android 客户端适配规范

## 核心规则

HL6 Android 通过 Capacitor 打包既有 `web/` React/Vite 生产构建。`web/src/` 是唯一 UI 源码，APK 必须携带本地 `web/dist`，禁止把线上网站作为客户端 UI 来源。

Android 集成层只可处理应用生命周期、深链、系统浏览器 OIDC、安全会话存储和 API 传输，不得维护第二套 UI，不得复制业务规则、权限决策、校验、DNS 变更、积分或服务端处理逻辑。

## 双端适配要求

1. 所有影响 Android 使用的网页、服务端、API、数据结构、字段、UI、文案、状态或文档变更，必须在同一变更集中同步更新 Capacitor 集成和本文档。
2. 新增或修改接口、字段、返回结构和状态码时，必须检查 Android 构建和可见客户端行为。
3. Android 必须继续使用同一份网页 UI 源码，包括页面、路由、组件、样式、国际化、弹窗、加载状态和错误提示；禁止以简化的 Android 专用页面替代。
4. 网页构建、Capacitor 同步、Android 签名校验、OIDC 回调、版本策略、CORS 和受影响 API 行为未核对前，迭代不得视为完成。

## 通讯与安全

1. 全局通讯密钥只能由管理后台生成、轮换和作废。服务端只保存 SHA-256 哈希，明文仅返回一次。
2. 密钥会被编译进 APK，因此它是应用构建标识，不是用户权限凭据或服务端私密密钥。不得出现在日志、URL、源代码、工作流摘要或 Release 说明中。
3. 每个原生 API 请求携带 `X-HL6-Client-Key`；受保护请求还必须携带服务端签发的 Bearer 会话。
4. 原生 Bearer 会话只能存储在 Android Keystore 支持的安全存储中，禁止使用普通 LocalStorage 或持久化日志。
5. 通讯密钥轮换或作废后，旧构建必须失效，重新登录前必须使用新密钥构建 APK。
6. 服务端必须允许 Capacitor 来源 `https://localhost`，并在接受来源时返回 `Vary: Origin`。原生变更请求的预检必须允许 `Content-Type`、`X-HL6-Client-Key` 和 `X-Idempotency-Key`。CORS 允许不代表绕过通讯密钥或会话校验。

## 构建与交付

`.github/workflows/client-build.yml` 是 Android 构建入口。它必须校验六项动态输入：通讯域名、通讯密钥、应用名称、图标、版本号和 Android 包名，以及四项 Android 签名 Secret。

1. 工作流必须构建 `web/dist`、同步 Capacitor、注入包名/名称/图标/深链配置、签名 Release APK，并通过 `apksigner verify` 校验。
2. 不得在源码中硬编码具体部署域名、通讯密钥、包名、版本、应用名称或图标。
3. 自定义图标只能使用 HTTPS PNG/WebP 地址或仓库相对 PNG/WebP 文件，必须拒绝凭据、重定向、超大文件和错误 Content-Type。
4. 临时签名材料、生成的 Android 配置、图标和网页构建产物必须在工作流结束时清理，且不得提交仓库。
5. 常规构建工作流不得创建 GitHub Release。仓库所有者已明确批准 `.github/workflows/client-build.yml` 通过 GitHub Pages 发布最新签名 APK；其他外部交付仍须单独授权并配置专用渠道。
6. GitHub Actions Artifact 下载固定为 ZIP，禁止将其描述为直接 APK 下载。获批的 Pages 渠道必须提供 `android/<applicationId>/latest.apk` 原始 HTTPS 文件和同目录 `manifest.json`，内部 Pages 传输 Artifact 仅保留 1 天。
7. Pages 每次部署只保留最新构建及其版本文件，不承诺历史 APK 存档。`manifest.json` 可以包含版本、提交和 SHA-256，但禁止包含通讯密钥、签名材料或其他凭据。

## 版本策略

1. 管理后台控制最新版本号、强制更新开关、更新公告和 HTTPS 更新链接。
2. Android 启动时通过通讯密钥请求 `/api/v1/client/version?current_version=<versionName>`。
3. 普通更新可关闭提示继续使用；强制更新必须阻止进入业务页面，直至存在有效 HTTPS 更新链接。
4. 更新链接必须指向经明确批准、可信的 HTTPS APK 分发地址。当前默认交付地址使用 `https://hanlull.github.io/hl6/android/<applicationId>/latest.apk`，不创建 GitHub Release。

## OIDC

1. OIDC 提供商仅登记 `https://<domain>/api/v1/auth/callback`，不得将 Android 深链登记为提供商回调。
2. 原生登录开始和授权码交换接口均需要通讯密钥；密钥绝不能进入浏览器 URL、OIDC 参数或深链。
3. 服务端创建短期、一次性的授权码，并跳转至 `hl6.<applicationId>://auth/callback?code=...`。
4. Capacitor App 插件接收深链后，客户端交换授权码并保存安全 Bearer 会话。
5. 原生 `Failed to fetch` 首先按域名、网络和 CORS 部署状态排查，预检响应必须包含 `Access-Control-Allow-Origin: https://localhost`。

## 变更记录

| 日期 | 变更 |
| --- | --- |
| 2026-07-15 | 使用 Capacitor 打包既有 React/Vite UI，Android 与网页端复用同一份 UI、路由、国际化和 API 客户端源码。 |
| 2026-07-15 | 增加通讯密钥鉴权、Android Keystore 安全会话存储、OIDC 深链交换和服务端控制的版本策略。 |
| 2026-07-15 | 更新 Android 构建规则：只构建、签名和校验 APK；未经仓库所有者明确授权，不创建 GitHub Release 或发布 APK。 |
| 2026-07-15 | 记录 Capacitor CORS 部署要求，并将原生网络失败统一转换为可操作的客户端错误提示。 |
| 2026-07-15 | 修复原生登录 CORS 预检：服务端允许 `X-Idempotency-Key`，与客户端所有非安全方法请求自动携带的请求头保持一致。 |
| 2026-07-15 | 经仓库所有者确认，客户端工作流在签名校验后通过 GitHub Pages 发布最新 APK、稳定 `latest.apk` 和无敏感信息的 `manifest.json`，不创建 GitHub Release。 |

所有客户端相关代码、接口、UI、工作流、安全或文档更新都必须在此表追加带日期的记录。
