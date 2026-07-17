# API 指南

基础路径：`/api/v1`。

响应使用此信封：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

错误使用非零 `code`、HTTP 状态、人类可读的 `message` 和可选的稳定 `message_key`。

## 认证

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| POST | `/auth/registration/request` | 请求新注册的验证邮件 |
| POST | `/auth/activation/request` | 请求现有迁移用户的激活邮件 |
| POST | `/auth/password/forgot` | 请求密码重置邮件，不枚举账户 |
| POST | `/auth/password/complete` | 使用一次性令牌和密码完成注册、激活或重置 |
| POST | `/auth/login` | 邮箱/密码登录 |
| GET | `/auth/me` | 当前用户和积分余额 |
| POST | `/auth/logout` | 清除当前会话 |
| PUT | `/auth/profile` | 更新姓名、头像 URL、简介或网站 |

### 请求示例

```json
POST /auth/login
{
  "email": "user@example.com",
  "password": "a password of at least twelve characters"
}
```

```json
POST /auth/password/complete
{
  "token": "one-time-url-safe-token",
  "password": "a password of at least twelve characters"
}
```

浏览器会话使用 Cookie。Android 登录添加 `X-HL6-Client-Key` 并在响应数据中接收 `access_token`。受保护的原生请求添加 `Authorization: Bearer <access_token>` 和相同的客户端密钥头。登出会递增凭证会话版本并使已发布的浏览器和 Android JWT 失效。

## 封禁状态

当已登录用户被封禁时，受保护请求返回带有 `reason`、`banned_at` 和 `banned_until` 的 `403`。应用允许已登录用户访问 `/auth/me`、封禁信息、申诉和登出，以便封禁界面可以显示完整的时间信息并提交申诉。

## Android 版本

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/client/version?current_version=2.0.0` | 最新版本、更新标志、强制更新标志、公告和 URL |

有效客户端发送 `X-HL6-Client-Key`。缺失或无效的密钥仅在管理员配置了替换元数据时收到强制更新恢复负载；永不允许正常 API 访问。

## 管理

所有管理路由需要已认证的管理员。

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET/PUT | `/admin/settings/access` | 注册和精确域名策略 |
| GET | `/admin/security-events` | 认证安全事件列表 |
| GET/PUT | `/admin/config` | 现有站点、邮件、SEO 和集成配置界面 |
| POST | `/admin/config/url-confirm` | 确认当前公共 URL 配置 |
| GET/PUT | `/admin/client/config` | Android 版本和更新策略 |
| POST | `/admin/client/communication-key` | 生成通讯密钥，仅返回一次 |
| DELETE | `/admin/client/communication-key` | 撤销活动通讯密钥 |
| POST | `/admin/maintenance/export` | 生成并下载服务器控制的数据库 ZIP |
| POST | `/admin/maintenance/restore/challenge` | 密码验证后发行短期恢复挑战 |
| POST | `/admin/maintenance/restore` | 在新密码/挑战/确认后恢复已验证的存档 |
| GET | `/admin/maintenance/restores` | 列出数据库恢复作业和维护状态 |

### 访问设置

```json
{
  "registration_enabled": true,
  "domain_policy_mode": "allowlist",
  "domain_policy_domains": ["example.com", "example.org"]
}
```

允许的模式为 `unrestricted`、`allowlist` 和 `blocklist`。域名为小写和 IDNA 规范化后的精确匹配。

### 数据库恢复

恢复请求是多部分表单数据，包含以下字段：

```text
archive=<validated ZIP>
password=<current administrator password>
challenge=<one-time server challenge>
confirmation=RESTORE DATABASE