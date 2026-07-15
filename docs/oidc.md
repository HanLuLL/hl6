# OIDC 提供商配置指南

- **GitHub 仓库**：[https://github.com/HanLuLL/hl6](https://github.com/HanLuLL/hl6)
- **原项目仓库**：[https://git.houlang.cloud/houlangcloud/hl6](https://git.houlang.cloud/houlangcloud/hl6)

HL6 兼容所有标准 OIDC 提供商。登录前会通过 `/.well-known/openid-configuration` 自动发现 endpoint。

推荐通过环境变量配置：

```env
OIDC_ISSUER=https://your-oidc-provider.example.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
```

也支持在管理员设置页写入数据库配置（适合不方便改 env 的场景）。运行时优先级为字段级：

1. 环境变量（`OIDC_ISSUER` / `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET`）
2. 数据库配置（管理员设置页）

当环境变量与数据库都缺失时：

- 如果系统里还没有任何用户，登录页会弹出 OIDC 首配向导；
- 如果系统里已有用户，匿名首配入口会关闭，需管理员通过环境变量或后台设置恢复。

**回调地址（所有提供商通用）：**

```
https://your-hl6-domain.com/api/v1/auth/callback
```

在提供商的应用设置中将上述地址添加为 Redirect URI / Callback URL。

**请求的 Scope：** `openid email profile`

---

## 目录

- [Logto](#logto)
- [Casdoor](#casdoor)
- [Keycloak](#keycloak)
- [Authentik](#authentik)
- [Zitadel](#zitadel)
- [Authing](#authing)
- [Google](#google)
- [Microsoft Entra ID (Azure AD)](#microsoft-entra-id-azure-ad)
- [GitLab](#gitlab)
- [Auth0](#auth0)
- [快速参考表](#快速参考表)

---

## Logto

> 开源身份基础设施，支持自部署和云托管。

### 配置

```env
# 云托管
OIDC_ISSUER=https://<tenant-id>.logto.app/oidc

# 自部署
OIDC_ISSUER=https://<your-logto-domain>/oidc
```

**注意 Issuer 末尾的 `/oidc`，这是 Logto 特有的路径，不能省略。**

### 获取 Client ID / Client Secret

1. 打开 Logto Console → **Applications**
2. 创建类型为 **Traditional Web** 的应用
3. Application ID 即 `OIDC_CLIENT_ID`，Application Secret 即 `OIDC_CLIENT_SECRET`

### 应用内配置

- **Redirect URIs** → 添加 `https://your-hl6-domain.com/api/v1/auth/callback`
- **Post Sign-out Redirect URIs** → 添加 `https://your-hl6-domain.com`（用于登出后跳转）

### 备注

- 支持 `end_session_endpoint`（RP-Initiated Logout）
- 如果从旧版 HL6 迁移，原来的 `LOGTO_ENDPOINT` 值末尾加 `/oidc` 即为新的 `OIDC_ISSUER`

---

## Casdoor

> 开源 IAM / SSO 平台，支持多种认证协议。

### 配置

```env
OIDC_ISSUER=https://<your-casdoor-host>
```

### 获取 Client ID / Client Secret

1. 进入 Casdoor 管理后台 → **Applications**
2. 创建或选择应用，页面上直接显示 Client ID 和 Client Secret

### 应用内配置

- **Redirect URLs** → 添加 `https://your-hl6-domain.com/api/v1/auth/callback`

### 备注

- 支持 `end_session_endpoint`
- Casdoor 也支持按应用隔离的 Discovery endpoint：`https://<host>/.well-known/<app-name>/openid-configuration`，此时 Issuer 可能需要相应调整

---

## Keycloak

> 开源企业级身份和访问管理。

### 配置

```env
# Keycloak 17+ (Quarkus)
OIDC_ISSUER=https://<keycloak-host>/realms/<realm-name>

# Keycloak 16 及更早版本 (WildFly)
OIDC_ISSUER=https://<keycloak-host>/auth/realms/<realm-name>
```

**v17 起移除了 URL 中的 `/auth/` 前缀，请根据你的版本选择正确格式。**

### 获取 Client ID / Client Secret

1. 进入 Keycloak Admin Console → **Clients** → **Create client**
2. Client type 选择 **OpenID Connect**，输入 Client ID
3. 开启 **Client authentication**（使其成为 confidential client）
4. 创建后在 **Credentials** 标签页找到 Client Secret

### 应用内配置

- **Valid Redirect URIs** → 添加 `https://your-hl6-domain.com/api/v1/auth/callback`

### 备注

- 支持 `end_session_endpoint`
- Issuer URL **不要包含尾部斜杠**，否则会导致验证失败
- 如果 Keycloak 在反向代理后面，需确保 `hostname` 配置与外部访问 URL 一致

---

## Authentik

> 开源身份提供商，注重灵活性和可扩展性。

### 配置

```env
OIDC_ISSUER=https://<authentik-host>/application/o/<app-slug>/
```

**注意 Issuer 包含尾部斜杠，且是按应用（slug）隔离的。**

### 获取 Client ID / Client Secret

1. 进入 Authentik 管理界面 → **Providers** → 创建 **OAuth2/OpenID Connect** 类型
2. 设置授权流程和 Redirect URI，创建后页面显示 Client ID 和 Client Secret
3. 然后在 **Applications** 中创建应用，设置 slug 并关联到上一步的 Provider

### 应用内配置

- 创建 Provider 时设置 Redirect URI → `https://your-hl6-domain.com/api/v1/auth/callback`

### 备注

- 支持 `end_session_endpoint`
- 每个应用有独立的 Discovery endpoint，这与大多数提供商不同
- Docker/Kubernetes 部署时需确保 `AUTHENTIK_EXTERNAL_HOST` 设置正确，否则 Discovery 文档中会包含内部主机名

---

## Zitadel

> 开源云原生身份管理平台。

### 配置

```env
# 云托管
OIDC_ISSUER=https://<instance-name>.zitadel.cloud

# 自部署
OIDC_ISSUER=https://<your-custom-domain>
```

### 获取 Client ID / Client Secret

1. 进入 Zitadel Console → 创建 **Project** → 添加 **Application**
2. 类型选 Web，认证方式选 Basic
3. 向导完成后显示 Client ID and Client Secret，**请立即保存**

### 应用内配置

- 在应用配置中添加 Redirect URI → `https://your-hl6-domain.com/api/v1/auth/callback`

### 备注

- 支持 `end_session_endpoint`
- **重要**：当同时颁发 access_token 时，id_token 默认 **不包含** `profile`、`email` 等 scope 的 claims。需要在应用设置中开启 "User Info inside ID Token"，或者通过 `userinfo_endpoint` 获取

---

## Authing

> 国内领先的云原生 IDaaS 产品，兼容 OAuth 2.0, OIDC, SAML, AD/LDAP, WS-Fed 等主流认证协议。

### 配置

```env
# 云托管
OIDC_ISSUER=https://<your-app-domain>.authing.cn/oidc
```

**注意 Issuer 末尾的 `/oidc`，这是 Authing OIDC 应用特有的路径，不能省略。**

### 获取 Client ID / Client Secret

1. 登录 [Authing 控制台](https://console.authing.cn/)
2. 进入 **应用** → **应用列表** → **添加应用**
3. 填写应用信息，**认证地址** 填写应用域名，**回调链接** 填写 `https://your-hl6-domain.com/api/v1/auth/callback`
4. 在应用详情页面的 **应用 ID** 即 `OIDC_CLIENT_ID`，**应用密钥** 即 `OIDC_CLIENT_SECRET`

### 应用内配置

- **登录回调地址** → 添加 `https://your-hl6-domain.com/api/v1/auth/callback`
- **登出回调地址** → 添加 `https://your-hl6-domain.com`（用于登出后跳转）
- 在「安全性」卡片中，**id_token 签名算法** 选择 **RS256**
- **换取 token 身份验证方式**、**检验 token 身份验证方式**、**撤回 token 身份验证方式** 选择 **none**

### 备注

- 支持 `end_session_endpoint`（RP-Initiated Logout）
- 确保在 Authing 应用配置中启用了 `openid`、`email`、`profile` 等 scope
- 用户信息字段映射：用户名使用 `name`、`username`、`nickname`、`given_name` 或 `family_name`，邮箱使用 `email` 或 `email_address`，手机号使用 `phone`，头像使用 `picture`、`avatar` 或 `avatar_url`
- HL6 针对 Authing 进行了特别兼容性优化，支持多种字段映射、无kid（Key ID）的JWT令牌验证和灵活的 JWT 验证

---

## Google

> 商业 OIDC 提供商，适用于允许 Google 账号登录的场景。

### 配置

```env
OIDC_ISSUER=https://accounts.google.com
```

### 获取 Client ID / Client Secret

1. 打开 [Google Cloud Console](https://console.cloud.google.com) → **APIs & Services** → **Credentials**
2. 先配置 **OAuth consent screen**（必须步骤）
3. **Create Credentials** → **OAuth client ID** → 类型选 **Web application**
4. 创建后显示 Client ID 和 Client Secret

### 应用内配置

- **Authorized redirect URIs** → 添加 `https://your-hl6-domain.com/api/v1/auth/callback`
- Redirect URI **必须使用 HTTPS**（localhost 开发环境除外）

### 备注

- **不支持** `end_session_endpoint`（HL6 会自动处理：登出时直接返回前端 URL）
- 应用处于 "Testing" 状态时，consent 和 token **7 天后过期**。正式使用前需发布应用
- 凭据更改可能需要 5 分钟到数小时生效

---

## Microsoft Entra ID (Azure AD)

> 微软企业级身份平台。

### 配置

```env
# 单租户（推荐）
OIDC_ISSUER=https://login.microsoftonline.com/<tenant-id>/v2.0

# 多租户（同时支持工作/学校账户和个人微软账户）
OIDC_ISSUER=https://login.microsoftonline.com/common/v2.0
```

`<tenant-id>` 可以是：
- 租户 GUID（如 `12345678-1234-...`）
- 租户域名（如 `contoso.onmicrosoft.com`）

**务必使用 `v2.0` endpoint，v1 的 token 格式和 issuer 声明不同。**

### 获取 Client ID / Client Secret

1. 打开 [Microsoft Entra 管理中心](https://entra.microsoft.com) → **Applications** → **App registrations** → **New registration**
2. Overview 页面的 **Application (client) ID** 即 `OIDC_CLIENT_ID`
3. **Certificates & secrets** → **New client secret** → 复制 **Value**（仅显示一次）

### 应用内配置

- 注册应用时添加 **Web** 平台的 Redirect URI → `https://your-hl6-domain.com/api/v1/auth/callback`

### 备注

- 支持 `end_session_endpoint`
- Client Secret 有**过期时间**（最长 2 年），请设置提醒及时轮换
- 使用 `common` endpoint 时，token 中的 issuer 包含 `{tenantid}` 占位符，需要动态验证——建议使用指定租户的 endpoint

---

## GitLab

> 开源 DevOps 平台，同时可作为 OIDC 提供商。

### 配置

```env
# gitlab.com
OIDC_ISSUER=https://gitlab.com

# 自部署
OIDC_ISSUER=https://<your-gitlab-instance>
```

### 获取 Client ID / Client Secret

1. GitLab → **User Settings** → **Applications**（用户级别），或 **Admin Area** → **Applications**（实例级别）
2. 创建应用时 **务必勾选 `openid`、`email`、`profile` scope**
3. Application ID 即 `OIDC_CLIENT_ID`，Secret 即 `OIDC_CLIENT_SECRET`

### 应用内配置

- **Callback URL** → `https://your-hl6-domain.com/api/v1/auth/callback`

### 备注

- **不支持** `end_session_endpoint`（HL6 会自动处理）
- `email` claim 仅在用户设置了公开邮箱时才会包含在 token 中
- 自部署实例需确保外部可访问，否则 Discovery 会失败

---

## Auth0

> 商业身份即服务平台（Okta 旗下）。

### 配置

```env
# US 区域（2020 年 6 月后创建的租户）
OIDC_ISSUER=https://<tenant>.us.auth0.com/

# US 区域（2020 年 6 月前创建的租户）
OIDC_ISSUER=https://<tenant>.auth0.com/

# EU / AU 区域
OIDC_ISSUER=https://<tenant>.eu.auth0.com/
OIDC_ISSUER=https://<tenant>.au.auth0.com/

# 自定义域名
OIDC_ISSUER=https://auth.yourdomain.com/
```

**注意 Auth0 的 Issuer 包含尾部斜杠。**

### 获取 Client ID / Client Secret

1. 打开 Auth0 Dashboard → **Applications** → **Create Application**
2. 类型选 **Regular Web Applications**
3. **Settings** 标签页显示 Client ID 和 Client Secret

### 应用内配置

- **Allowed Callback URLs** → `https://your-hl6-domain.com/api/v1/auth/callback`
- **Allowed Logout URLs** → `https://your-hl6-domain.com`（用于登出后跳转）

### 备注

- 支持 `end_session_endpoint`，但 **2023 年 11 月 14 日前创建的租户** 需手动开启：Dashboard → Settings → Advanced → Login and Logout → RP-Initiated Logout End Session Endpoint Discovery
- 如果不传 `id_token_hint`，Auth0 会显示确认登出的页面。可在 Advanced 设置中关闭

---

## Android Capacitor 客户端适配

HL6 Android 客户端由 Capacitor 打包同一份 React/Vite 网页构建产物，页面、路由和 API 调用不维护第二套实现。OIDC 登录在系统浏览器中完成，浏览器 Cookie 不会被作为 App 会话使用。

OIDC 提供商仍只需要配置以下服务器 HTTPS 回调地址：

```
https://your-hl6-domain.com/api/v1/auth/callback
```

不要将 Android 深链接登记为 OIDC 提供商回调。服务端在验证 OIDC 后会生成一个 90 秒有效、只能消费一次的授权代码，再跳转到构建时生成的：

```
hl6.<android-package-name>://auth/callback?code=<one-time-code>
```

Capacitor 客户端先带 `X-HL6-Client-Key` 调用 `POST /api/v1/auth/native/start`，提交构建时生成的深链。服务端返回 90 秒有效、只能使用一次的浏览器登录地址；只有这个短期地址会进入系统浏览器，通讯密钥不会进入浏览器 URL、OIDC 提供商或深链。

OIDC 回调完成后，服务端会生成 90 秒有效、只能消费一次的授权代码并跳转到 Android 深链。Capacitor `App` 插件接收深链后，客户端带 `X-HL6-Client-Key` 调用 `POST /api/v1/auth/native/exchange` 交换 Bearer 会话令牌。会话令牌由 Android Keystore 支持的安全存储保存；每个受保护 API 请求都会由服务端重新校验通讯密钥。通讯密钥由 HL6 后台生成、轮换或作废，密钥失效后客户端必须重新构建和安装。

不要直接在浏览器调用 `/api/v1/auth/login?native_redirect_uri=...`；该参数已被服务端拒绝。Android 深链不需要、也不应登记为 OIDC 提供商回调地址。

详见 [Android 网页客户端构建与运维](native-client.md) 和 [客户端适配规范](agent.md)。

## 快速参考表

| 提供商 | `OIDC_ISSUER` 格式 | 支持登出 | 注意事项 |
|--------|-------------------|---------|---------|
| **Logto** | `https://{host}/oidc` | Yes | 必须包含 `/oidc` 后缀 |
| **Casdoor** | `https://{host}` | Yes | 支持按应用隔离的 Discovery |
| **Keycloak** | `https://{host}/realms/{realm}` | Yes | v17+ 移除了 `/auth/` 前缀 |
| **Authentik** | `https://{host}/application/o/{slug}/` | Yes | 按应用隔离，含尾部斜杠 |
| **Zitadel** | `https://{instance}.zitadel.cloud` | Yes | id_token 默认不含用户信息 |
| **Authing** | `https://{app-domain}.authing.cn/oidc` | Yes | 必须包含 `/oidc` 后缀；需配置 RS256 签名算法；支持多种用户信息字段映射；支持无kid的JWT令牌 |
| **Google** | `https://accounts.google.com` | **No** | 测试应用 7 天过期 |
| **Entra ID** | `https://login.microsoftonline.com/{tenant}/v2.0` | Yes | 务必使用 v2.0；Secret 会过期 |
| **GitLab** | `https://gitlab.com` | **No** | 需用户设置公开邮箱 |
| **Auth0** | `https://{tenant}.{region}.auth0.com/` | Yes | 含尾部斜杠；旧租户需手动开启 |

> 不支持 `end_session_endpoint` 的提供商，HL6 会在登出时直接返回前端首页 URL。
