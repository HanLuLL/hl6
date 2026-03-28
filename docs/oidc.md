# OIDC 提供商配置指南

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
3. 向导完成后显示 Client ID 和 Client Secret，**请立即保存**

### 应用内配置

- 在应用配置中添加 Redirect URI → `https://your-hl6-domain.com/api/v1/auth/callback`

### 备注

- 支持 `end_session_endpoint`
- **重要**：当同时颁发 access_token 时，id_token 默认 **不包含** `profile`、`email` 等 scope 的 claims。需要在应用设置中开启 "User Info inside ID Token"，或者通过 `userinfo_endpoint` 获取

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

## 快速参考表

| 提供商 | `OIDC_ISSUER` 格式 | 支持登出 | 注意事项 |
|--------|-------------------|---------|---------|
| **Logto** | `https://{host}/oidc` | Yes | 必须包含 `/oidc` 后缀 |
| **Casdoor** | `https://{host}` | Yes | 支持按应用隔离的 Discovery |
| **Keycloak** | `https://{host}/realms/{realm}` | Yes | v17+ 移除了 `/auth/` 前缀 |
| **Authentik** | `https://{host}/application/o/{slug}/` | Yes | 按应用隔离，含尾部斜杠 |
| **Zitadel** | `https://{instance}.zitadel.cloud` | Yes | id_token 默认不含用户信息 |
| **Google** | `https://accounts.google.com` | **No** | 测试应用 7 天过期 |
| **Entra ID** | `https://login.microsoftonline.com/{tenant}/v2.0` | Yes | 务必使用 v2.0；Secret 会过期 |
| **GitLab** | `https://gitlab.com` | **No** | 需用户设置公开邮箱 |
| **Auth0** | `https://{tenant}.{region}.auth0.com/` | Yes | 含尾部斜杠；旧租户需手动开启 |

> 不支持 `end_session_endpoint` 的提供商，HL6 会在登出时直接返回前端首页 URL。
