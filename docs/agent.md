# Android 客户端适配规范

本文件是 HL6 网页、服务端和 Android 客户端同步演进的强制约束。任何影响 Android 展示、登录、API、数据字段、版本或发布的改动，未完成本文件要求即视为迭代不完整。

## 1. 架构红线

1. Android 使用 Capacitor 打包本仓库 `web/` React/Vite 生产构建，复用网页端同一份 UI 源码。
2. 禁止把远程网站作为客户端 UI 来源，禁止新增第二套 Kotlin 业务页面或用原生层复制网页功能。
3. Android 层只处理生命周期、渲染容器、系统浏览器、深链、安全存储、网络传输和版本提示。
4. 权限、资源归属、DNS、积分、支付、审核和数据校验全部由服务端执行。

## 2. 同步适配

以下改动必须在同一变更集中检查并适配 Android：

- 页面、路由、组件、样式、图标、文案、国际化和交互。
- API 路径、方法、字段、响应结构、错误码和状态码。
- OIDC、会话、资料、封禁、申诉和通知。
- DNS、积分、支付、审核和管理后台的客户端可见行为。
- 客户端版本、更新链接、强制更新和发布流程。

网页 UI 变更通过共享源码自然进入 APK，但仍需验证移动布局、系统浏览器返回、键盘/状态栏和原生网络环境。

## 3. UI 一致性

- Android 与网页使用同一 React 路由和组件，不维护降级页面。
- 颜色、字体、间距、图标、弹窗、Toast、加载、空状态和错误状态保持一致。
- Android 屏幕尺寸下不得出现文本溢出、不可滚动弹窗或被系统栏遮挡的操作。
- 所有六种语言在网页和 Android 使用同一资源。
- 用户资料使用服务端持久化值；重新登录不得清空姓名或覆盖自定义头像。
- 封禁页面和邮件都显示封禁开始时间、预计解封时间或永久封禁说明。

## 4. 通讯与会话安全

1. 通讯密钥只能由管理后台生成、轮换和作废。
2. 服务端只保存 SHA-256；明文只返回一次，不进入仓库、URL、日志、摘要或 Release Notes。
3. 密钥会编译进 APK，只是客户端构建标识，不是用户凭据或管理员权限。
4. 每个原生请求携带 `X-HL6-Client-Key`；登录请求同时使用原生授权流程，受保护请求携带 Bearer 会话。
5. 非安全方法携带 `X-Idempotency-Key`；业务重试复用原键。
6. 会话只保存到 Android Keystore 支持的安全存储，不写入普通 LocalStorage 或持久日志。
7. 轮换密钥时先构建并发布新 APK，再作废旧密钥。

## 5. CORS

Capacitor 本地来源为 `https://localhost`。服务端必须：

```text
Access-Control-Allow-Origin: https://localhost
Access-Control-Allow-Headers: Origin, Content-Type, Accept, Authorization, X-HL6-Client-Key, X-Idempotency-Key
Vary: Origin
```

CORS 只允许浏览器传输，不得放松通讯密钥、会话或资源权限。

## 6. OIDC

1. 身份提供商只登记 `https://<domain>/api/v1/auth/callback`。
2. Android 调用 `/auth/native/start` 并在系统浏览器登录。
3. 服务端回跳 `hl6.<applicationId>://auth/callback?code=...`。
4. 客户端调用 `/auth/native/exchange` 交换短期一次性授权码。
5. 通讯密钥不进入 OIDC 参数、State、查询字符串或深链。
6. 包名变化必须同步 Manifest、深链、构建输入和文档。

## 7. 构建

`.github/workflows/client-build.yml` 是唯一 Android CI 构建入口，必须动态校验并注入：

- 后端 HTTPS 域名。
- 通讯密钥。
- 客户端显示名称。
- PNG/WebP 图标。
- `versionName`。
- `applicationId`。

签名 Secret：

- `ANDROID_KEYSTORE_BASE64`
- `ANDROID_KEYSTORE_PASSWORD`
- `ANDROID_KEY_ALIAS`
- `ANDROID_KEY_PASSWORD`

工作流必须完成网页生产构建、Capacitor 同步、动态配置、release 签名和 `apksigner verify`。临时 keystore、属性文件、图标和构建产物无论成功失败都要清理。

## 8. 分发与正式 Release

成功客户端构建发布：

```text
/android/<applicationId>/latest.apk
/android/<applicationId>/<applicationId>-<version>.apk
/android/<applicationId>/manifest.json
```

Manifest 只包含版本、包名、提交、时间和 SHA-256，不包含密钥。Pages 只保证最新部署；正式系统版本必须将验证后的版本 APK 附加到 GitHub Release。

正式 Release 只能人工触发，必须先校验 Manifest 版本、包名和 SHA-256，再创建标签、Release 和 Docker 版本镜像。普通提交不得自动创建 Release。

## 9. 版本策略

客户端启动请求 `/api/v1/client/version?current_version=<versionName>`。服务端控制：

- 最新版本。
- 普通或强制更新。
- 更新公告。
- HTTPS 下载链接。

普通更新可关闭；强制更新必须阻止业务入口。提高强制版本前，必须确认 Pages/Release APK 可下载、摘要正确且使用同一签名。

## 10. 每次迭代检查

- [ ] Android 使用更新后的共享 UI，未新增远程 UI 或第二套业务页面。
- [ ] API 路径、字段、错误和 TypeScript 类型同步。
- [ ] 移动布局、表单、弹窗、Toast 和六种语言正常。
- [ ] CORS 包含原生来源和全部自定义头。
- [ ] OIDC 登录、深链、交换、登出和会话恢复正常。
- [ ] 用户资料与头像重新登录后保持。
- [ ] 封禁时间在页面和邮件中完整显示。
- [ ] 普通/强制更新和下载链接正常。
- [ ] APK 签名、包名、版本和 Manifest SHA-256 已验证。
- [ ] `docs/android-client.md`、`docs/oidc.md`、`docs/api.md` 和本变更记录已按影响更新。

## 11. 变更记录

只记录客户端架构、协议、用户可见功能或交付规则变化，不记录每个小修复。

| 日期 | 变更 |
| --- | --- |
| 2026-07-15 | Android 改为 Capacitor 打包现有 React/Vite UI，网页与客户端共享页面、路由、国际化和 API 源码。 |
| 2026-07-15 | 建立通讯密钥、Keystore 会话、原生 OIDC 授权码交换、版本检查和强制更新机制。 |
| 2026-07-15 | 建立 GitHub Actions 签名构建、Pages 直接 APK 分发、Manifest 摘要和正式 Release APK 归档。 |
| 2026-07-15 | 重构客户端、OIDC、API、部署和运维文档，正式版本只记录稳定能力。 |
