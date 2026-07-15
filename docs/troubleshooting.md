# HL6 故障排查指南

本指南覆盖 HL6 部署与运行中常见问题的诊断与解决方案。

---

## 目录

- [1. 数据库连接问题](#1-数据库连接问题)
- [2. OIDC 认证失败](#2-oidc-认证失败)
- [3. Docker 拉取问题](#3-docker-拉取问题)
- [4. 支付回调未收到](#4-支付回调未收到)
- [5. DNS 操作失败](#5-dns-操作失败)
- [6. SSE 通知不工作](#6-sse-通知不工作)
- [7. 性能调优](#7-性能调优)
- [8. 日志位置与调试方法](#8-日志位置与调试方法)
- [9. 健康检查使用](#9-健康检查使用)

---

## 1. 数据库连接问题

### 1.1 症状

- 启动日志出现 `failed to connect database`
- 容器反复重启
- API 返回 500 错误

### 1.2 排查步骤

**检查 PostgreSQL 容器状态：**

```bash
docker ps | grep postgres
docker logs hl6-postgres
```

**验证连接串：**

确认 `DATABASE_URL` 格式正确：

```
postgres://<user>:<password>@<host>:<port>/<dbname>?sslmode=disable
```

compose 部署时 host 应为服务名 `postgres`，而非 `localhost`。

**手动测试连接：**

```bash
docker exec hl6-postgres psql -U hl6 -d hl6 -c "SELECT 1;"
```

### 1.3 常见原因

| 原因 | 解决方案 |
|------|---------|
| PostgreSQL 未就绪就启动 app | compose 已配置 `depends_on` + `healthcheck`，确保使用 `condition: service_healthy` |
| 密码包含特殊字符 | URL 编码特殊字符，如 `@` → `%40`、`#` → `%23` |
| 端口冲突 | 宿主机 5432 被占用，修改 compose 端口映射 |
| 数据库不存在 | `POSTGRES_DB` 指定的库首次启动时自动创建 |
| sslmode 配置错误 | 容器内连接通常用 `sslmode=disable`；外部数据库可能需要 `sslmode=require` |

### 1.4 使用外部数据库

如使用外部托管 PostgreSQL，移除 compose 中的 `postgres` 服务，将 `DATABASE_URL` 指向外部地址：

```dotenv
DATABASE_URL=postgres://hl6:password@db.internal.example.com:5432/hl6?sslmode=require
```

同时移除 `app` 服务的 `depends_on` 配置。

---

## 2. OIDC 认证失败

### 2.1 症状

- 登录页跳转后报错
- 回调地址返回 400/500
- `OIDC discovery failed` 日志

### 2.2 排查步骤

**验证 Discovery endpoint 可访问：**

```bash
curl -s https://your-oidc-provider.example.com/.well-known/openid-configuration | head
```

应返回包含 `issuer`、`authorization_endpoint`、`token_endpoint` 的 JSON。

**检查回调地址：**

在 OIDC 提供商控制台确认 Redirect URI 为：

```
https://your-hl6-domain.com/api/v1/auth/callback
```

地址必须完全匹配（包括协议、路径、是否有尾部斜杠）。

**检查环境变量：**

```bash
docker exec hl6-app env | grep OIDC
```

确认 `OIDC_ISSUER`、`OIDC_CLIENT_ID`、`OIDC_CLIENT_SECRET` 已正确注入。

### 2.3 常见原因

| 原因 | 解决方案 |
|------|---------|
| Issuer URL 格式错误 | 参阅 [OIDC 配置指南](./oidc.md)，不同提供商格式不同 |
| Issuer 含/不含尾部斜杠 | Authentik、Auth0 需要尾部斜杠；Keycloak、Google 不需要 |
| Client Secret 错误 | 在提供商控制台重新复制 Secret（部分提供商仅显示一次） |
| 网络不通 | 容器无法访问 OIDC 提供商，检查 DNS 和出站网络 |
| 回调地址不匹配 | 协议（http/https）、域名、路径必须与提供商配置完全一致 |
| `FRONTEND_URL`/`BACKEND_URL` 未配置 | 回调跳转依赖正确的 URL 配置 |

### 2.4 Authing 特殊注意事项

- Issuer 必须包含 `/oidc` 后缀：`https://<app-domain>.authing.cn/oidc`
- id_token 签名算法必须选择 RS256
- 换取/检验/撤回 token 的身份验证方式必须设为 `none`
- 支持无 kid（Key ID）的 JWT 令牌验证

### 2.5 首配向导

若 `OIDC_*` 三项均为空：

- 系统无用户时：登录页弹出首配向导，可在线配置
- 系统已有用户时：首配入口关闭，需通过环境变量或管理员后台恢复

---

## 3. Docker 拉取问题

HL6 的国际正式镜像为 `ghcr.io/hanlull/hl6`；中国大陆环境应使用 `ghcr.milu.moe/hanlull/hl6` 代理地址。两者的标签规则、Compose 配置和摘要校验见 [容器镜像说明](container-images.md)。

### 3.1 症状

- `docker pull` 超时或失败
- `image not found` 错误
- 拉取速度极慢

### 3.2 国内网络选择

不要为了拉取 HL6 而向 Docker daemon 全局加入来源不明的镜像加速器。该设置会影响机器上所有镜像的供应链。

中国大陆网络请仅将 HL6 服务的 `HL6_IMAGE` 设置为：

```dotenv
HL6_IMAGE=ghcr.milu.moe/hanlull/hl6:latest
```

然后执行：

```bash
docker compose -f docker-compose.prod.yml --env-file .env pull app
docker compose -f docker-compose.prod.yml --env-file .env up -d app
```

国际网络则使用默认值 `ghcr.io/hanlull/hl6:latest`。完整标签规则和摘要校验见 [容器镜像说明](container-images.md)。

### 3.3 私有镜像仓库

若国际 GHCR 或中国大陆代理均不可达，可：

1. 从 GitHub 仓库自行构建镜像：
   ```bash
   git clone https://github.com/HanLuLL/hl6.git
   cd hl6
   docker build -t hl6:local .
   ```

2. 修改 `docker-compose.prod.yml` 使用本地构建的镜像：
   ```yaml
   app:
     image: hl6:local
   ```

### 3.4 PostgreSQL 镜像

`postgres:16-alpine` 拉取失败时，在 `.env` 中指定组织已审批的 PostgreSQL 镜像地址，再重新拉取服务：

```dotenv
POSTGRES_IMAGE=your-approved-registry.example/library/postgres:16-alpine
```

```bash
docker compose -f docker-compose.prod.yml --env-file .env pull postgres
docker compose -f docker-compose.prod.yml --env-file .env up -d postgres
```

---

## 4. 支付回调未收到

### 4.1 症状

- 用户支付成功但积分未到账
- 订单状态停留在 `pending`
- 网关后台显示回调失败

### 4.2 排查步骤

**确认回调地址配置：**

在支付网关商户后台确认异步通知地址（Notify URL）为：

| 网关 | 回调地址 |
|------|---------|
| 易支付 | `https://hl6.example.com/api/v1/payment/epay/notify` |
| 码支付 | `https://hl6.example.com/api/v1/payment/codepay/notify` |

**验证回调可达性：**

```bash
curl -I https://hl6.example.com/api/v1/payment/epay/notify
```

应返回 HTTP 状态码（非 404/502）。

**检查订单状态：**

```bash
docker exec hl6-postgres psql -U hl6 -d hl6 -c \
  "SELECT order_no, status, amount, credits, paid_at FROM payment_orders ORDER BY created_at DESC LIMIT 10;"
```

**查看应用日志：**

```bash
docker logs hl6-app | grep -i "payment\|callback\|epay\|codepay"
```

### 4.3 常见原因

| 原因 | 解决方案 |
|------|---------|
| 回调地址为 HTTP 而非 HTTPS | 部分网关要求 HTTPS；部署 TLS 反向代理 |
| 反向代理未放行回调路径 | 确保 `/api/v1/payment/` 路径不被拦截 |
| 商户密钥错误 | 签名校验失败，回调会被拒绝；前往后台「系统设置 → 支付配置」核对易支付/码支付商户密钥 |
| 防火墙拦截 | 网关服务器无法访问 HL6 回调地址 |
| 订单已过期 | 订单有效期 30 分钟，过期后回调会被忽略 |

### 4.4 手动补单

如确认用户已支付但积分未到账，可通过数据库排查后联系开发人员处理。**不建议直接修改数据库**，应通过审计日志追溯。

---

## 5. DNS 操作失败

### 5.1 症状

- 创建/更新/删除 DNS 记录报错
- 子域名认领失败
- 审计扫描中 DNS 操作失败

### 5.2 排查步骤

**检查 DNS 提供商账户状态：**

```bash
docker exec hl6-postgres psql -U hl6 -d hl6 -c \
  "SELECT id, provider, name, status, last_error_message FROM dns_provider_accounts;"
```

`status` 为 `degraded` 或 `disabled` 表示账户存在问题。

**验证凭据：**

在管理员后台 → DNS 提供商账户页面，点击验证按钮重新校验凭据。

**查看应用日志：**

```bash
docker logs hl6-app | grep -i "dns\|cloudflare\|provider"
```

### 5.3 常见原因

| 原因 | 解决方案 |
|------|---------|
| API Token 过期或权限不足 | 在 DNS 提供商控制台重新生成 Token |
| Zone ID 错误 | 确认域名对应的 Zone ID 正确 |
| 凭据格式错误 | JSON 格式凭据需为合法 JSON 对象 |
| 速率限制 | DNS 提供商 API 限流，降低操作频率 |
| 记录冲突 | 同类型同内容的记录已存在（唯一索引约束） |
| CNAME 共存规则 | 每个子域名只能有一个 CNAME 记录 |

### 5.4 支持的 DNS 提供商

| 提供商 | 标识 | 必需凭据 |
|--------|------|---------|
| Cloudflare | `cloudflare` | `api_token` |
| DNSPod | `dnspod` | `secret_id`, `secret_key` |
| 阿里云 DNS | `aliyun_dns` | `access_key_id`, `access_key_secret` |
| 华为云 DNS | `huawei_cloud_dns` | `ak`, `sk` |
| AWS Route53 | `aws_route53` | `access_key_id`, `access_key_secret` |
| Google Cloud DNS | `google_cloud_dns` | `service_account_json` |
| 百度云 DNS | `baidu_cloud_dns` | `access_key`, `secret_key` |
| DNS.COM | `dns_com` | `api_id`, `api_key` |
| DNSLA | `dnsla` | `api_id`, `api_secret` |
| 西部数码 DNS | `westcn_dns` | `username`, `password` |

### 5.5 加密密钥问题

若 `ENCRYPTION_KEY` 在部署后更换，已加密的 DNS 凭据将无法解密。需在管理员后台重新录入所有 DNS 提供商凭据。

---

## 6. SSE 通知不工作

### 6.1 症状

- 通知铃铛不实时更新
- 需刷新页面才能看到新通知
- 浏览器控制台 SSE 连接报错

### 6.2 排查步骤

**验证 SSE 端点：**

```bash
curl -N -H "Accept: text/event-stream" \
  -H "Cookie: hl6_session=<your-session>" \
  https://hl6.example.com/api/v1/notifications/sse
```

应收到 `Content-Type: text/event-stream` 的持续流。

**检查浏览器控制台：**

F12 → Network → 筛选 EventSource，查看连接状态和错误信息。

### 6.3 常见原因

| 原因 | 解决方案 |
|------|---------|
| 反向代理缓冲了 SSE | Nginx 需设置 `proxy_buffering off; proxy_cache off;` |
| 代理超时断开 | 设置 `proxy_read_timeout 86400s;` |
| 多实例部署无共享 SSE | SSE 基于进程内 broker，多实例时通知仅送达当前实例连接的用户；使用 Redis 队列可缓解审计任务分发，但 SSE 仍需 sticky session |
| 浏览器连接数限制 | 每域名最多 6 个 HTTP/1.1 连接；启用 HTTP/2 可解决 |
| Cookie 未携带 | SSE 连接需要会话 Cookie，确认同源或 CORS 配置正确 |

### 6.4 Nginx SSE 配置

```nginx
location /api/v1/notifications/sse {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_buffering off;
    proxy_cache off;
    proxy_read_timeout 86400s;
}
```

---

## 7. 性能调优

### 7.1 Redis 审计队列

单实例部署使用进程内队列即可。**多实例部署必须配置 Redis**，否则审计任务无法跨实例分发。

配置 Redis：

```dotenv
REDIS_ADDR=redis:6379
```

在 compose 中添加 Redis 服务：

```yaml
services:
  redis:
    image: redis:7-alpine
    container_name: hl6-redis
    restart: unless-stopped
    volumes:
      - redis-data:/data
```

HL6 使用 Redis Stream 作为审计任务队列，自动创建消费者组。Redis 不可用时自动回退到进程内队列（日志会出现 `WARN: REDIS_ADDR configured but ping failed`）。

### 7.2 Worker 数量调优

| 变量 | 默认值 | 调优建议 |
|------|--------|---------|
| `AUDIT_SCAN_WORKER_COUNT` | `2` | 子域名数量 > 1000 时建议 `4-8`；CPU 密集型，按核数调整 |
| `AUDIT_SCAN_INTERVAL` | `30m` | 子域名多时可降低到 `10m`；注意 DNS 提供商 API 限流 |
| `AUDIT_SCAN_TIMEOUT` | `15s` | 网络较差时提高到 `30s` |

### 7.3 数据库连接池

GORM 默认连接池配置。高并发场景可在 `DATABASE_URL` 中添加参数，或通过环境变量调整（如 `pool_max_conns`）。

### 7.4 DNS 批量操作

`DNS_BATCH_ASYNC_THRESHOLD`（默认 200）控制何时切换为异步批量任务。子域名记录数较多时，同步操作会阻塞请求，可适当降低阈值。

---

## 8. 日志位置与调试方法

### 8.1 容器日志

```bash
# 实时查看
docker logs -f hl6-app

# 最近 100 行
docker logs --tail 100 hl6-app

# 按时间筛选
docker logs --since 30m hl6-app

# 筛选错误
docker logs hl6-app 2>&1 | grep -i "error\|fatal\|panic"
```

### 8.2 PostgreSQL 日志

```bash
docker logs hl6-postgres
```

### 8.3 日志级别

HL6 使用 Go 标准 `log` 和 `log/slog`：

- `log.Println` / `log.Printf`：常规启动信息
- `slog.Info`：审计等业务日志
- `slog.Warn`：可恢复的异常
- `slog.Error`：操作失败
- `log.Fatal`：致命错误，进程退出

### 8.4 数据库排查

直接查询数据库定位问题：

```bash
# 查看用户
docker exec hl6-postgres psql -U hl6 -d hl6 -c \
  "SELECT id, external_id, email, role, is_banned FROM users;"

# 查看子域名状态
docker exec hl6-postgres psql -U hl6 -d hl6 -c \
  "SELECT id, fqdn, status, suspended_reason FROM subdomains ORDER BY created_at DESC LIMIT 20;"

# 查看审计扫描记录
docker exec hl6-postgres psql -U hl6 -d hl6 -c \
  "SELECT id, fqdn, status, http_status_code, created_at FROM subdomain_scans ORDER BY created_at DESC LIMIT 20;"

# 查看系统配置
docker exec hl6-postgres psql -U hl6 -d hl6 -c \
  "SELECT key, value FROM system_configs;"
```

### 8.5 进入容器调试

```bash
docker exec -it hl6-app sh
```

容器基于 alpine，可用 `wget` 测试内网连通性：

```bash
wget -q -O- http://localhost:8080/health
wget -q -O- http://postgres:5432  # 应返回连接错误（正常，端口非 HTTP）
```

---

## 9. 健康检查使用

### 9.1 端点

```
GET /health
```

响应：

```json
{"status":"ok"}
```

### 9.2 Docker 健康检查

```bash
docker inspect --format='{{.State.Health.Status}}' hl6-app
```

状态值：`starting` / `healthy` / `unhealthy`

查看最近 5 次检查结果：

```bash
docker inspect --format='{{range .State.Health.Log}}{{.End}}: {{.ExitCode}} {{.Output}}{{end}}' hl6-app
```

### 9.3 外部监控集成

**Uptime Kuma / Prometheus / SmokePing** 等工具配置 HTTP 探测：

- URL：`https://hl6.example.com/health`
- 预期状态码：200
- 预期响应体：包含 `"ok"`

> 健康检查端点仅验证 HTTP 服务存活，不检查数据库或 OIDC 连通性。深度监控需结合日志分析和业务接口探测。
