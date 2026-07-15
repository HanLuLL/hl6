# 运维与故障排查

本文用于生产值班、监控和事故处理。部署参数与反向代理基线见[生产部署与升级](deployment.md)。处理事故时先保护数据和证据，再进行变更。

## 1. 快速状态检查

```bash
docker compose --env-file .env -f docker-compose.prod.yml ps
docker compose --env-file .env -f docker-compose.prod.yml logs --tail=100 app
curl --fail --show-error https://domain.example.com/health
```

健康响应中的 `status=degraded` 或 `database=false` 表示进程仍存活但数据库不可用。此时不要反复重启应用，应先检查 PostgreSQL 容器、磁盘、连接数和凭据。

## 2. 日志

```bash
# 实时应用日志
docker compose --env-file .env -f docker-compose.prod.yml logs -f app

# 最近 30 分钟
docker compose --env-file .env -f docker-compose.prod.yml logs --since=30m app

# PostgreSQL
docker compose --env-file .env -f docker-compose.prod.yml logs --tail=200 postgres
```

日志可以记录请求路径、状态和错误，但不得记录 OIDC Secret、DNS Token、AI API Key、支付密钥、Android 通讯密钥或完整会话令牌。向外部提交日志前先脱敏域名、邮箱、用户 ID 和请求头。

## 3. 数据库问题

### 连接失败

1. 检查 `postgres` 是否 healthy。
2. 核对 `POSTGRES_DB`、`POSTGRES_USER`、`POSTGRES_PASSWORD` 与 Compose 生成的 `DATABASE_URL`。
3. 检查卷是否只读、磁盘是否已满。
4. 查看连接数与长事务。

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec postgres \
  sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "select now();"'
```

不要在未备份时删除 `postgres-data` 卷。迁移失败时保留完整日志和数据库快照，再决定升级修复或恢复。

### 会话突然全部失效

检查 `SESSION_SECRET` 是否被更换、容器是否连接到另一套数据库，以及数据库内部 `_internal_session_secret` 是否可读。空 `SESSION_SECRET` 只适合让服务首次生成并持久化；多实例必须共享同一值或同一数据库内部密钥。

### 敏感配置无法解密

通常由 `ENCRYPTION_KEY` 缺失或错误导致。恢复原密钥后重启，不要用新密钥覆盖保存失败的配置。若原密钥永久丢失，只能重新录入受影响的 OIDC、DNS 或 AI 凭据。

## 4. OIDC 与登录

| 现象 | 检查 |
| --- | --- |
| 登录后回到错误地址 | `APP_URL`、`BACKEND_URL`、反向代理 `X-Forwarded-Proto/Host` |
| `redirect_uri` 不匹配 | 提供商只登记 `https://域名/api/v1/auth/callback` |
| 发现配置但登录失败 | Issuer Discovery、Client ID、Secret、服务器时钟和证书链 |
| 后台修改 OIDC 不生效 | 对应环境变量仍覆盖数据库字段 |
| Android 登录按钮网络失败 | CORS 预检、通讯密钥、域名与 `/auth/native/start` |
| Android 授权后不返回 App | 包名对应的深链、已安装版本和系统默认浏览器 |

详细流程与预检命令见[OIDC 认证配置](oidc.md)。

## 5. DNS 操作失败

1. 在后台 DNS 提供商状态页确认账号启用且区域可列出。
2. 检查 API Token 最小权限、Zone/Domain 绑定和上游限流。
3. 确认记录值符合 A、AAAA、CNAME 或 TXT 规则。
4. 检查 CNAME 与其他记录的共存冲突、重复记录和保留前缀。
5. 对异步批量任务，查询管理员批量任务详情和失败项，不要重复提交相同操作。

写操作携带 `X-Idempotency-Key`。客户端超时后可用同一键安全重试；生成新键可能创建第二次业务尝试。

## 6. 审核队列与 AI 审查

### 进程内队列

单实例默认使用进程内队列，重启会丢失尚未开始的任务。检查审核扫描列表并对遗漏子域名执行复扫。

### Redis Streams

设置 `REDIS_ADDR` 后，启动日志应显示 Redis 队列已启用。若连接失败，服务会告警并回退到进程内队列。多实例环境出现回退时应视为严重告警，因为各实例无法共享任务和去重状态。

### AI 审查失败

- 检查模型配置是否启用、API 地址和模型名是否兼容。
- API Key 只显示脱敏值；测试失败不应通过日志输出明文。
- 检查 RPM、超时、温度和提示词变量。
- AI 结论进入审核记录后仍由服务端处置策略和管理员二审控制。

## 7. 通知、SSE 与邮件

SSE 端点为 `/api/v1/notifications/sse`。反向代理必须关闭缓冲并延长读取超时。CDN 不应缓存 SSE。

邮件未发送时：

1. 查看后台邮件记录和失败原因。
2. 核对 SMTP 主机、端口、TLS、发件人和凭据。
3. 使用后台测试邮件功能。
4. 修复后对失败记录执行重试。

封禁通知应包含封禁开始时间和预计解封时间；永久封禁必须明确标记为无自动解封时间。

## 8. 支付与积分

支付回调必须从公网访问，并保持原始查询/表单参数。出现订单未到账时：

1. 在后台订单列表查找订单号和网关交易号。
2. 检查网关地址、商户 ID、密钥和渠道开关。
3. 验证回调签名错误日志与服务器时钟。
4. 确认订单状态后再人工处理，避免重复发放积分。

易支付与码支付配置存储在后台，不读取旧版支付环境变量。积分交易记录是审计依据，不应直接修改余额表绕过交易流水。

## 9. Android 客户端

| 现象 | 处理 |
| --- | --- |
| `Failed to fetch` | 检查公网 HTTPS、Capacitor 来源 CORS 和 `X-HL6-Client-Key`/`X-Idempotency-Key` |
| `invalid client key` | 密钥已轮换或作废，使用新密钥重新构建 APK |
| 登录后回不到 App | 检查 `hl6.<applicationId>://auth/callback` 与安装包包名 |
| 无法覆盖安装 | 新旧 APK 包名或 release keystore 不一致 |
| 强制更新循环 | 后台版本号、客户端下载链接和本地 `versionName` 不一致 |
| 下载内容不是 APK | 检查 Pages/Release URL、HTTP 状态、Content-Type 与 SHA-256 |

完整构建和发布流程见[Android 客户端](android-client.md)。

## 10. 性能与容量

- 提高 `AUDIT_SCAN_WORKER_COUNT` 前先观察上游 DNS/AI 限流和数据库负载。
- 大量 DNS 删除超过 `DNS_BATCH_ASYNC_THRESHOLD` 后进入异步任务，不应通过无限提高阈值规避队列。
- PostgreSQL 性能问题先检查慢查询、锁、连接池和磁盘，再考虑增加实例资源。
- 通知 SSE 每个在线用户保持长连接，反向代理和文件描述符限制需按并发量调整。
- 多实例必须使用共享 Redis 审核队列，并在负载均衡层正确转发真实来源和 HTTPS 信息。

## 11. 事故处理清单

1. 记录开始时间、受影响功能、版本标签和最近变更。
2. 保护数据库、日志和当前容器信息；必要时先创建备份。
3. 用 `/health` 和最小只读请求确定故障边界。
4. 停止有风险的自动任务或写流量，但避免无证据地删除数据。
5. 修复后验证登录、DNS、通知、支付和 Android 版本检查等受影响链路。
6. 记录根因和长期防护；只把用户可见的正式能力变化写入 Release Notes。
