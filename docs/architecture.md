# HL6 架构设计

本文档说明 HL6 的系统架构、目录结构、数据模型和核心设计。

- **GitHub 仓库**：[https://github.com/HanLuLL/hl6](https://github.com/HanLuLL/hl6)
- **原项目仓库**：[https://git.houlang.cloud/houlangcloud/hl6](https://git.houlang.cloud/houlangcloud/hl6)

---

## 目录

- [1. 系统概览](#1-系统概览)
- [2. 目录结构](#2-目录结构)
- [3. 数据模型](#3-数据模型)
- [4. API 设计](#4-api-设计)
- [5. 认证流程](#5-认证流程)
- [6. DNS 提供商抽象](#6-dns-提供商抽象)
- [7. 审计系统](#7-审计系统)
- [8. 支付流程](#8-支付流程)
- [9. SEO 系统](#9-seo-系统)

---

## 1. 系统概览

HL6 是一个域名/子域名管理平台，采用前后端分离架构。

```
┌──────────────────────────────────────────────────────────┐
│                      浏览器（用户）                        │
│              React SPA + TanStack React Query             │
└────────────────────────┬─────────────────────────────────┘
                         │ HTTPS
                         ▼
┌──────────────────────────────────────────────────────────┐
│                    Nginx 反向代理                         │
│              TLS 终止 + 静态资源 + API 路由               │
└──────────┬───────────────────────────────────┬───────────┘
           │                                   │
           ▼                                   ▼
┌─────────────────────┐           ┌─────────────────────────┐
│  前端静态资源        │           │      Go 后端 (Gin)       │
│  web/dist (SPA)     │           │   /api/v1/* + /health    │
│  由后端内嵌服务      │           │                          │
└─────────────────────┘           └──────────┬──────────────┘
                                             │
                            ┌────────────────┼────────────────┐
                            │                │                │
                            ▼                ▼                ▼
                   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
                   │ PostgreSQL 16│  │  Redis (可选) │  │  OIDC 提供商  │
                   │   数据持久化  │  │  审计任务队列  │  │  外部身份认证  │
                   └──────────────┘  └──────────────┘  └──────────────┘
                                             │
                                             ▼
                                   ┌──────────────────────┐
                                   │   DNS 提供商 API      │
                                   │ Cloudflare / 阿里云 / │
                                   │ DNSPod / 华为云 等    │
                                   └──────────────────────┘
```

### 技术栈

| 层 | 技术 |
|----|------|
| 前端 | React 19 + TypeScript + Vite + Tailwind CSS 4 + TanStack React Query + i18next + Shadcn UI |
| 后端 | Go (Gin + GORM) + lestrrat-go/jwx (JWT) |
| 数据库 | PostgreSQL 16 |
| 队列 | 进程内 channel（单实例）/ Redis Stream（多实例） |
| 认证 | OIDC 兼容 + JWT (HS256) |
| 部署 | Docker Compose + 多阶段构建 |

### 单一二进制部署

生产镜像为单一二进制：Go 后端内嵌前端静态资源（`web/dist`），通过 `setupFrontendRoutes` 由 Gin 直接服务 SPA 和静态文件。无需独立前端服务器。

---

## 2. 目录结构

### 2.1 后端 (`server/`)

```
server/
├── cmd/server/
│   └── main.go                 # 入口点：加载配置、数据库迁移、启动服务
├── internal/
│   ├── apperr/                 # 错误码定义
│   ├── config/                 # 环境变量配置加载、URL 解析
│   ├── ctxutil/                # 上下文工具（用户信息传递）
│   ├── handler/                # HTTP 处理器
│   │   ├── auth.go             # 用户信息、资料更新
│   │   ├── oidc*.go            # OIDC 登录/回调/登出/首配
│   │   ├── domain.go           # 域名管理
│   │   ├── subdomain.go        # 子域名认领/释放
│   │   ├── dns.go              # DNS 记录 CRUD
│   │   ├── dns_provider_account.go  # DNS 提供商账户管理
│   │   ├── credit.go           # 积分查询/签到
│   │   ├── payment.go          # 支付订单/回调
│   │   ├── friendlink.go       # 友情链接管理
│   │   ├── audit.go            # 审计规则/扫描记录
│   │   ├── admin*.go           # 管理员接口
│   │   ├── notification*.go    # 通知管理
│   │   ├── seo.go              # SEO 元数据
│   │   ├── branding.go         # 品牌定制
│   │   ├── sse_broker.go       # SSE 推送 broker
│   │   └── referral.go         # 推荐奖励
│   ├── helpers/                # 通用工具（分页、解析、文本）
│   ├── middleware/
│   │   ├── auth.go             # JWT 会话验证
│   │   ├── admin.go            # 管理员权限校验
│   │   └── cors.go             # CORS 中间件
│   ├── model/                  # GORM 数据模型
│   │   ├── models.go           # User/UserGroup/Domain/Subdomain/DNSRecord/FriendLink 等
│   │   ├── credit.go           # Credit 类型（int64 内部单位）
│   │   ├── payment.go          # PaymentOrder
│   │   ├── notification.go     # 通知模型
│   │   ├── dns_provider.go     # DNS 提供商常量
│   │   ├── dns_operation.go    # DNS 操作日志
│   │   ├── dns_bulk_job.go     # DNS 批量任务
│   │   └── domain_dns_migration.go  # 域名迁移任务
│   ├── oidc/                   # OIDC Discovery 解析
│   ├── referral/               # 推荐码生成
│   ├── repository/             # GORM 数据访问层（按领域分文件）
│   ├── router/
│   │   ├── router.go           # 路由总装配、审计栈初始化
│   │   ├── server.go           # HTTP 服务启动 + 优雅关闭
│   │   ├── frontend.go         # SPA 静态资源服务
│   │   ├── handlers.go         # 处理器聚合
│   │   └── routes_*.go         # 按领域分组的路由注册
│   ├── service/                # 业务逻辑层
│   │   ├── cloudflare.go       # Cloudflare DNS 实现
│   │   ├── aliyun_dns.go       # 阿里云 DNS 实现
│   │   ├── dnspod.go           # DNSPod 实现
│   │   ├── huawei_dns.go       # 华为云 DNS 实现
│   │   ├── baidu_cloud_dns.go  # 百度云 DNS 实现
│   │   ├── dns_com.go          # DNS.COM 实现
│   │   ├── dnsla.go            # DNSLA 实现
│   │   ├── westcn_dns.go       # 西部数码 DNS 实现
│   │   ├── route53.go          # AWS Route53 实现
│   │   ├── google_cloud_dns.go # Google Cloud DNS 实现
│   │   ├── provider_factory.go # 提供商工厂（统一构造入口）
│   │   ├── dns_operation_service.go  # DNS 操作编排
│   │   ├── audit_service.go    # 审计扫描与处置
│   │   ├── audit_rule_engine.go # 审计规则匹配引擎
│   │   ├── payment_service.go  # 支付网关（易支付/码支付）
│   │   ├── notification_service.go  # 通知发送
│   │   ├── subdomain_service.go # 子域名释放等操作
│   │   └── domain_migration_service.go  # 域名 DNS 迁移
│   └── worker/
│       ├── audit_scan_worker.go    # 审计扫描消费者
│       ├── audit_scheduler.go      # 定时调度器
│       └── audit_exemption_worker.go  # 豁免重检 worker
├── pkg/
│   ├── audit/                  # 审计抓取器（HTTP/HTTPS 双通道）
│   ├── crypto/                 # AES-256-GCM 加解密
│   ├── queue/                  # 任务队列（进程内 / Redis Stream）
│   ├── response/               # 统一 API 响应格式
│   └── validator/              # DNS 记录验证
├── Dockerfile
├── go.mod
└── go.sum
```

### 2.2 前端 (`web/`)

```
web/
├── src/
│   ├── App.tsx                 # 路由根组件
│   ├── main.tsx                # 应用入口
│   ├── pages/                  # 路由页面
│   │   ├── landing.tsx         # 落地页
│   │   ├── dashboard.tsx       # 仪表盘
│   │   ├── domains.tsx         # 域名列表
│   │   ├── subdomains.tsx      # 我的子域名
│   │   ├── subdomain-detail.tsx # 子域名详情（DNS 管理）
│   │   ├── credits.tsx         # 积分中心
│   │   ├── profile.tsx         # 个人资料
│   │   ├── callback.tsx        # OIDC 回调处理
│   │   ├── admin/              # 管理员页面
│   │   │   ├── audit/          # 审计工作台
│   │   │   ├── domains.tsx     # 域名管理
│   │   │   ├── users.tsx       # 用户管理
│   │   │   ├── dns-records.tsx # DNS 记录管理
│   │   │   ├── credits-settings.tsx
│   │   │   ├── notifications.tsx
│   │   │   └── ...
│   │   └── not-found.tsx
│   ├── components/
│   │   ├── ui/                 # Shadcn/Radix 基础组件
│   │   ├── domain/             # 域名/子域名业务组件
│   │   ├── dns/                # DNS 记录组件
│   │   ├── notification/       # 通知组件
│   │   └── layout/             # 布局组件
│   ├── hooks/                  # 自定义 hooks（封装 React Query）
│   │   ├── use-auth.ts
│   │   ├── use-subdomains.ts
│   │   ├── use-dns-records.ts
│   │   ├── use-credits.ts
│   │   ├── use-payment.ts
│   │   ├── use-friend-links.ts
│   │   ├── use-notifications.ts
│   │   ├── use-sse.ts
│   │   └── ...
│   ├── lib/
│   │   ├── api.ts              # REST API 客户端（Bearer token）
│   │   ├── api-query.ts        # React Query 配置
│   │   ├── prefetch.ts         # 路由预取
│   │   └── utils.ts
│   ├── i18n/                   # i18next 配置（en/zh/zh-Hant/es/ru/ja）
│   └── types/                  # TypeScript 类型定义
├── package.json
└── vite.config.ts              # envDir: ".." 读取根目录 .env
```

---

## 3. 数据模型

### 3.1 核心实体关系

```
User ─┬─< Subdomain ──< DNSRecord
      │      │
      │      └──> Domain ──> DNSProviderAccount
      │
      ├─< CreditBalance
      ├─< CreditTransaction
      ├─< DailyCheckinClaim
      ├─< PaymentOrder
      ├─< UserReferral
      └─< NotificationRead

UserGroup ─< DomainGroupAccess >─ Domain

AuditRule
AuditExemptionPending
SubdomainScan
AuditLog
Notification ─< NotificationRead ─< NotificationImage
BrandingAsset
SystemConfig
```

### 3.2 主要模型

#### User

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | uint | 主键 |
| `external_id` | string | OIDC 提供商的 `sub`，唯一索引 |
| `email` / `name` / `avatar_url` | string | 用户资料 |
| `role` | string | `user` / `admin` |
| `is_banned` / `banned_reason` / `banned_at` | - | 封禁状态 |
| `referral_code` | string | 推荐码，唯一 |
| `group_id` | *uint | 用户组 |

首个注册用户自动成为管理员。

#### Domain

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 域名，唯一 |
| `provider` | string | DNS 提供商标识 |
| `provider_zone_id` | string | 提供商侧 Zone ID |
| `provider_account_id` | uint | 关联的 DNS 提供商账户 |
| `credit_cost` | Credit | 默认认领积分成本 |
| `migration_state` | string | DNS 迁移状态 |

#### Subdomain

| 字段 | 类型 | 说明 |
|------|------|------|
| `domain_id` / `user_id` | uint | 关联域名和用户 |
| `name` | string | 子域名前缀 |
| `fqdn` | string | 完整域名，唯一 |
| `claim_cost` | Credit | 认领时消耗的积分 |
| `status` | string | `active` / `suspended` |

#### DNSRecord

| 字段 | 类型 | 说明 |
|------|------|------|
| `subdomain_id` | uint | 所属子域名 |
| `type` | string | `A` / `AAAA` / `CNAME` / `TXT` |
| `name` / `content` | string | 记录名和值 |
| `ttl` | int | TTL |
| `proxied` | bool | 是否代理（Cloudflare） |
| `provider_record_id` | string | 提供商侧记录 ID |
| `status` | string | `active` / `suspended` |

约束：
- `(subdomain_id, type, content)` 唯一（防重复记录）
- 每个子域名最多一个 CNAME（部分唯一索引）

#### Credit

`Credit` 是 `int64` 的类型别名，内部以 0.1 积分为单位存储（`CreditScale = 10`）。显示时除以 10。JSON 序列化为浮点数。

#### CreditBalance / CreditTransaction

- `CreditBalance`：用户当前余额
- `CreditTransaction`：流水记录，含 `type`、`amount`、`balance_after`、`description_key`（i18n）

#### PaymentOrder

| 字段 | 类型 | 说明 |
|------|------|------|
| `order_no` | string | 订单号，唯一 |
| `gateway` | string | `epay` / `codepay` |
| `payment_method` | string | `alipay` / `wechat` / `qq` |
| `amount` | float64 | CNY 金额 |
| `credits` | Credit | 兑换的积分 |
| `status` | string | `pending` / `paid` / `failed` / `expired` |
| `pay_url` | string | 网关支付链接 |
| `notify_data` | jsonb | 回调原始数据 |

#### AuditRule

管理员配置的内容审核规则，支持：
- **匹配类型**：`keyword`（关键词）/ `regex`（正则）/ `status_eq`（状态码）/ `unreachable`（不可达）
- **目标**：`body` / `title` / `final_url` / `status_code`
- **处置**：`observe`（观察）/ `delete_dns`（删除 DNS）/ `site`（释放子域名）/ `user`（封禁用户全部子域名）
- **豁免**：支持豁免等待 + 定时重检

#### SubdomainScan

每次扫描的记录，包含 HTTP 状态码、最终 URL、匹配规则、证据片段、内容哈希等。

#### Notification / NotificationRead

- 通知支持按用户/用户组定向推送
- `NotificationRead` 记录用户已读状态
- 支持图片附件

#### DNSProviderAccount

DNS 提供商凭据账户，凭据以加密形式存储（`ENCRYPTION_KEY` 启用时）。状态含 `active` / `degraded` / `disabled`。

### 3.3 SystemConfig

键值对存储系统配置，如：
- `registration_bonus_credits`：注册赠送积分
- `referral_enabled` / `referral_inviter_credits` / `referral_invitee_credits`
- `daily_checkin_enabled` / `daily_checkin_credits`
- `brand_name` / `seo_*`：品牌与 SEO 配置
- `_internal_session_secret`：会话密钥（自动生成）

---

## 4. API 设计

### 4.1 统一响应格式

所有 API 返回统一的 `ApiResponse` 结构：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

- `code: 0` 表示成功，`-1` 表示失败
- `message` 为可读消息，`message_key` 为 i18n 键（前端用于翻译）

### 4.2 分页响应

```json
{
  "code": 0,
  "message": "ok",
  "data": [],
  "total": 100,
  "page": 1,
  "per_page": 20
}
```

部分接口使用 `offset` / `limit` 形式。

### 4.3 路由组织

所有 API 前缀 `/api/v1`，按领域分组：

| 分组 | 路径前缀 | 认证 |
|------|---------|------|
| 认证 | `/api/v1/auth/*` | 部分（登录/回调公开） |
| 品牌 | `/api/v1/branding` | 公开 |
| SEO | `/api/v1/seo/meta`、`/robots.txt`、`/sitemap.xml` | 公开 |
| 域名 | `/api/v1/domains` | 认证 |
| 子域名 | `/api/v1/subdomains` | 认证 |
| DNS | `/api/v1/dns/*` | 认证 |
| 积分 | `/api/v1/credits/*` | 认证 |
| 通知 | `/api/v1/notifications/*` | 认证 |
| 支付 | `/api/v1/payment/*` | 认证（回调公开） |
| 友情链接 | `/api/v1/friend-links` | 公开（后台管理需管理员权限） |
| 管理员 | `/api/v1/admin/*` | 认证 + 管理员权限 |

### 4.4 错误处理

使用 i18n key 返回错误，前端通过 `message_key` 匹配翻译：

```json
{
  "code": -1,
  "message": "error.invalidToken",
  "message_key": "error.invalidToken"
}
```

---

## 5. 认证流程

### 5.1 OIDC 登录流程

```
浏览器                    HL6 后端                  OIDC 提供商
  │                          │                          │
  │ 1. GET /api/v1/auth/login│                          │
  │ ────────────────────────>│                          │
  │                          │ 2. Discovery             │
  │                          │ ────────────────────────>│
  │                          │ <────────────────────────│
  │ 3. 302 重定向到 OIDC     │                          │
  │ <────────────────────────│                          │
  │ 4. 跳转到 OIDC 登录页    │                          │
  │ ───────────────────────────────────────────────────>│
  │                          │                          │
  │ 5. 用户登录授权          │                          │
  │ <──────────────────────────────────────────────────│
  │ 6. 302 回调带 code       │                          │
  │ ────────────────────────>│                          │
  │                          │ 7. 用 code 换 token      │
  │                          │ ────────────────────────>│
  │                          │ <────────────────────────│
  │                          │ 8. 解析 id_token，       │
  │                          │    提取 sub/email/name   │
  │                          │ 9. 查找或创建 User       │
  │                          │ 10. 生成 JWT (HS256)     │
  │                          │     设置 hl6_session     │
  │ 11. 302 回前端           │                          │
  │ <────────────────────────│                          │
```

### 5.2 会话管理

- **会话 Cookie**：`hl6_session`，HttpOnly，SameSite=Lax
- **JWT 签名**：HS256，密钥来自 `SESSION_SECRET`（env 或数据库）
- **Issuer/Audience**：均为 `hl6`
- **Subject**：用户的 `external_id`（OIDC `sub`）

### 5.3 中间件

```
请求 → CORS → Auth.Required() → AdminRequired()（管理员路由） → Handler
```

- `Auth.Required()`：解析 Cookie 中的 JWT，加载用户，检查封禁状态
- `AdminRequired()`：校验 `user.Role == "admin"` **或** `user.Group.IsAdmin == true`（支持用户组管理员）
- 封禁用户仅允许访问 `/api/v1/auth/logout`

### 5.4 OIDC Discovery

启动时通过 `OIDC_ISSUER/.well-known/openid-configuration` 自动发现 `authorization_endpoint`、`token_endpoint`、`jwks_uri`、`end_session_endpoint`。

不支持 `end_session_endpoint` 的提供商（Google、GitLab），登出时直接返回前端 URL。

### 5.5 首配向导

`OIDC_ISSUER` / `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET` 均为空且系统无用户时，`/api/v1/auth/oidc/bootstrap` 允许匿名提交配置。系统有用户后该入口关闭。

---

## 6. DNS 提供商抽象

### 6.1 接口定义

```go
type DNSProviderClient interface {
    ListZones(ctx context.Context) ([]ZoneInfo, error)
    CreateRecord(ctx context.Context, zoneID, recordType, name, content string, ttl int, proxied bool) (string, error)
    UpdateRecord(ctx context.Context, zoneID, recordID, recordType, name, content string, ttl int, proxied bool) error
    DeleteRecord(ctx context.Context, zoneID, recordID string) error
    FindRecord(ctx context.Context, zoneID, recordType, name, content string) (string, error)
}
```

### 6.2 工厂模式

`BuildProviderClient(provider, credentials)` 根据 provider 标识构造对应实现。凭据统一解析为 `map[string]string`，支持多种字段别名（如 `api_token` / `ak` / `access_key_id`）。

### 6.3 支持的提供商

| 提供商 | 标识 | 实现文件 |
|--------|------|---------|
| Cloudflare | `cloudflare` | `cloudflare.go` |
| DNSPod | `dnspod` | `dnspod.go` |
| 阿里云 DNS | `aliyun_dns` | `aliyun_dns.go` |
| 华为云 DNS | `huawei_cloud_dns` | `huawei_dns.go` |
| AWS Route53 | `aws_route53` | `route53.go` |
| Google Cloud DNS | `google_cloud_dns` | `google_cloud_dns.go` |
| 百度云 DNS | `baidu_cloud_dns` | `baidu_cloud_dns.go` |
| DNS.COM | `dns_com` | `dns_com.go` |
| DNSLA | `dnsla` | `dnsla.go` |
| 西部数码 DNS | `westcn_dns` | `westcn_dns.go` |

### 6.4 凭据加密

`ENCRYPTION_KEY` 配置后，DNS 提供商凭据使用 AES-256-GCM 加密存储。`pkg/crypto` 提供加解密能力。

### 6.5 DNS 操作编排

`DNSOperationService` 封装 DNS 操作的编排逻辑，包含：
- 记录 CRUD（同步数据库与提供商）
- 批量操作（超过阈值切换异步任务）
- 操作日志记录（`DNSOperationRequest` / `DNSOperationEvent`）
- 域名 DNS 迁移（`DomainMigrationService`）

---

## 7. 审计系统

### 7.1 架构

```
┌─────────────────────────────────────────────────────┐
│                   审计系统架构                       │
└─────────────────────────────────────────────────────┘

  ┌─────────────┐     ┌──────────────┐     ┌──────────────┐
  │ Scheduler   │────>│  Task Queue  │────>│   Workers    │
  │ (定时调度)  │     │ (进程内/Redis)│     │ (并发扫描)   │
  └─────────────┘     └──────────────┘     └──────┬───────┘
                                                  │
                                                  ▼
                                       ┌──────────────────┐
                                       │  AuditService    │
                                       │  ScanSubdomain   │
                                       └────────┬─────────┘
                                                │
                              ┌─────────────────┼─────────────────┐
                              │                 │                 │
                              ▼                 ▼                 ▼
                    ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
                    │ SafeFetcher  │  │ RuleEngine   │  │  处置动作     │
                    │ (HTTP/HTTPS  │  │ (规则匹配)   │  │ (释放/封禁/   │
                    │  双通道抓取)  │  │              │  │  删DNS/观察) │
                    └──────────────┘  └──────────────┘  └──────────────┘
```

### 7.2 调度器（AuditScheduler）

按 `AUDIT_SCAN_INTERVAL`（默认 30 分钟）定期扫描所有活跃子域名，将扫描任务推入队列。使用去重机制避免重复调度。

### 7.3 任务队列

- **进程内队列**（`InProcQueue`）：单实例部署，channel 实现
- **Redis Stream**（`RedisStream`）：多实例部署，支持消费者组和 AutoClaim

`REDIS_ADDR` 配置时自动使用 Redis，否则回退进程内。

### 7.4 扫描 Worker

`AUDIT_SCAN_WORKER_COUNT`（默认 2）个 worker 并发消费队列。每个 worker：
1. 从队列读取任务
2. 调用 `AuditService.ScanSubdomain`
3. Ack 任务

AutoClaim 机制处理超时未 Ack 的任务（2 分钟空闲后重新分配）。

### 7.5 扫描流程

`ScanSubdomain` 执行步骤：

1. 加载子域名和 DNS 记录
2. 检查是否有可扫描记录（A/AAAA/CNAME）
3. **双通道抓取**（HTTP + HTTPS），`SafeFetcher.FetchDualConfirmed`
4. 计算内容哈希，若与上次 clean 扫描相同且规则未更新则跳过规则匹配
5. **规则匹配**：`AuditRuleEngine.MatchAll` 对所有启用的规则匹配
6. 命中违规时按规则处置档位执行：
   - `observe`：仅记录 + 通知
   - `delete_dns`：删除可扫描的 DNS 记录
   - `site`：释放子域名（删除所有 DNS 记录 + 解除认领）
   - `user`：封禁用户所有子域名
7. 持久化 `SubdomainScan` 记录

### 7.6 豁免机制

规则可启用豁免：首次违规时创建 `AuditExemptionPending`，在 `ExemptRecheckMinutes` 后由 `AuditExemptionWorker` 重新扫描。豁免期间不执行处置。

### 7.7 恢复机制

`RestoreSubdomain` 恢复被封禁的子域名：重建 DNS 记录 + 恢复状态。部分失败时执行补偿（删除已重建的记录）。

---

## 8. 支付流程

### 8.1 流程图

```
用户                HL6 后端              支付网关
 │                     │                     │
 │ 1. POST /payment/orders                    │
 │    {gateway, method, amount}               │
 │ ───────────────────>│                     │
 │                     │ 2. 创建 PaymentOrder │
 │                     │    status=pending    │
 │                     │ 3. 构造支付 URL      │
 │                     │    (签名+参数)       │
 │ 4. 返回 pay_url     │                     │
 │ <───────────────────│                     │
 │ 5. 跳转到支付网关   │                     │
 │ ──────────────────────────────────────────>│
 │                     │                     │
 │ 6. 用户完成支付     │                     │
 │ <─────────────────────────────────────────│
 │ 7. 返回支付成功页   │                     │
 │   /api/v1/payment/return                  │
 │                     │                     │
 │                     │ 8. 异步回调通知      │
 │                     │ <───────────────────│
 │                     │ 9. 验证签名          │
 │                     │ 10. 更新订单 paid    │
 │                     │ 11. 增加用户积分     │
 │                     │ 12. 返回 success     │
 │                     │ ───────────────────>│
```

### 8.2 订单生命周期

1. **创建**：`POST /api/v1/payment/orders`，生成订单号 `P{timestamp}{userID}`，有效期 30 分钟
2. **支付**：用户跳转到 `pay_url` 完成支付
3. **回调**：网关异步调用 `/api/v1/payment/{epay|codepay}/notify`
4. **到账**：签名验证通过后，订单标记 `paid`，积分入账

### 8.3 签名算法

两者均使用 MD5：

- **易支付**：参数按字典序排序 + 拼接 + 追加 Key + MD5
- **码支付**：固定字段顺序拼接 + 追加 Key + MD5

### 8.4 积分兑换

1 CNY = 1 显示积分 = 10 内部 Credit 单位。充值范围 1 - 10000 CNY。

---

## 9. SEO 系统

### 9.1 公开端点

| 路径 | 说明 |
|------|------|
| `/robots.txt` | 爬虫规则 |
| `/sitemap.xml` | 站点地图 |
| `/api/v1/seo/meta` | SEO 元数据（站点名、描述、关键词等） |

### 9.2 可配置项

通过 `SystemConfig` 存储，管理员后台配置：

- `seo_description` / `seo_keywords` / `seo_author`
- `seo_og_image`：Open Graph 图片
- `seo_twitter_card` / `seo_twitter_site`
- `seo_analytics_id`：Google Analytics ID

### 9.3 前端集成

前端通过 `use-seo` hook 获取 SEO 元数据，动态设置 `<meta>` 标签和文档标题。React Router 的路由切换会更新对应的页面标题。
