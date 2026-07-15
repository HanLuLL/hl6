# Android 网页客户端构建与运维

## 架构

Android 客户端通过 Capacitor 将既有 `web/` React/Vite 应用的生产构建产物打包进 APK。APK 内包含本次构建的 `web/dist`，页面、路由、组件、样式、国际化和 API 调用均与网页端使用同一份源码。

客户端不会把线上网站作为 UI 加载。Android 平台层只负责应用启动、深链、OIDC 系统浏览器、Android Keystore 支持的会话存储和 API 传输；权限判断、校验、DNS 操作、积分和其他业务决策始终由服务端完成。

## 通讯密钥与会话

- 在 **管理后台 > 客户端管理** 中生成、轮换或作废通讯密钥。
- 服务端只保存密钥的 SHA-256 哈希，生成时仅返回一次明文。开始构建前应保存到密码管理器。
- 密钥会被注入 APK，因此它用于识别客户端构建，而不是用户凭据或服务端私密密钥。
- 每个原生 API 请求均携带 `X-HL6-Client-Key`；已登录请求还携带服务端签发的 Bearer 会话令牌。
- Bearer 令牌通过 Android Keystore 支持的安全存储插件保存，禁止写入普通 LocalStorage、源码、日志或工作流输出。

轮换或作废通讯密钥会使旧客户端构建失效。应先重新构建并分发新 APK，再要求用户重新登录。

## GitHub Actions 构建

在 GitHub Actions 中手动运行 **Build Web Android Client**。工作流会校验并动态注入以下参数：

| 参数 | 说明 |
| --- | --- |
| `communication_domain` | HTTPS HL6 域名，不含 `/api/v1`，例如 `domain.example.com` |
| `communication_key` | 在管理后台生成的通讯密钥 |
| `client_name` | Android 应用显示名称 |
| `client_icon` | 可选 HTTPS PNG/WebP 图标地址或仓库相对 PNG/WebP 路径 |
| `version` | `major.minor.patch`，例如 `1.0.0` |
| `android_package_name` | 小写 Android applicationId，例如 `cc.example.domain` |

在 **Settings > Secrets and variables > Actions** 中配置四项仓库 Secret：

- `ANDROID_KEYSTORE_BASE64`：单行 Base64 编码的 release keystore
- `ANDROID_KEYSTORE_PASSWORD`：keystore 密码
- `ANDROID_KEY_ALIAS`：签名密钥别名
- `ANDROID_KEY_PASSWORD`：签名密钥密码

Windows PowerShell 可使用以下命令编码 keystore：

    [Convert]::ToBase64String([IO.File]::ReadAllBytes(".\hl6-release.keystore"))

同一包名的所有版本必须使用同一份 release keystore，否则 Android 无法在旧版上覆盖安装更新。

工作流依次构建网页 UI、同步 Capacitor、写入动态 Android 配置、签名 Release APK，并执行 `apksigner verify`。默认**不会**创建 GitHub Release、发布 APK 或上传外部交付物。GitHub Actions Artifact 下载固定为 ZIP；若需要直接 APK，必须由仓库所有者明确授权并单独配置发布渠道，例如 Release 或受控 HTTPS 下载服务。

## OIDC

OIDC 提供商只需配置服务端回调：

    https://your-hl6-domain.example/api/v1/auth/callback

不要把 Android 深链登记为 OIDC 提供商回调。客户端携带通讯密钥调用 `POST /api/v1/auth/native/start`，在系统浏览器中打开服务端返回的短期登录地址；登录完成后服务端跳转至：

    hl6.<android-package-name>://auth/callback?code=<one-time-code>

客户端通过 `POST /api/v1/auth/native/exchange` 交换一次性授权码，并安全保存会话。通讯密钥不得出现在浏览器 URL、OIDC 参数或深链中。

## CORS 部署检查

Capacitor Android 请求的来源是 `https://localhost`。当前服务端版本已允许该来源，但必须先部署新镜像，浏览器引擎才能收到正确的 CORS 响应。

部署后在可信机器上执行预检：

    curl -i -X OPTIONS https://your-hl6-domain.example/api/v1/auth/native/start \
      -H 'Origin: https://localhost' \
      -H 'Access-Control-Request-Method: POST' \
      -H 'Access-Control-Request-Headers: content-type,x-hl6-client-key'

响应中必须包含：

    Access-Control-Allow-Origin: https://localhost

若缺失该响应头，说明容器仍在运行旧镜像，或反向代理移除了 CORS 响应头。应更新服务端镜像并检查代理配置后再重新构建 APK。

## 本地验证

本地构建需要 Node.js 22、pnpm 11.7.0、JDK 21、Android SDK 36 以及 release keystore。

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

随后使用 Android SDK 的 `apksigner verify --verbose app/build/outputs/apk/release/app-release.apk` 校验签名。

## 排错

| 现象 | 处理方式 |
| --- | --- |
| 原生登录显示 `Failed to fetch` | 检查 API 域名、网络和上方 CORS 预检。预期来源为 `https://localhost`。 |
| 缺少 `Access-Control-Allow-Origin` | 部署当前服务端镜像；若仍缺失，检查反向代理响应头。 |
| OIDC 后未返回 App | 确认包名、生成的深链和已安装 APK 包名完全一致。 |
| `invalid native auth code` | 授权码已过期或已使用，重新发起登录。 |
| `invalid client key` | 通讯密钥已作废或轮换，使用当前密钥重新构建并安装 APK。 |
| Android 拒绝覆盖安装更新 | 使用同一包名和同一份签名 keystore 构建。 |
