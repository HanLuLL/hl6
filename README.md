# HL6

HL6 是一个域名、子域名和 DNS 管理平台。提供用户子域名认领、DNS 记录管理、积分系统、内容治理、管理后台，以及基于本地 React UI 构建的 Android 客户端。

## v2.0.0

- 自主邮箱认证：支持注册验证、密码设置、旧账户激活、密码重置、Argon2id 哈希、会话失效、精确邮箱域名白名单/黑名单规则
- 现有 v1 用户保留相同的用户 ID、资料、头像、积分、域名、DNS 记录、封禁状态、通知和其他关联数据，只需通过现有邮箱一次性激活新密码
- 用户资料由 HL6 管理：自定义姓名或头像在登录时不会被覆盖。数字 `@qq.com` 邮箱在没有自定义头像时使用 HTTPS QQ 头像回退
- 封禁邮件和封禁界面显示封禁开始时间和预计解封时间
- Android 客户端打包本地构建的 Web UI，直接调用 HL6 API，使用 Android Keystore 存储会话，请求时携带服务端通讯密钥
- 管理员可控制 Android 版本、强制更新策略、更新公告、更新链接、通讯密钥生命周期，以及受保护的 PostgreSQL ZIP 导出/恢复

## 镜像

```bash
# 全球 GHCR 镜像
docker pull ghcr.io/hanlull/hl6:v2.0.0

# 国内代理
docker pull ghcr.milu.moe/hanlull/hl6:v2.0.0
```

使用 `latest` 获取最新稳定镜像。代理地址镜像内容相同，适用于无法直接访问 GHCR 的环境。

## 快速开始

```bash
git clone https://github.com/HanLuLL/hl6.git
cd hl6
cp .env.example .env
```

在 `.env` 中至少设置以下生产环境配置：

```dotenv
POSTGRES_DB=hl6
POSTGRES_USER=hl6
POSTGRES_PASSWORD=替换为强随机密码
APP_URL=https://domain.example.com
AUTH_PASSWORD_PEPPER_ID=v1
AUTH_PASSWORD_PEPPER=替换为长随机密钥
SMTP_BOOTSTRAP_HOST=smtp.example.com
SMTP_BOOTSTRAP_USERNAME=mailer@example.com
SMTP_BOOTSTRAP_PASSWORD=替换为SMTP密码
SMTP_BOOTSTRAP_FROM_ADDR=mailer@example.com
SMTP_BOOTSTRAP_ENABLED=true
```

启动生产环境：

```bash
docker compose --env-file .env -f docker-compose.prod.yml up -d
docker compose --env-file .env -f docker-compose.prod.yml ps
```

对于新安装，SMTP 启动配置仅用于补全缺失的服务器设置，以便在管理员存在之前完成首次邮箱/密码注册。之后请在管理后台管理 SMTP。在迁移现有 v1 安装之前，请遵循 [部署文档](docs/deployment.md) 中的切换流程。

## 文档

| 主题 | 文档 |
| --- | --- |
| 文档索引 | [docs/README.md](docs/README.md) |
| 邮箱认证和 v1 迁移 | [docs/authentication.md](docs/authentication.md) |
| 部署、升级、备份和回滚 | [docs/deployment.md](docs/deployment.md) |
| 管理后台 | [docs/administration.md](docs/administration.md) |
| Android 客户端和 GitHub Actions 构建 | [docs/android-client.md](docs/android-client.md) |
| API 集成 | [docs/api.md](docs/api.md) |
| 架构 | [docs/architecture.md](docs/architecture.md) |
| 运维和恢复 | [docs/operations.md](docs/operations.md) |
| 开发 | [docs/development.md](docs/development.md) |
| Android 兼容性约定 | [docs/agent.md](docs/agent.md) |

## 开发

```bash
make dev
make dev-server
make dev-web
make db-up
make db-down
```

影响客户端的更改必须在同一变更集中遵循 [docs/agent.md](docs/agent.md)。