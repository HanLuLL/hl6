# 系统架构

HL6 采用 React 单页应用、Go API 和 PostgreSQL 的分层架构。生产镜像将前端静态资源嵌入同一个服务端镜像；Android 通过 Capacitor 打包同一份前端 UI，所有业务状态仍由 API 和服务端控制。

## 1. 组件

```text
Browser / Android Capacitor
          |
          | HTTPS + JSON + SSE
          v
Gin Router -> Middleware -> Handler -> Service -> Repository -> PostgreSQL
                                  |          |
                                  |          +-> Redis Streams (optional)
                                  +-> DNS providers / OIDC / SMTP / Payment / AI API
```

- Router：注册公开、登录、管理员路由并组装依赖。
- Middleware：CORS、会话解析、管理员权限。
- Handler：HTTP 参数、响应和错误映射。
- Service：DNS、迁移、审核、通知等业务事务。
- Repository：GORM 查询、锁、分页和持久化。
- Worker：定时扫描、审核任务、豁免恢复。

## 2. 前端与 Android

网页端使用 React Router、TanStack Query、Shadcn UI、Tailwind CSS 和 i18next。Hook 封装查询与 mutation，统一 API 客户端管理基址、会话、幂等和错误。

Android 的 `web/dist` 在构建时复制进 APK。平台插件只负责系统浏览器、深链和安全存储；没有远程 Web UI，也没有另一套原生业务状态。

## 3. 主要数据域

### 身份与权限

- `User`：OIDC 身份、资料、角色、封禁状态与时间。
- `UserGroup`：管理员能力、额度和组级策略。
- 用户资料由 HL6 持久化，OIDC 登录不覆盖用户自定义姓名和头像。

### DNS

- `DNSProviderAccount`：提供商类型、加密凭据和状态。
- `Domain`：平台根域名、Zone、提供商账号和公开规则。
- `Subdomain`：用户认领、状态和归属。
- `DNSRecord`：记录类型、名称、内容、TTL、代理状态和上游 ID。
- `DNSOperation`：Scope + Idempotency Key 的幂等执行记录。
- `DNSBulkJob`：大批量操作和逐项结果。
- `DomainDNSMigration`：跨账号/Zone 迁移状态。

### 积分与支付

- `CreditBalance`：当前余额。
- `CreditTransaction`：发放、消费、签到、推荐和支付流水。
- `PaymentOrder`：网关、渠道、金额、订单状态和回调信息。

### 审核与申诉

- `AuditRule`：匹配条件、场景、动作和启用状态。
- `SubdomainScan`：扫描输入、结论、处置和错误。
- AI 模型、提示词、审核记录：兼容模型调用和管理员二审。
- Appeal：封禁用户申诉及管理员结论。

### 运营

- `SystemConfig`：OIDC、支付、客户端版本、SEO 等动态键值。
- Branding：站点名称、Logo、Favicon、公告和页脚。
- Notification / NotificationRead：站内消息与已读状态。
- EmailLog：邮件发送、失败和重试。
- FriendLink：公开友情链接。

## 4. 配置所有权

环境变量负责启动不可缺少的基础设施和部署边界；数据库动态配置负责管理员可在线调整的业务设置。

字段级优先级通常为：专用环境变量、`APP_URL` 回退、数据库值、可信请求自动检测。敏感数据库配置使用 `ENCRYPTION_KEY` 加密。会话种子可通过 `SESSION_SECRET` 注入，留空时首次启动生成并存入内部配置。

## 5. 认证与会话

### 网页

服务端发起 OIDC Code Flow，验证 State、Token、Issuer、Audience 和 JWKS。回调后创建/关联用户，并签发 HL6 会话。首个用户自动成为管理员。

### Android

客户端先通过通讯密钥请求原生登录地址，系统浏览器完成相同网页 OIDC。服务端生成一次性授权码并跳转应用深链，客户端交换为会话并存入 Keystore 支持的安全存储。

通讯密钥只识别客户端构建；真正的用户权限由会话、角色、用户组和服务端资源归属共同决定。

## 6. 权限模型

- 公开路由：品牌、SEO、公开域名、可用性检查、支付方式、友情链接、客户端版本和 OIDC 状态。
- 登录路由：用户资料、子域名/DNS、积分、订单、通知、申诉。
- 管理员路由：域名、账号、用户组、封禁、系统配置、审核、品牌、支付订单和日志。

管理员中间件接受 `user.Role == admin` 或 `user.Group.IsAdmin == true`。资源级权限仍由 Handler/Service 检查，不能只依赖路由菜单。

## 7. DNS 写入流程

1. Handler 认证用户并绑定 JSON。
2. Validator 校验类型、值、共存和重复规则。
3. Service 检查资源归属、额度和幂等键。
4. Provider 适配器调用上游 API。
5. Repository 在成功后更新本地记录和操作结果。
6. 审计/通知记录需要时同步生成。

批量删除超过阈值后创建异步任务。域名迁移按任务保存进度、失败项和清理状态。

## 8. 审核架构

Audit Scheduler 定时选择需要扫描的子域名，进入进程内队列或 Redis Streams。多个 Worker 执行规则和可选 AI 审查，Audit Service 统一记录结果并执行处置。豁免 Worker 在条件满足时安排恢复或复扫。

Redis 不可用时服务会告警并回退进程内队列。单实例可接受，多实例会失去共享去重和任务分配，应由监控告警。

## 9. 支付流程

1. 用户读取启用的支付方法和积分产品。
2. 服务端创建待支付订单并生成网关签名。
3. 用户跳转网关。
4. 网关调用公开异步通知端点。
5. 服务端验证签名、金额、商户和订单状态。
6. 订单以幂等方式标记成功并写入积分交易。
7. 同步返回页只用于用户体验，不作为到账依据。

## 10. 通知与邮件

通知写入数据库后通过 Broker 推送 SSE 事件。客户端再读取通知详情和未读状态，避免把 SSE 当作唯一数据源。

邮件发送记录持久化，管理员可以测试 SMTP 和重试失败项。封禁生命周期通知包含开始与预计结束时间。

## 11. 部署与交付

Docker 镜像包含 Go 服务端和前端静态资源。`main` 构建 `latest`，正式手动 Release 生成版本标签和 Release Notes。

Android 工作流独立构建签名 APK，部署 Pages `latest.apk`、版本 APK 和 Manifest；正式系统 Release 再把验证后的版本 APK作为永久资产附加。

## 12. 安全边界

- HTTPS 和准确的外部 URL 是 OIDC、CORS 与回调的基础。
- 密钥不进入前端源码、URL、日志、Release Notes 或工作流摘要。
- 外部 URL 需要协议、凭据、主机和路径验证，并对管理配置执行确认。
- DNS、支付和审核写操作以服务端校验和幂等为准。
- 文件上传限制类型、大小和存储路径。
- 所有数据恢复和破坏性管理操作保留审计证据。
