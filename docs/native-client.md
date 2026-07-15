# Android 网页客户端构建与运维

## 架构

Android 客户端使用 Capacitor 打包 `web/` 的 React/Vite 生产构建。APK 内包含本地 `web/dist`，因此页面、样式、路由、国际化和 API 调用与网页端来自完全相同的源码。Android 平台层只提供应用容器、深链、系统浏览器和安全会话存储。

## 签名 Secrets

在 GitHub 仓库 **Settings > Secrets and variables > Actions** 配置：

- `ANDROID_KEYSTORE_BASE64`：release keystore 的单行 Base64。
- `ANDROID_KEYSTORE_PASSWORD`：keystore 密码。
- `ANDROID_KEY_ALIAS`：签名别名。
- `ANDROID_KEY_PASSWORD`：私钥密码。PKCS12 通常与 keystore 密码相同。

只生成并长期保存一份 release keystore。更换证书会导致 Android 无法覆盖安装旧版本。Windows PowerShell 可用以下命令将 keystore 转成 Secret 值：

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes(".\hl6-release.keystore"))
```

不要提交 keystore、Base64、密码、通讯密钥或构建后的 APK。

## GitHub 构建

在 GitHub Actions 手动运行 `Build Web Android Client`，填写：

| 参数 | 说明 |
| --- | --- |
| `communication_domain` | HL6 HTTPS 域名，不含 `/api/v1`，例如 `domain.example.com`。 |
| `communication_key` | 后台生成的全局通讯密钥。 |
| `client_name` | Android 应用展示名称。 |
| `client_icon` | 可选 HTTPS PNG/WebP URL 或仓库相对 PNG/WebP 路径；远程图标禁止凭据和重定向，最大 2 MiB。 |
| `version` | `major.minor.patch`，例如 `1.0.0`。 |
| `android_package_name` | 小写包名，例如 `cc.example.domain`；每段必须以字母开头，且仅使用小写字母和数字，以兼容 OIDC 深链。 |

工作流依次构建网页、同步 Capacitor Android 项目、生成图标与动态应用配置、签名、校验 APK。完成后会创建 GitHub Release，并将 `.apk` 作为直接下载资产发布；不会使用 ZIP Artifact。

通讯密钥会被打进 APK，所以不是用户权限凭据。工作流会掩码日志中的值，但手动工作流输入不等同于 GitHub Secret，只有受信任的仓库管理员应触发构建。

## OIDC 配置

OIDC 提供商始终只需要配置服务端回调：

```text
https://your-hl6-domain.example/api/v1/auth/callback
```

不要向提供商配置 Android 深链。客户端调用 `POST /api/v1/auth/native/start` 创建短期登录请求，在系统浏览器中完成 OIDC；服务端随后跳转到：

```text
hl6.<android-package-name>://auth/callback?code=<one-time-code>
```

Capacitor 接收该深链，并使用 `POST /api/v1/auth/native/exchange` 与 `X-HL6-Client-Key` 换取 Bearer 会话。会话令牌存储于 Android Keystore 支持的安全存储中。密钥轮换或作废时，受保护请求会失效，用户必须安装使用新密钥构建的 APK 并重新登录。

## 本地验证

需要 Node.js 22、JDK 21、Android SDK 36 和同一份 release keystore。设置六个 `CLIENT_*` 环境变量后执行：

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

随后使用 Android SDK 的 `apksigner verify --verbose app/build/outputs/apk/release/app-release.apk` 校验签名。

## 排错

| 现象 | 处理 |
| --- | --- |
| OIDC 后未回到 App | 确认构建包名、深链和已安装 APK 的包名一致。 |
| `invalid native auth code` | 授权代码已过期或已被使用；重新发起登录。 |
| `invalid client key` | 后台已轮换或作废密钥；重新构建并安装 APK。 |
| 无法覆盖安装更新 | 所有构建必须使用同一份 Android signing Secrets。 |
| 下载后得到 ZIP | 应从 workflow 创建的 GitHub Release 资产下载 `.apk`，不要使用 Actions Artifact 下载入口。 |
