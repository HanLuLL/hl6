# 生产部署与升级

本文面向生产环境维护者，覆盖 Docker Compose 部署、镜像选择、环境变量、反向代理、升级、回滚和备份。日常监控与故障诊断见[运维与故障排查](operations.md)。

## 1. 部署拓扑

官方生产 Compose 启动两个服务：

- `app`：包含 Go API 与构建后的 React 前端，容器内监听 `8080`。
- `postgres`：PostgreSQL 16，数据保存在命名卷 `postgres-data`。

单实例部署不要求 Redis；未设置 `REDIS_ADDR` 时，审核任务使用进程内队列。多实例部署应配置共享 Redis，并确保所有实例使用相同数据库、会话密钥和加密密钥。

## 2. 前置条件

- Linux x86_64/arm64 主机，建议至少 2 vCPU、2 GB 内存和 10 GB 可用磁盘。
- Docker Engine 24 或更高版本。
- Docker Compose v2。
- 指向服务器的域名和有效 HTTPS 证书。
- 一个兼容 OIDC 的身份提供商。
- 至少一个受支持的 DNS 提供商账号及最小权限 API 凭据。

检查版本：

```bash
docker version
docker compose version
```

## 3. 镜像选择

| 网络环境 | `HL6_IMAGE` | 说明 |
| --- | --- | --- |
| 国际网络 | `ghcr.io/hanlull/hl6:latest` | 官方 GHCR 镜像 |
| 中国大陆 | `ghcr.milu.moe/hanlull/hl6:latest` | GHCR 拉取代理，适合无法直连 GHCR 的主机 |
| 固定正式版本 | 将 `latest` 改为 `v1.0.0` | 便于审计、回滚和多节点一致部署 |

PostgreSQL 默认使用 `postgres:16-alpine`。只有 Docker Hub 确实不可达时，才将 `POSTGRES_IMAGE` 改为组织批准并验证过的镜像地址。

## 4. 准备配置

```bash
git clone https://github.com/HanLuLL/hl6.git
cd hl6
cp .env.example .env
chmod 600 .env
```

生产环境至少设置：

```dotenv
APP_PORT=8080
HL6_IMAGE=ghcr.io/hanlull/hl6:v1.0.0
POSTGRES_IMAGE=postgres:16-alpine

POSTGRES_DB=hl6
POSTGRES_USER=hl6
POSTGRES_PASSWORD=替换为高强度随机密码

APP_URL=https://domain.example.com
FRONTEND_URL=https://domain.example.com
BACKEND_URL=https://domain.example.com
ALLOWED_ORIGINS=https://domain.example.com

SESSION_SECRET=替换为至少32字节随机值
ENCRYPTION_KEY=替换为64位十六进制字符串
```

生成随机值示例：

```bash
openssl rand -base64 48
openssl rand -hex 32
```

第一条可用于 `SESSION_SECRET`，第二条用于 `ENCRYPTION_KEY`。密钥一旦投入生产，必须进入密码管理器和备份制度；随意更换会导致已有会话失效，错误更换加密密钥还会使数据库中的敏感配置无法解密。

### URL 优先级

每个 URL 字段按以下顺序独立解析：

1. `FRONTEND_URL` 或 `BACKEND_URL`。
2. 对应字段为空时使用 `APP_URL`。
3. 环境变量为空时使用数据库中的动态配置。
4. 仍为空时根据可信请求自动检测并写入数据库。

生产环境建议显式设置三个 URL。跨域部署时，`ALLOWED_ORIGINS` 必须列出网页端来源；多个来源以逗号分隔。Android Capacitor 固定来源 `https://localhost` 由服务端原生客户端 CORS 规则处理。

### OIDC 配置方式

二选一：

- 环境变量锁定：设置 `OIDC_ISSUER`、`OIDC_CLIENT_ID`、`OIDC_CLIENT_SECRET`。
- 首配向导：三项全部留空，在系统没有任何用户时通过网页向导写入数据库。

同一字段的环境变量优先于数据库。不要只清空其中一项后期待后台配置完全接管。详细回调和提供商说明见[OIDC 认证配置](oidc.md)。

### 可选 Redis

```dotenv
REDIS_ADDR=redis.example.internal:6379
AUDIT_SCAN_INTERVAL=30m
AUDIT_SCAN_WORKER_COUNT=2
AUDIT_SCAN_TIMEOUT=15s
```

生产多实例建议使用 Redis Streams。当前 Compose 未内置 Redis，需要使用独立 Redis 服务或在组织维护的 Compose 扩展中增加。

## 5. 启动与首配

```bash
docker compose --env-file .env -f docker-compose.prod.yml pull
docker compose --env-file .env -f docker-compose.prod.yml up -d
docker compose --env-file .env -f docker-compose.prod.yml ps
```

确认健康端点：

```bash
curl --fail http://127.0.0.1:8080/health
```

正常响应示例：

```json
{"status":"ok","database":true}
```

完成 HTTPS 反向代理后访问站点。若使用首配向导，先保存 OIDC 配置，再发起登录。首个注册成功的用户自动成为管理员；完成后立即检查用户组、DNS 凭据和公开域名配置。

## 6. Nginx 反向代理

```nginx
server {
    listen 443 ssl http2;
    server_name domain.example.com;

    ssl_certificate     /etc/letsencrypt/live/domain.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/domain.example.com/privkey.pem;

    client_max_body_size 10m;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/v1/notifications/sse {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 3600s;
    }
}
```

HTTP 站点应永久重定向到 HTTPS。若使用 CDN，还需确认 CDN 不缓存 OIDC 回调、API 写请求或 SSE，并原样传递 `Origin`、`Authorization` 和自定义客户端请求头。

## 7. 升级

升级前阅读 Release Notes，备份数据库和 `.env`，然后固定目标版本：

```dotenv
HL6_IMAGE=ghcr.io/hanlull/hl6:v1.0.0
```

执行：

```bash
docker compose --env-file .env -f docker-compose.prod.yml pull app
docker compose --env-file .env -f docker-compose.prod.yml up -d app
docker compose --env-file .env -f docker-compose.prod.yml ps
curl --fail https://domain.example.com/health
```

服务启动时自动执行 GORM 数据库迁移。升级后检查登录、DNS 查询、通知 SSE 和后台统计。Android 版本策略独立于服务端镜像，由管理后台客户端配置控制。

## 8. 回滚

1. 确认目标旧版本与当前数据库结构兼容。
2. 在 `.env` 中将 `HL6_IMAGE` 改为已验证的旧标签。
3. 拉取并重建 `app` 容器。
4. 验证健康、登录和关键写操作。

```bash
docker compose --env-file .env -f docker-compose.prod.yml pull app
docker compose --env-file .env -f docker-compose.prod.yml up -d app
```

数据库迁移通常只向前兼容。若版本说明要求数据库回滚，应先停止写流量，并使用升级前备份恢复，而不是手工删除列或表。

## 9. 备份与恢复

创建备份目录并限制权限：

```bash
install -d -m 700 backups
docker compose --env-file .env -f docker-compose.prod.yml exec -T postgres \
  sh -c 'pg_dump -Fc -U "$POSTGRES_USER" "$POSTGRES_DB"' \
  > "backups/hl6-$(date +%Y%m%d-%H%M%S).dump"
```

恢复前停止应用写入并确认目标数据库：

```bash
docker compose --env-file .env -f docker-compose.prod.yml stop app
docker compose --env-file .env -f docker-compose.prod.yml exec -T postgres \
  sh -c 'pg_restore --clean --if-exists -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < backups/hl6-YYYYMMDD-HHMMSS.dump
docker compose --env-file .env -f docker-compose.prod.yml start app
```

同时备份 `.env`、反向代理配置和外部 Redis 配置，但应加密保存。定期在隔离环境执行恢复演练；未验证可恢复的备份不算有效备份。

## 10. 上线检查清单

- 镜像使用明确版本标签，数据库密码和密钥未使用示例值。
- `ENCRYPTION_KEY`、OIDC Secret 和 DNS 凭据已进入安全备份。
- HTTPS、OIDC 回调、外部 URL 和 CORS 来源完全一致。
- `/health` 返回数据库正常，容器没有重启循环。
- 首个管理员已确认，默认域名、保留前缀和用户组符合预期。
- DNS 提供商使用最小权限令牌并完成一次只读/写入验证。
- 支付回调、邮件、通知 SSE 和审核队列按启用功能完成验证。
- 数据库备份和恢复演练已有记录。
