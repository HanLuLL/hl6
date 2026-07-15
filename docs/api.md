# API 集成指南

HL6 网页端和 Android 客户端都调用同一组 REST API。本文描述稳定的协议约定与路由族；具体字段以仓库中的 Handler、Model 和前端 TypeScript 类型为准。

## 1. 基础地址

```text
https://<domain>/api/v1
```

健康检查不在 API 前缀下：

```text
GET /health
```

`robots.txt` 和 `sitemap.xml` 也位于站点根路径。

## 2. 响应格式

成功：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

创建成功使用 HTTP 201，`message` 为 `created`。错误：

```json
{
  "code": -1,
  "message": "human-readable fallback",
  "message_key": "error.invalidRequestBody",
  "data": {}
}
```

前端优先使用 `message_key` 国际化；第三方客户端应保留 `message` 作为回退，不要根据英文文本判断业务类型。

页码分页：

```json
{
  "code": 0,
  "message": "ok",
  "data": [],
  "total": 120,
  "page": 1,
  "per_page": 20
}
```

部分大型列表使用 `offset`、`limit`。

## 3. 认证头

登录后请求：

```http
Authorization: Bearer <session-token>
```

Android 原生请求额外携带：

```http
X-HL6-Client-Key: <communication-key>
```

会话决定用户身份，通讯密钥只决定构建是否被允许连接。管理员接口还会检查用户角色或管理员用户组。

## 4. 幂等

POST、PUT、DELETE 等变更请求应携带：

```http
X-Idempotency-Key: <unique-request-id>
```

DNS、域名、批量任务等关键写操作将 Scope 与 Key 结合保存结果。网络超时后重试同一业务请求应复用原 Key；用户明确发起新操作时才生成新 Key。

## 5. 公开接口

### 品牌、SEO 与版本

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/branding` | 站点名称、公告、页脚等公开品牌信息 |
| GET | `/branding/logo.webp` | 当前 Logo |
| GET | `/branding/favicon.ico` | 当前 Favicon |
| GET | `/seo/meta` | SEO 描述、关键词和公开元数据 |
| GET | `/client/version?current_version=1.0.0` | Android 更新策略 |

### OIDC

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/auth/oidc/status` | 是否已配置 OIDC、是否允许首配 |
| POST | `/auth/oidc/bootstrap` | 无用户时首配 OIDC |
| GET | `/auth/login` | 发起网页登录 |
| GET | `/auth/callback` | OIDC 提供商回调 |
| POST | `/auth/native/start` | Android 发起原生登录，需要通讯密钥 |
| POST | `/auth/native/exchange` | 交换一次性原生授权码，需要通讯密钥 |

### 域名与公共页面

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/public/domains` | 可公开认领的域名 |
| GET | `/public/subdomains/check` | 子域名可用性检查 |
| GET | `/friend-links` | 启用的友情链接 |
| GET | `/payment/methods` | 当前可用支付渠道 |

支付网关还会调用 `/payment/epay/notify`、`/payment/codepay/notify`，用户同步返回使用 `/payment/return`。这些端点公开不代表无需签名；Handler 必须验证网关签名和订单。

## 6. 登录用户接口

### 会话与资料

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/auth/me` | 当前用户、用户组和权限信息 |
| PUT | `/auth/profile` | 保存自定义姓名和头像 |
| POST | `/auth/logout` | 注销会话 |

资料更新写入 HL6 数据库。客户端不应把 OIDC 缺失的姓名解释为清空命令。

### 域名、子域名与记录

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/domains` | 当前用户可用域名 |
| GET | `/subdomains` | 用户已认领子域名 |
| GET | `/subdomains/settings` | 认领规则与额度 |
| POST | `/subdomains` | 认领子域名 |
| GET | `/subdomains/:id` | 子域名详情 |
| DELETE | `/subdomains/:id` | 释放认领 |
| GET | `/subdomains/:id/records` | DNS 记录列表 |
| POST | `/subdomains/:id/records` | 创建记录 |
| PUT | `/subdomains/:id/records/:recordId` | 更新记录 |
| DELETE | `/subdomains/:id/records/:recordId` | 删除记录 |

服务端校验资源归属、用户组额度、记录格式、重复和 CNAME 共存规则。

### 积分、推荐与支付

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/credits` | 当前余额 |
| GET | `/credits/transactions` | 交易流水 |
| GET | `/credits/checkin/status` | 今日签到状态 |
| POST | `/credits/checkin` | 每日签到 |
| GET | `/referrals` | 推荐信息与奖励 |
| GET | `/payment/products` | 可购买积分产品 |
| POST | `/payment/orders` | 创建支付订单 |
| GET | `/payment/orders` | 用户订单列表 |

### 通知与申诉

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/notifications` | 通知列表 |
| GET | `/notifications/unread` | 未读状态 |
| GET | `/notifications/sse` | 实时 SSE |
| GET | `/notifications/images/:id` | 通知图片 |
| GET | `/notifications/:id` | 通知详情 |
| POST | `/notifications/:id/read` | 标记已读 |
| GET | `/ban-info` | 当前封禁原因与时间 |
| POST | `/appeals` | 提交申诉 |
| GET | `/appeals` | 我的申诉 |
| GET | `/ai-audit/stats` | 用户可见的 AI 审查统计 |

SSE 只是更新信号，重连后仍应通过普通列表接口恢复权威状态。

## 7. 管理员接口

所有以下路径需要登录并通过管理员中间件。

### 域名、DNS 账号与迁移

```text
POST   /admin/domains
GET    /admin/domains/reserved-prefixes
PUT    /admin/domains/reserved-prefixes
PUT    /admin/domains/:id
DELETE /admin/domains/:id
GET    /admin/domains-full

GET    /admin/dns-accounts
POST   /admin/dns-accounts
PUT    /admin/dns-accounts/:id
DELETE /admin/dns-accounts/:id
GET    /admin/dns-accounts/:id/zones

POST   /admin/domains/:id/migrations
GET    /admin/domains/:id/migrations
GET    /admin/domains/:id/migrations/:taskId
POST   /admin/domains/:id/migrations/:taskId/retry-failures
POST   /admin/domains/:id/migrations/:taskId/cleanup-source
```

### 全局 DNS 与认领管理

```text
GET    /admin/dns-records
DELETE /admin/dns-records/:id
GET    /admin/dns-bulk-jobs/:id
GET    /admin/dns-bulk-jobs/:id/items
GET    /admin/claimed-subdomains
DELETE /admin/claimed-subdomains/:id
```

### 用户、组和积分

```text
GET  /admin/users
PUT  /admin/users/:id/group
PUT  /admin/users/:id/ban
PUT  /admin/users/:id/unban
GET  /admin/groups
POST /admin/groups
PUT  /admin/groups/:id
DELETE /admin/groups/:id
POST /admin/credits/grant
```

### 系统与客户端

```text
GET    /admin/config
PUT    /admin/config
POST   /admin/config/url-confirm
GET    /admin/client/config
PUT    /admin/client/config
POST   /admin/client/communication-key
DELETE /admin/client/communication-key
GET    /admin/stats
GET    /admin/dns-providers/status
GET    /admin/audit-logs
```

通讯密钥生成响应中的明文只返回一次。任何日志、分析平台或客户端页面缓存都不应持久化该值。

### 审核与 AI

```text
GET    /admin/audit/summary
GET    /admin/audit/cases
GET    /admin/audit/subdomains/:id
GET    /admin/audit/subdomains/:id/scans
POST   /admin/audit/subdomains/:id/restore
DELETE /admin/audit/subdomains/:id/release
POST   /admin/audit/subdomains/:id/rescan
POST   /admin/audit/subdomains/bulk-rescan
GET    /admin/audit/rules
POST   /admin/audit/rules
PUT    /admin/audit/rules/:id
DELETE /admin/audit/rules/:id
PUT    /admin/audit/rules/:id/toggle
GET    /admin/audit/rules/scenarios
POST   /admin/audit/rules/test
GET    /admin/audit/scans
GET    /admin/audit/scans/:id

GET/POST/PUT/DELETE /admin/ai-audit/models...
GET/POST/PUT/DELETE /admin/ai-audit/prompt-templates...
GET/PUT             /admin/ai-audit/reviews...
GET/PUT             /admin/ai-audit/appeals...
```

### 运营配置

管理员还可管理：

- `/admin/notifications` 与 `/admin/notifications/images`
- `/admin/branding` 与 Logo 上传/删除
- `/admin/friend-links`
- `/admin/payment/orders`
- `/admin/emails`、失败重试和 SMTP 测试

## 8. HTTP 状态

| 状态 | 含义 |
| --- | --- |
| 200 | 查询或更新成功 |
| 201 | 创建成功 |
| 204 | CORS 预检等无正文成功 |
| 400 | JSON、参数、业务输入或幂等键无效 |
| 401 | 会话、原生授权码或通讯密钥无效 |
| 403 | 权限、用户组、资源归属或封禁限制 |
| 404 | 资源不存在或不对当前用户可见 |
| 409 | 重复、共存冲突或状态冲突 |
| 429 | 上游/AI 限流或请求频率限制 |
| 500 | 内部错误或外部服务异常 |

客户端同时检查 HTTP 状态和响应 `code`，并显示 `message_key` 对应的本地化提示。

## 9. CORS 与代理

网页来源必须在 `ALLOWED_ORIGINS`。Android 来源为 `https://localhost`。预检需允许 `Authorization`、`X-HL6-Client-Key` 和 `X-Idempotency-Key`，并返回 `Vary: Origin`。

反向代理不得删除 Authorization 或自定义头，不应缓存写请求、OIDC 回调或 SSE。

## 10. 集成安全

- 使用 HTTPS 和受控 API 基址。
- 不在 URL 中放置会话、通讯密钥或第三方凭据。
- 对重试使用稳定幂等键和有界超时。
- 不根据前端菜单推断权限。
- 对分页设置合理上限，不抓取无限列表。
- 尊重 429 与上游错误，使用退避而不是并发重放。
- 上传内容、外部 URL 和支付回调必须在服务端重新验证。
