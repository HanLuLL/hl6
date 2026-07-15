# 原生 Android 客户端构建与运维

## 前置条件

在 HL6 管理后台的“客户端版本与通信”卡片中生成通讯密钥，并保存仅显示一次的明文。配置版本号、更新类型、公告和 HTTPS 更新链接。

在 GitHub 仓库 **Settings > Secrets and variables > Actions** 中设置以下仓库级签名信息：

- `ANDROID_KEYSTORE_BASE64`: release keystore 的 Base64 内容。
- `ANDROID_KEYSTORE_PASSWORD`: keystore 密码。
- `ANDROID_KEY_ALIAS`: 签名别名。
- `ANDROID_KEY_PASSWORD`: 签名私钥密码。

这些 Secrets 缺失时，构建会明确失败。`CLIENT_KEYSTORE_PASSWORD` 报错对应缺少 `ANDROID_KEYSTORE_PASSWORD`；工作流会一次列出全部缺失项。不能使用每次自动生成的新签名密钥，因为 Android 无法用不同证书覆盖安装后续更新。

首次构建前，请在受控设备上只生成一次 release keystore，并长期妥善保存。示例：

```bash
keytool -genkeypair -keystore hl6-release.keystore -storetype PKCS12 -alias hl6-release -keyalg RSA -keysize 4096 -validity 9125
base64 -w 0 hl6-release.keystore
```

将第二条命令输出的一行 Base64 配置为 `ANDROID_KEYSTORE_BASE64`，并将 `keytool` 提示输入的 keystore 密码、别名和密钥密码分别配置为其余三个 Secret。PKCS12 通常使用相同的 keystore 密码和密钥密码。Windows PowerShell 可用以下命令生成 Base64：

```powershell
[Convert]::ToBase64String([IO.File]::ReadAllBytes(".\hl6-release.keystore"))
```

不要提交 keystore、Base64 文本或密码。工作流仅在运行目录写入 keystore，构建完成或失败后均会清理。

Windows 证书工具导出的 PKCS12 文件可能使用与填写别名不同的内部别名。对于恰好包含一个私钥条目的 keystore，工作流会通过 `keytool` 自动识别内部别名和 keystore 类型，再注入 Gradle；不会在日志中输出签名值。包含多个私钥条目的 keystore 会被拒绝，避免误用错误的发布证书。

## 手动构建

在 GitHub Actions 中运行 `Build Native Android Client`，填写：

| 参数 | 说明 |
| --- | --- |
| `communication_domain` | HL6 HTTPS 域名，不含 `/api/v1`。 |
| `communication_key` | 后台生成的全局通讯密钥。 |
| `client_name` | Android 应用展示名称。 |
| `client_icon` | 可选的仓库内 PNG 相对路径。 |
| `version` | `major.minor.patch`，例如 `1.0.0`；每个数字段为 0-999。 |
| `android_package_name` | 小写 Android 包名，例如 `cloud.houlang.hl6`；最长 60 个字符，以满足原生 OIDC URI scheme 限制。 |

工作流将验证参数、生成临时构建配置、验证签名，并上传 `app-release.apk` Artifact。`communication_key` 是敏感值：工作流会在日志中掩码，但 GitHub 的手动工作流输入不等同于 GitHub Secret，因此只能由受信任的仓库管理员触发和查看。

工作流文件必须先进入仓库默认分支，GitHub Actions 才会在 Actions 页面注册 `workflow_dispatch` 的手动构建入口。

## OIDC 配置

OIDC 提供商仍只需要配置服务器回调：

```
https://your-hl6-domain.example/api/v1/auth/callback
```

原生客户端先通过带 `X-HL6-Client-Key` 的 `POST /api/v1/auth/native/start` 提交构建时生成的深链。服务端保存该深链对应的一次性、90 秒有效请求并返回浏览器登录地址。App 只把该不含密钥的短期地址交给系统 Custom Tabs。

OIDC 登录完成后，服务端重定向到构建时生成的 `hl6.<applicationId>://auth/callback?code=...`。深链仅携带 90 秒有效且只能消费一次的代码，App 必须再带 `X-HL6-Client-Key` 调用 `POST /api/v1/auth/native/exchange` 换取 Bearer 会话。原生会话的每次受保护 API 调用都会在服务端重新校验通讯密钥；密钥轮换或作废会立即使旧 APK 的会话失效。

使用新密钥覆盖安装新 APK 时，客户端会替换 Android Keystore 中保存的旧密钥并清除与旧密钥哈希绑定的会话令牌，用户需要重新完成 OIDC 登录。

不要把 Android 深链接配置成 OIDC 提供商回调地址，也不要直接调用 `/api/v1/auth/login?native_redirect_uri=...`。该旧参数已被拒绝，避免未受通讯密钥保护的深链登录请求。

## 原生 API 契约

所有 Android API 请求都携带 `X-HL6-Client-Key`。`/auth/native/exchange` 后取得的 Bearer 会话带有当前通讯密钥哈希声明，服务端会在每次受保护请求中同时验证请求密钥和该声明，因此密钥轮换会立即失效旧会话。

| 接口 | 用途 |
| --- | --- |
| `GET /api/v1/client/version?current_version=x.y.z` | 服务端计算 `update_available` 并返回更新策略。 |
| `POST /api/v1/auth/native/start` | 创建一次性浏览器登录请求，输入为 `redirect_uri`。 |
| `POST /api/v1/auth/native/exchange` | 用 Android 深链中的一次性 `code` 换取 Bearer 会话。 |

## 排错

| 现象 | 原因与处理 |
| --- | --- |
| App 未被登录回调唤醒 | 检查构建时包名是否与安装的 APK 一致，重新安装对应包名的 APK。 |
| `invalid native auth code` | 授权代码仅能使用一次且 90 秒过期；重新发起登录。 |
| `invalid native login request` | 浏览器登录地址超过 90 秒、重复使用，或不是由 `/auth/native/start` 创建；回到 App 重新点击登录。 |
| 客户端通讯密钥不可用 | 密钥已被轮换或作废；在后台生成新密钥后重新构建并安装 APK。 |
| 无法覆盖安装更新 | 新 APK 使用了不同签名；确保所有 Actions 构建使用相同 GitHub Secrets 中的 keystore。 |
| 强制更新无法进入 App | 这是服务端强制更新策略；修正更新链接并安装最新 APK。 |
