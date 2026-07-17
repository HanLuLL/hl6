# Android 客户端

## 交付模型

Android 使用 Capacitor 打包仓库本地构建的 `web/` React 应用。它不会加载远程站点作为应用 UI，也不包含业务规则的第二个副本。页面渲染、路由、样式、图标、翻译和交互组件与 Web 构建共享。

服务器仍然是认证、授权、资料更新、域名、DNS、积分、封禁、申诉、验证、数据维护和更新策略的权威来源。

## 原生认证

Android 登录、注册、激活和密码重置页面调用与 Web 应用相同的邮箱认证 API。成功的 Android 登录仅在请求包含有效 `X-HL6-Client-Key` 时接收原生 Bearer 会话。会话使用 Android Keystore 支持的安全存储插件持久化。

每个受保护的原生请求同时发送：

```text
Authorization: Bearer <session>
X-HL6-Client-Key: <build communication key>
```

客户端密钥在 Web 管理控制台中生成和撤销。它被有意视为构建标识符，而非用户凭证。

打包的构建在首次启动时提供标识符，然后客户端使用 Android Keystore 支持的存储持久化活动值，并在更新的构建携带不同值时替换它。由于分发的 APK 总是可以被检查，此标识符不是不可提取的密钥，决不能用于用户授权、权限或所有权决策。

## UI 兼容性

对 Web 路由、UI、API 负载、字段、错误、会话行为、资料行为、封禁通知、更新流程或翻译的任何更改必须在同一变更中适配 Android。参见 [agent.md](agent.md)。

## 构建工作流

使用 `.github/workflows/client-build.yml` 及以下输入：

| 输入 | 要求 |
| --- | --- |
| `communication_domain` | HTTPS 后端域名，无路径、查询、片段或凭证 |
| `communication_key` | 在 HL6 管理后台生成的 32 字节 URL 安全密钥 |
| `client_name` | Android 显示名称 |
| `client_icon` | 可选 HTTPS PNG/WebP URL 或仓库相对 PNG/WebP 路径 |
| `version` | `major.minor.patch` 版本名 |
| `android_package_name` | 小写 Android 应用 ID |

在分发签名构建之前配置这些 GitHub Actions 密钥：

```text
ANDROID_KEYSTORE_BASE64
ANDROID_KEYSTORE_PASSWORD
ANDROID_KEY_ALIAS
ANDROID_KEY_PASSWORD
```

工作流验证输入、掩码敏感值、构建本地 UI、同步到 Android、动态注入名称/图标/包名/版本、签名发布 APK、验证其签名，并发布带有 SHA-256 元数据的直接 `.apk` 制品。它永不将 APK 存档在 ZIP 中。

对于官方 LinYu 发布构建，使用检入的 `web/resources/linyu-client-icon.webp` 路径。仓库相对图标路径避免发布时对第三方网络可达性的依赖，同时保留工作流的外部 HTTPS 图标选项用于其他品牌构建。

## 正式发布

仅在 Android 工作流发布匹配的 APK 后使用 `.github/workflows/release.yml`。正式发布工作流仅接受仓库控制的 GitHub Pages 路径：

```text
https://<owner>.github.io/<repository>/android/<application-id>/latest.apk
https://<owner>.github.io/<repository>/android/<application-id>/manifest.json
```

它拒绝重定向、任意 HTTPS 主机、查询字符串、不匹配的包名/版本元数据、不匹配的校验和、意外的签名证书，以及不标识发布提交的清单。发布的 GitHub Release 直接附加原始 APK 及其 `.sha256` 文件。

## 更新

启动时客户端请求：

```text
GET /api/v1/client/version?current_version=<version>
```

Web 管理控制台决定最新版本、强制更新标志、公告和 HTTPS 更新链接。普通更新可以关闭。强制更新阻止进入直到用户按照提供的链接操作。

当旧版构建呈现已撤销或无效的通讯密钥时，`/client/version` 仅公开强制更新恢复响应。正常 API 调用仍然被拒绝。管理员必须在轮换或撤销活动密钥之前配置有效的新版本和 HTTPS 更新 URL。

## CORS

服务器接受 Capacitor 的 `https://localhost` 来源用于打包的 Android 客户端。通过反向代理保留这些请求头：

```text
Origin, Content-Type, Accept, Authorization, X-HL6-Client-Key, X-Idempotency-Key
```