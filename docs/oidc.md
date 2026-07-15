# OIDC 认证配置

HL6 使用标准 OpenID Connect 登录。网页端、服务端和 Android 客户端共享同一个 OIDC Client；身份提供商只登记服务端网页回调，Android 深链不直接暴露给身份提供商。

## 1. 唯一回调地址

```text
https://<HL6 公网域名>/api/v1/auth/callback
```

示例：

```text
https://domain.example.com/api/v1/auth/callback
```

回调必须精确匹配协议、主机、端口和路径。不要登记前端 `/callback` 页面，也不要登记 `hl6.<applicationId>://...` 深链。

## 2. 创建 OIDC 应用

在身份提供商中创建 Web/Regular Web Application：

1. 启用 Authorization Code Flow。
2. 登记上述回调 URL。
3. 允许 `openid`、`profile`、`email` Scope。
4. 记录 Issuer、Client ID 和 Client Secret。
5. 如果提供商要求 Logout URL，填写 HL6 公网首页或组织批准的退出页。

HL6 通过 `/.well-known/openid-configuration` 获取授权、令牌和 JWKS 端点。Issuer 必须是 Discovery 文档声明的精确值，而不是管理控制台 URL。

## 3. 常见 Issuer 形式

| 提供商 | Issuer 示例 | 注意事项 |
| --- | --- | --- |
| Logto | `https://tenant.logto.app/oidc` | 自部署实例使用自己的 Logto Endpoint |
| Keycloak 17+ | `https://id.example.com/realms/<realm>` | 不使用旧版 `/auth` 前缀，除非实际部署保留 |
| Authentik | `https://id.example.com/application/o/<slug>/` | Provider Slug 必须与应用一致 |
| Zitadel | `https://tenant.zitadel.cloud` | 使用 Web Application 和 Code Flow |
| Google | `https://accounts.google.com` | 回调域名必须在 OAuth 同意屏幕配置范围内 |
| Microsoft Entra ID | `https://login.microsoftonline.com/<tenant>/v2.0` | `<tenant>` 可为租户 ID 或组织策略允许值 |
| Auth0 | `https://tenant.auth0.com/` | 自定义域名时使用 Discovery 返回的 Issuer |
| GitLab | `https://gitlab.com` | 自部署实例改为实际 GitLab 根地址 |
| Casdoor | Casdoor 实例公开 Issuer | 以 Discovery 文档为准 |
| Authing | 应用 OIDC Issuer | 不使用控制台地址代替 Issuer |

配置前可验证 Discovery：

```bash
curl --fail https://issuer.example.com/.well-known/openid-configuration
```

响应至少应包含 `issuer`、`authorization_endpoint`、`token_endpoint` 和 `jwks_uri`。

## 4. HL6 配置方式

### 环境变量

```dotenv
OIDC_ISSUER=https://issuer.example.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
```

适合由基础设施锁定配置的生产环境。环境变量按字段覆盖数据库，后台修改同一字段不会生效。

### 首配向导

当三项 OIDC 环境变量都为空且数据库没有用户时，访问 HL6 会进入首配流程。服务端校验 Issuer 和 Discovery 后保存配置，首个成功注册用户成为管理员。

首配完成后不可通过清空用户界面绕过管理员权限。后续配置修改需要管理员会话，并受 URL 确认和加密存储策略保护。

## 5. 网页登录流程

1. 前端读取 `/api/v1/auth/oidc/status`。
2. 用户访问 `/api/v1/auth/login`。
3. 服务端生成带状态校验的授权请求并跳转到提供商。
4. 提供商回调 `/api/v1/auth/callback`。
5. 服务端校验状态、授权码、令牌签名和声明，创建或更新用户身份关联。
6. 服务端签发会话，跳回前端回调页。
7. 前端通过 `/api/v1/auth/me` 获取当前用户。

用户自定义姓名和头像存储在 HL6 数据库。OIDC 未返回姓名时不得清空已保存姓名；普通重新登录也不得用默认头像覆盖用户自定义头像。

## 6. Android 原生登录

Android 使用系统浏览器，不在 WebView 中承载身份提供商页面：

1. 客户端携带 `X-HL6-Client-Key` 调用 `POST /api/v1/auth/native/start`。
2. 服务端返回短期网页登录地址。
3. 客户端使用 Capacitor Browser 打开系统浏览器。
4. 网页 OIDC 回调完成后，服务端生成短期、一次性原生授权码。
5. 服务端跳转到：

```text
hl6.<applicationId>://auth/callback?code=<one-time-code>
```

6. App 插件接收深链，客户端调用 `POST /api/v1/auth/native/exchange`。
7. 服务端返回会话令牌，客户端通过 Android Keystore 支持的安全存储保存。

通讯密钥绝不能放入 OIDC URL、查询参数、State 或深链。原生授权码短期有效且只能使用一次。

当前包名 `cc.lxii.domain` 对应：

```text
hl6.cc.lxii.domain://auth/callback
```

这个地址由 Android Manifest/Capacitor 配置处理，不登记到 OIDC 提供商。

## 7. Capacitor CORS

Android WebView 中本地 UI 的来源是 `https://localhost`。原生登录 POST 会携带：

- `Content-Type`
- `X-HL6-Client-Key`
- `X-Idempotency-Key`

部署后验证预检：

```bash
curl -i -X OPTIONS https://domain.example.com/api/v1/auth/native/start \
  -H 'Origin: https://localhost' \
  -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: content-type,x-hl6-client-key,x-idempotency-key'
```

必须包含：

```text
Access-Control-Allow-Origin: https://localhost
Access-Control-Allow-Headers: Origin, Content-Type, Accept, Authorization, X-HL6-Client-Key, X-Idempotency-Key
Vary: Origin
```

CORS 允许只解决浏览器传输限制，不会绕过通讯密钥、会话或权限检查。

## 8. 安全检查

- 生产回调和 Issuer 全部使用 HTTPS。
- Client Secret、会话密钥和加密密钥不进入前端构建、APK、日志或 Release。
- 服务器时间通过 NTP 同步，避免令牌时间校验失败。
- 限制 OIDC 应用允许的回调 URL，不使用宽泛通配符。
- 身份提供商启用 MFA、登录审计和管理员保护。
- 删除用户或禁用身份时，同时评估 HL6 会话和本地账号状态。

## 9. 故障排查

| 错误 | 原因 | 处理 |
| --- | --- | --- |
| `redirect_uri_mismatch` | 提供商回调不精确 | 登记唯一服务端回调 |
| Discovery 失败 | Issuer 错误、证书或网络 | 直接请求 `/.well-known/openid-configuration` |
| `invalid state` | Cookie/会话丢失或回调重复 | 检查 HTTPS、域名、代理和浏览器 Cookie |
| Token 校验失败 | JWKS、Issuer、Audience 或时钟不符 | 核对 Discovery、Client ID 和 NTP |
| 后台配置不生效 | 环境变量仍覆盖数据库 | 移除对应环境变量并重启，或继续由环境变量维护 |
| Android `Failed to fetch` | CORS、网络或通讯密钥 | 执行预检并检查当前服务端镜像 |
| `invalid native auth code` | 授权码过期或重复使用 | 重新发起登录，不重放旧深链 |
| Android 未回到 App | 包名/深链不一致 | 使用相同包名重新构建并检查 Manifest |
