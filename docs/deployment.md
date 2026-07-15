# HL6 部署指南

本指南说明如何使用 Docker Compose 在生产环境部署 HL6。

- **GitHub 仓库**：[https://github.com/HanLuLL/hl6](https://github.com/HanLuLL/hl6)
- **原项目仓库**：[https://git.houlang.cloud/houlangcloud/hl6](https://git.houlang.cloud/houlangcloud/hl6)
- **国际镜像**：`ghcr.io/hanlull/hl6:latest`
- **中国大陆代理**：`ghcr.milu.moe/hanlull/hl6:latest`，见 [容器镜像说明](container-images.md)

---

## 目录

- [1. 前置条件](#1-前置条件)
- [2. 快速开始](#2-快速开始)
- [3. 环境变量](#3-环境变量)
- [4. 支付网关配置](#4-支付网关配置)
- [5. OIDC 配置](#5-oidc-配置)
- [6. 升级流程](#6-升级流程)
- [7. 备份与恢复](#7-备份与恢复)
- [8. 反向代理配置](#8-反向代理配置)
- [9. 健康检查](#9-健康检查)

---

## 1. 前置条件

| 依赖 | 版本要求 | 说明 |
|------|---------|------|
| Docker | ≥ 20.10 | 容器运行时 |
| Docker Compose | ≥ 2.0 | 容器编排（推荐使用 `docker compose` 子命令） |
| PostgreSQL | 16 | 镜像内置 `postgres:16-alpine`，也可使用外部数据库 |

服务器最低配置建议：

- CPU：1 核
- 内存：1 GB（仅 HL6 + PostgreSQL，不含 OIDC 提供商）
- 磁盘：10 GB（数据库 + 容器镜像）

如果启用 Redis 审计队列（多实例部署），需额外准备 Redis 6+ 实例。

---

## 2. 快速开始

### 2.1 准备 `docker-compose.prod.yml`

在部署目录创建以下文件：

```yaml
services:
  app:
    image: "${HL6_IMAGE:-ghcr.io/hanlull/hl6:latest}"
    container_name: hl6-app
    restart: unless-stopped
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "${APP_PORT:-8080}:8080"
    environment:
      SERVER_PORT: "8080"
      APP_URL: "${APP_URL:-}"
      FRONTEND_URL: "${FRONTEND_URL:-}"
      BACKEND_URL: "${BACKEND_URL:-}"
      ALLOWED_ORIGINS: "${ALLOWED_ORIGINS:-}"
      DATABASE_URL: "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"
      SESSION_SECRET: "${SESSION_SECRET:-}"
      ENCRYPTION_KEY: "${ENCRYPTION_KEY:-}"

      # OIDC（可选，留空时支持 Web UI 首配向导）
      OIDC_ISSUER: "${OIDC_ISSUER:-}"
      OIDC_CLIENT_ID: "${OIDC_CLIENT_ID:-}"
      OIDC_CLIENT_SECRET: "${OIDC_CLIENT_SECRET:-}"

      # 审计扫描（可选）
      AUDIT_SCAN_INTERVAL: "${AUDIT_SCAN_INTERVAL:-30m}"
      AUDIT_SCAN_WORKER_COUNT: "${AUDIT_SCAN_WORKER_COUNT:-2}"
      AUDIT_SCAN_TIMEOUT: "${AUDIT_SCAN_TIMEOUT:-15s}"
      REDIS_ADDR: "${REDIS_ADDR:-}"

      # 支付网关配置已移至后台「系统设置 → 支付配置」面板，无需环境变量

  postgres:
    image: "${POSTGRES_IMAGE:-postgres:16-alpine}"
    container_name: hl6-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: "${POSTGRES_DB}"
      POSTGRES_USER: "${POSTGRES_USER}"
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 5s
      timeout: 3s
      retries: 10

volumes:
  postgres-data:
```

### 2.2 准备 `.env`

在同一目录创建 `.env` 文件：

```dotenv
# 访问端口（宿主机映射）
APP_PORT=8080

# 容器镜像。国际网络使用 GHCR；中国大陆网络可使用 GHCR 代理。
HL6_IMAGE=ghcr.io/hanlull/hl6:latest
# HL6_IMAGE=ghcr.milu.moe/hanlull/hl6:latest

# PostgreSQL 默认从 Docker Hub 拉取；仅在 Docker Hub 不可达时改为已审批的镜像地址。
POSTGRES_IMAGE=postgres:16-alpine

# 公网访问地址（同域部署建议填写）
APP_URL=https://hl6.example.com
ALLOWED_ORIGINS=https://hl6.example.com

# 数据库
POSTGRES_DB=hl6
POSTGRES_USER=hl6
POSTGRES_PASSWORD=please-change-this-password

# 会话密钥（留空时首次启动自动生成并写入数据库）
SESSION_SECRET=

# 加密密钥（64 位十六进制 / 32 字节 AES-256-GCM；留空则敏感配置明文存储）
ENCRYPTION_KEY=

# OIDC（可选，留空时通过 Web UI 首配向导配置）
OIDC_ISSUER=
OIDC_CLIENT_ID=
OIDC_CLIENT_SECRET=

# 前后端地址（留空时回退到 APP_URL）
FRONTEND_URL=
BACKEND_URL=
```

生成 `ENCRYPTION_KEY`：

```bash
openssl rand -hex 32
```

> `ENCRYPTION_KEY` 建议在首次部署时确定。后续更换密钥会导致历史加密数据（如 Cloudflare Token、OIDC Client Secret）解密失败。

### 2.3 启动

```bash
docker compose -f docker-compose.prod.yml --env-file .env up -d
```

查看启动日志：

```bash
docker compose -f docker-compose.prod.yml --env-file .env logs -f app
```

启动成功后访问：

- 配置了 `APP_URL`：直接访问该地址
- 未配置 `APP_URL`：访问 `http://<服务器IP>:<APP_PORT>`

### 2.4 首次登录

- **首个注册用户自动成为管理员。**
- 若 `OIDC_*` 三项环境变量均为空且系统无用户，登录页会弹出 OIDC 首配向导。
- 若系统已有用户且 OIDC 未配置，需通过环境变量或管理员后台恢复配置。

---

## 3. 环境变量

### 3.1 必填变量

| 变量名 | 说明 | 示例 |
|--------|------|------|
| `DATABASE_URL` | PostgreSQL 连接串 | `postgres://hl6:pwd@postgres:5432/hl6?sslmode=disable` |
| `POSTGRES_DB` | 数据库名（仅 compose 内 PostgreSQL 使用） | `hl6` |
| `POSTGRES_USER` | 数据库用户（仅 compose 内 PostgreSQL 使用） | `hl6` |
| `POSTGRES_PASSWORD` | 数据库密码（仅 compose 内 PostgreSQL 使用） | 强密码 |

### 3.2 服务配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SERVER_PORT` | `8081` | 后端监听端口（容器内固定为 `8080`） |
| `APP_PORT` | `8080` | 宿主机映射端口（仅 compose 使用） |
| `APP_URL` | 空 | 同域部署的公共 URL，作为 FRONTEND/BACKEND 兜底 |
| `FRONTEND_URL` | 空 | 前端对外地址，支持逗号/换行分隔多个 |
| `BACKEND_URL` | 空 | 后端对外地址，支持逗号/换行分隔多个 |
| `ALLOWED_ORIGINS` | 空 | CORS 白名单，逗号分隔 |
| `SESSION_SECRET` | 空 | 会话密钥种子；留空时首启自动生成并写入数据库 |
| `ENCRYPTION_KEY` | 空 | 64 位十六进制 AES-256-GCM 密钥；加密数据库中的敏感字段 |

### 3.3 URL 运行时优先级

`FRONTEND_URL` / `BACKEND_URL` / `APP_URL` 的取值优先级：

1. `FRONTEND_URL` / `BACKEND_URL`（若提供）
2. `APP_URL` 兜底（若对应侧 URL 缺失）
3. 数据库配置（`frontend_urls` / `backend_urls`）
4. 从请求自动探测并落库

> 当环境变量提供了这些 URL 时，数据库 URL 配置不生效，后台也无法修改。

### 3.4 OIDC 配置

| 变量名 | 说明 |
|--------|------|
| `OIDC_ISSUER` | OIDC 提供商 Issuer URL |
| `OIDC_CLIENT_ID` | OIDC 应用 Client ID |
| `OIDC_CLIENT_SECRET` | OIDC 应用 Client Secret |

OIDC 三项字段级独立取值，优先级：**环境变量 > 数据库**。某字段在 env 中有值时，数据库值被忽略且后台不可覆盖。

### 3.5 审计扫描配置

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `AUDIT_SCAN_INTERVAL` | `30m` | 巡检调度间隔 |
| `AUDIT_SCAN_WORKER_COUNT` | `2` | 扫描 worker 数量 |
| `AUDIT_SCAN_TIMEOUT` | `15s` | 单次扫描 HTTP 抓取超时 |
| `REDIS_ADDR` | 空 | Redis 地址；留空使用进程内队列（仅适用单实例） |

### 3.6 DNS 批量操作

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `DNS_BATCH_ASYNC_THRESHOLD` | `200` | 记录数超过阈值时切换为异步批量任务 |

---

## 4. 支付网关配置

HL6 支持两种支付网关：**易支付（EPay）** 和 **码支付（CodePay）**。两者可同时启用，也可只启用其一。支付方式支持支付宝、微信、QQ。

> **配置位置**：支付网关配置已从环境变量迁移至后台「系统设置 → 支付配置」面板。管理员登录后前往 `/admin/settings` 页面填写网关地址、商户 ID、商户密钥，并按渠道（支付宝/微信/QQ）独立启用或停用。

### 4.1 易支付（EPay）

在后台支付配置面板填写：

| 字段 | 说明 |
|------|------|
| 网关地址 | 易支付网关地址，如 `https://pay.example.com` |
| 商户 ID | 易支付商户 ID |
| 商户密钥 | 易支付商户密钥（留空表示不修改当前值） |

签名算法为 MD5，签名类型 `sign_type=MD5`。

### 4.2 码支付（CodePay）

在后台支付配置面板填写：

| 字段 | 说明 |
|------|------|
| 网关地址 | 码支付网关地址 |
| 商户 ID | 码支付商户 ID |
| 商户密钥 | 码支付商户密钥（留空表示不修改当前值） |

### 4.3 渠道启用

每种网关可独立启用支付宝、微信、QQ 三个渠道。前台充值页面会根据后台启用的渠道动态展示可用支付方式。

### 4.4 回调地址

支付回调地址为后端公开端点，**无需认证**：

| 网关 | 回调地址 |
|------|---------|
| 易支付 | `https://hl6.example.com/api/v1/payment/epay/notify` |
| 码支付 | `https://hl6.example.com/api/v1/payment/codepay/notify` |

支持 GET 和 POST 两种回调方式。支付完成后用户会跳转到 `https://hl6.example.com/api/v1/payment/return`。

### 4.5 积分兑换比例

**1 CNY = 1 显示积分**（内部存储为 10 个单位，1 显示积分 = 10 内部单位）。充值金额范围 1 - 10000 CNY，订单有效期 30 分钟。

---

## 5. OIDC 配置

HL6 兼容所有标准 OIDC 提供商（Logto、Keycloak、Authentik、Authing、Google、Zitadel、Casdoor、GitLab、Auth0、Microsoft Entra ID 等）。登录前通过 `/.well-known/openid-configuration` 自动发现 endpoint。

### 5.1 环境变量配置

```env
OIDC_ISSUER=https://your-oidc-provider.example.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret
```

### 5.2 回调地址

所有提供商通用回调地址：

```
https://your-hl6-domain.com/api/v1/auth/callback
```

请求的 Scope：`openid email profile`

### 5.3 配置方式

- **环境变量**：推荐方式，启动时读取
- **数据库配置**：管理员设置页写入，适合不便改 env 的场景
- **首配向导**：OIDC 三项为空且系统无用户时，登录页弹出向导

详细提供商配置见 [OIDC 配置指南](./oidc.md)。

---

## 6. 升级流程

### 6.1 拉取最新镜像

```bash
docker compose -f docker-compose.prod.yml --env-file .env pull app
```

### 6.2 重启服务

```bash
docker compose -f docker-compose.prod.yml --env-file .env up -d app
```

### 6.3 数据库迁移

HL6 在启动时自动执行 schema 迁移（`AutoMigrate` + 手动迁移逻辑），使用 `pg_advisory_xact_lock` 防止多实例并发迁移。**无需手动执行迁移脚本**。

### 6.4 回滚

如需回滚到旧版本，指定具体镜像 tag：

```bash
# 在 .env 中选择需要回滚到的镜像版本
HL6_IMAGE=ghcr.io/hanlull/hl6:v1.2.0
```

然后：

```bash
docker compose -f docker-compose.prod.yml --env-file .env up -d app
```

> **注意**：数据库 schema 迁移通常不可逆。回滚前建议先备份数据库。

---

## 7. 备份与恢复

### 7.1 数据库备份

使用 `pg_dump` 在运行中的 PostgreSQL 容器内执行：

```bash
docker exec hl6-postgres pg_dump -U ${POSTGRES_USER} -d ${POSTGRES_DB} -F c -f /tmp/hl6-backup.dump
docker cp hl6-postgres:/tmp/hl6-backup.dump ./hl6-backup-$(date +%Y%m%d).dump
docker exec hl6-postgres rm /tmp/hl6-backup.dump
```

### 7.2 数据库恢复

```bash
docker cp ./hl6-backup-20260101.dump hl6-postgres:/tmp/hl6-backup.dump
docker exec hl6-postgres pg_restore -U ${POSTGRES_USER} -d ${POSTGRES_DB} -c /tmp/hl6-backup.dump
docker exec hl6-postgres rm /tmp/hl6-backup.dump
```

### 7.3 自动备份脚本示例

```bash
#!/bin/bash
set -e
BACKUP_DIR="/backups/hl6"
RETENTION_DAYS=7
mkdir -p "$BACKUP_DIR"

docker exec hl6-postgres pg_dump -U hl6 -d hl6 -F c -f /tmp/hl6-backup.dump
docker cp hl6-postgres:/tmp/hl6-backup.dump "$BACKUP_DIR/hl6-$(date +%Y%m%d-%H%M%S).dump"
docker exec hl6-postgres rm /tmp/hl6-backup.dump

find "$BACKUP_DIR" -name "*.dump" -mtime +$RETENTION_DAYS -delete
```

配合 cron 定时执行：

```cron
0 3 * * * /opt/hl6/backup.sh
```

### 7.4 停止与清理

停止服务：

```bash
docker compose -f docker-compose.prod.yml --env-file .env down
```

停止并删除数据卷（**危险操作，会清除所有数据**）：

```bash
docker compose -f docker-compose.prod.yml --env-file .env down -v
```

---

## 8. 反向代理配置

生产环境建议在 HL6 前面部署 Nginx 反向代理，处理 TLS 终止和域名路由。

### 8.1 Nginx 配置示例

```nginx
server {
    listen 80;
    server_name hl6.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name hl6.example.com;

    ssl_certificate     /etc/letsencrypt/live/hl6.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/hl6.example.com/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    # 前端静态资源和 SPA 路由
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # API 接口
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SSE 长连接（通知推送）
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

    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:8080;
        access_log off;
    }
}
```

### 8.2 关键配置说明

- **SSE 端点**必须关闭 `proxy_buffering`，否则通知推送会被缓冲导致延迟
- `proxy_read_timeout` 对 SSE 端点需设置较长（如 86400s），避免空闲断开
- TLS 证书推荐使用 [Let's Encrypt](https://letsencrypt.org/) + certbot 自动续期

---

## 9. 健康检查

### 9.1 健康检查端点

```
GET /health
```

返回：

```json
{"status":"ok"}
```

HTTP 状态码 `200` 表示服务正常。

### 9.2 Docker 健康检查

容器镜像内置 HEALTHCHECK：

```dockerfile
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O- http://localhost:8080/health || exit 1
```

查看健康状态：

```bash
docker inspect --format='{{.State.Health.Status}}' hl6-app
```

### 9.3 外部监控

可使用 curl 或监控工具定期探测：

```bash
curl -fsS https://hl6.example.com/health || echo "UNHEALTHY"
```

> 健康检查端点仅验证 HTTP 服务存活，不检查数据库连通性。若需深度健康检查，可通过 `docker logs hl6-app` 观察启动日志确认数据库连接状态。
