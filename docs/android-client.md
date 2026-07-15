# Android 客户端构建与发布

HL6 Android 客户端使用 Capacitor 将现有 `web/` React/Vite 生产构建打包进 APK。网页端与 Android 使用同一份页面、路由、组件、样式、国际化和 API 客户端源码；客户端不会把线上站点作为远程 UI 加载，也没有第二套 Kotlin 业务界面。

## 1. 职责边界

Android 平台层只负责：

- 应用启动和生命周期。
- 本地打包 UI 的渲染容器。
- 系统浏览器 OIDC 与深链。
- Android Keystore 支持的会话存储。
- API 传输、网络状态和版本提示。

域名、DNS、积分、支付、审核、用户权限和所有业务校验由服务端决定。客户端不得复制服务端权限规则或在本地离线执行 DNS 操作。

## 2. 通讯密钥

在 **管理后台 > 客户端管理** 中生成通讯密钥。服务端只保存 SHA-256 哈希，明文只返回一次。

通讯密钥会编译进 APK，用于识别获准客户端构建。它不能替代用户会话或管理员权限，也不应被视为无法提取的服务端 Secret。每个原生 API 请求携带 `X-HL6-Client-Key`；登录后请求还必须携带服务端会话令牌。

轮换顺序：

1. 生成新通讯密钥并安全保存。
2. 使用新密钥构建、签名和验证新 APK。
3. 发布新 APK，并按版本策略通知用户。
4. 确认迁移窗口后作废旧密钥。

提前作废会立即使所有旧 APK 的原生请求失败。

## 3. GitHub Actions 输入

手动运行 **Build Web Android Client**：

| 输入 | 规则 |
| --- | --- |
| `communication_domain` | HTTPS 后端域名，不含路径，例如 `domain.example.com` |
| `communication_key` | 后台生成的 43 字符 URL-safe 密钥 |
| `client_name` | Android 显示名称 |
| `client_icon` | 可选 HTTPS PNG/WebP URL 或仓库相对 PNG/WebP 文件 |
| `version` | `major.minor.patch`，例如 `1.0.0` |
| `android_package_name` | 小写 applicationId，例如 `cc.lxii.domain` |

所有值在编译期注入，无具体部署域名、密钥、包名、名称、图标或版本硬编码在源码中。

当前正式客户端：

```text
Name: 林域
Version: 1.0.0
Package: cc.lxii.domain
Deep link: hl6.cc.lxii.domain://auth/callback
```

## 4. 签名 Secrets

在仓库 **Settings > Secrets and variables > Actions** 配置：

- `ANDROID_KEYSTORE_BASE64`
- `ANDROID_KEYSTORE_PASSWORD`
- `ANDROID_KEY_ALIAS`
- `ANDROID_KEY_PASSWORD`

PowerShell 编码 keystore：

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes('.\hl6-release.keystore'))
```

同一包名的所有更新必须使用同一 release keystore。丢失 keystore 或更换签名后，Android 无法覆盖安装旧版本，只能使用新包名重新发布。

工作流会验证 keystore 密码、唯一私钥条目和别名；若配置别名与唯一私钥条目不一致，会使用检测到的别名并给出警告。

## 5. 构建流程

工作流依次执行：

1. 校验六项输入和四项签名 Secret。
2. 使用 Node.js 22、pnpm 11.7.0、JDK 21 与 Android SDK 36。
3. 执行 `pnpm install --frozen-lockfile` 和现有网页生产构建。
4. 通过 `cap sync android` 同步 UI 与插件。
5. 动态注入 applicationId、名称、图标、版本和深链。
6. 解码临时 keystore 并执行 Release 构建。
7. 使用 `apksigner verify` 校验签名。
8. 生成版本 APK、`latest.apk` 与 `manifest.json`。
9. 部署到 GitHub Pages。
10. 无论成功失败都清理临时签名材料和构建配置。

## 6. OIDC 与会话

客户端调用 `/api/v1/auth/native/start`，在系统浏览器完成网页 OIDC，再通过一次性授权码调用 `/api/v1/auth/native/exchange`。提供商仍只登记服务端回调：

```text
https://<domain>/api/v1/auth/callback
```

会话令牌通过 Android Keystore 支持的安全存储插件保存，不写入普通 LocalStorage、日志或 URL。完整协议见[OIDC 认证配置](oidc.md)。

## 7. APK 分发

仓库 Pages 来源必须设置为 **GitHub Actions**。成功构建会发布：

```text
https://hanlull.github.io/hl6/android/<applicationId>/latest.apk
https://hanlull.github.io/hl6/android/<applicationId>/<applicationId>-<version>.apk
https://hanlull.github.io/hl6/android/<applicationId>/manifest.json
```

当前地址：

```text
https://hanlull.github.io/hl6/android/cc.lxii.domain/latest.apk
https://hanlull.github.io/hl6/android/cc.lxii.domain/cc.lxii.domain-1.0.0.apk
https://hanlull.github.io/hl6/android/cc.lxii.domain/manifest.json
```

Pages 每次部署只保证最新构建及其版本文件。正式版本会将同一 APK 附加到 GitHub Release，作为永久版本资产；普通 Actions Artifact 只用于 Pages 内部传输，不作为用户 ZIP 下载。

## 8. 版本与更新

客户端启动时调用：

```text
GET /api/v1/client/version?current_version=<versionName>
```

管理后台控制最新版本、强制更新、公告和 HTTPS 更新链接：

- 普通更新：用户可关闭弹窗继续使用。
- 强制更新：弹窗不可关闭，必须存在有效下载链接才能继续。
- 相同版本：正常进入应用。

版本比较使用语义版本，不使用字符串字典顺序。发布新客户端时先确认 APK 可下载，再提高后台最新版本和开启强制更新。

## 9. 本地验证

本地完整构建需要 Node.js 22、pnpm 11.7.0、JDK 21、Android SDK 36 和 release keystore。先设置与工作流同名的环境变量，再执行：

```bash
cd web
pnpm install --frozen-lockfile
pnpm run build
pnpm exec cap sync android
node scripts/configure-capacitor-build.mjs
cd android
./gradlew :app:assembleRelease \
  -PHL6_KEYSTORE_FILE=release.keystore \
  -PHL6_KEYSTORE_PASSWORD="$CLIENT_KEYSTORE_PASSWORD" \
  -PHL6_KEYSTORE_TYPE=PKCS12 \
  -PHL6_KEY_ALIAS="$CLIENT_KEY_ALIAS" \
  -PHL6_KEY_PASSWORD="$CLIENT_KEY_PASSWORD"
```

使用 Android Build Tools：

```bash
apksigner verify --verbose app/build/outputs/apk/release/app-release.apk
```

不要提交生成的 keystore、属性文件、动态图标、`web/dist` 或 APK。

## 10. 发布清单

- 网页生产构建成功且 UI 与网页端一致。
- applicationId、版本、名称、图标和深链正确。
- APK 使用预期 release 证书签名。
- Pages `manifest.json` 的版本、包名和 SHA-256 与 APK 一致。
- 原生登录、深链交换、登出和会话恢复可用。
- 资料、子域名、DNS、积分、通知和管理功能通过同一 API 可用。
- 普通更新与强制更新行为符合后台配置。
- 正式 Release 附加 APK，并使用中英文正式说明。
- `docs/agent.md` 已记录本次客户端相关变化。

## 11. 故障排查

| 现象 | 处理 |
| --- | --- |
| 工作流缺少配置 | 检查六项输入和四项签名 Secret |
| keystore 无效 | 重新编码 Base64，核对 store/key 密码和唯一私钥条目 |
| 图标步骤失败 | 确认 HTTPS、PNG/WebP、大小和 Content-Type，不使用重定向 URL |
| `Failed to fetch` | 检查公网 API、Capacitor CORS 和自定义请求头 |
| `invalid client key` | 使用当前后台密钥重新构建，检查旧密钥是否已作废 |
| OIDC 不返回 App | 核对包名、深链、安装版本和系统浏览器 |
| 更新安装失败 | 核对旧版与新版包名和签名证书 |
| Pages 404 | 检查 Pages Source、deploy 任务和仓库 Pages URL |
| Release APK 摘要不符 | 停止发布，重新核对 manifest 与版本化 APK，禁止强行上传 |
