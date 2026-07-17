# 部署与升级

## 前置条件

- Docker Engine 24+ 和 Docker Compose v2
- PostgreSQL 16，由 `docker-compose.prod.yml` 提供，除非使用外部数据库
- Web 应用和邮件链接的公共 HTTPS 主机名
- 启用注册或迁移现有用户之前需要工作的 SMTP 账户

## 镜像

```bash
# 全球注册表
ghcr.io/hanlull/hl6:v2.0.0

# 国内代理
ghcr.milu.moe/hanlull/hl6:v2.0.0
```

将 `HL6_IMAGE` 设置为任一地址。代理仅是拉取路径替代；不得用作镜像构建源。

## 环境

从 `.env.example` 创建 `.env` 并设置生产值：

```dotenv
POSTGRES_DB=hl6
POSTGRES_USER=hl6
POSTGRES_PASSWORD=replace-with-a-random-password
APP_URL=https://domain.example.com
FRONTEND_URL=https://domain.example.com
BACKEND_URL=https://domain.example.com
ALLOWED_ORIGINS=https://domain.example.com
AUTH_PASSWORD_PEPPER_ID=v1
AUTH_PASSWORD_PEPPER=replace-with-a-long-random-secret
ENCRYPTION_KEY=optional-64-character-hex-value
SMTP_BOOTSTRAP_HOST=smtp.example.com
SMTP_BOOTSTRAP_PORT=587
SMTP_BOOTSTRAP_USERNAME=mailer@example.com
SMTP_BOOTSTRAP_PASSWORD=replace-with-the-smtp-password
SMTP_BOOTSTRAP_FROM_NAME=HL6
SMTP_BOOTSTRAP_FROM_ADDR=mailer@example.com
SMTP_BOOTSTRAP_USE_TLS=true
SMTP_BOOTSTRAP_ENABLED=true
MAINTENANCE_DATA_DIR=/var/lib/hl6/maintenance
```

`AUTH_PASSWORD_PEPPER` 在启用本地认证之前是必需的。不要将其放入仓库、发布说明、APK 输入或数据库导出中。

`MAINTENANCE_DATA_DIR` 作为 `maintenance-data` Docker 卷挂载。它临时存储服务器生成的备份存档；下载后立即将生产备份复制到独立的加密存储。

`SMTP_BOOTSTRAP_*` 值仅在相应 SMTP 值缺失时复制到数据库设置。这是任何管理员存在之前邮箱验证的安全首次启动路径。在生产中设置 `ENCRYPTION_KEY` 以便复制的密码被加密，然后在第一个管理员登录后通过管理控制台管理 SMTP。

## 启动

```bash
docker compose --env-file .env -f docker-compose.prod.yml pull
docker compose --env-file .env -f docker-compose.prod.yml up -d
docker compose --env-file .env -f docker-compose.prod.yml ps
curl -fsS https://domain.example.com/health
```

服务在启动时迁移增量模式。不会自动删除遗留身份列或启用新认证系统。

## SMTP 和首次注册

1. 在首次启动之前设置工作的 `SMTP_BOOTSTRAP_*` 值
2. 通过公共邮箱验证页面注册并设置第一个密码
3. 空安装上的第一个本地注册用户成为管理员
4. 登录后，打开 **管理后台 -> 邮件通知** 确认或更改 SMTP，然后使用 **测试发送**
5. 在 **管理后台 -> 访问与注册** 配置注册策略

## v1 到 v2 升级

硬切换仅适用于有 v1 现有用户的安装。

1. 部署带有 `AUTH_PASSWORD_PEPPER` 和工作的 `SMTP_BOOTSTRAP_*` 值的 v2 镜像。现有用户数据库在显式切换之前仍然故意禁用本地登录
2. 从容器发送已验证的 SMTP 测试：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin mail test --recipient administrator@example.com
```

3. 找到管理员 ID 并创建所需的已验证数据库存档：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec postgres \
  sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT id, email FROM users WHERE role = '\''admin'\'' ORDER BY id;"'

docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin maintenance export --created-by-user-id 1
```

记录返回的备份 `id` 并在主机外保留 ZIP。

4. 运行预检：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin auth preflight
```

5. 解决每个报告的阻碍项。不要绕过规范化的重复邮箱或缺失的备份继续
6. 运行切换：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin auth cutover --confirm --backup-id 123
```

7. 重启应用：

```bash
docker compose --env-file .env -f docker-compose.prod.yml restart app
```

现有用户现在通过其历史邮箱激活本地密码。其业务数据仍然链接到相同的用户 ID。

## 回滚

切换前的数据库存档是回滚路径。不要对部分切换的数据库启动旧版二进制文件。

1. 停止应用
2. 通过 **数据维护** 或隔离的恢复环境恢复已验证的切换前存档
3. 验证恢复的数据库并重启匹配的应用镜像

## 反向代理

将 HTTPS 代理到应用端口。保留 `Host`、`X-Forwarded-Proto`、`X-Forwarded-Host`、`Origin`、`Authorization`、`X-HL6-Client-Key` 和 `X-Idempotency-Key`。不要缓存 API 变更、邮件链接页面或服务器发送事件。

服务器接受 Capacitor 的 `https://localhost` 来源用于打包的 Android 客户端。`ALLOWED_ORIGINS` 用于额外的浏览器来源；反向代理仍必须保留上面列出的 Android 请求头。