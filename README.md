# HL6

HL6 是一个域名、子域名和 DNS 管理平台。提供用户子域名认领、DNS 记录管理、积分系统、内容治理、管理后台，以及基于本地 React UI 构建的 Android 客户端。

## 镜像

```bash
# 全球 GHCR 镜像
docker pull ghcr.io/hanlull/hl6:v2.0.0

# 国内镜像（需先注册并登录）
docker login mirror.houlang.cloud
docker pull mirror.houlang.cloud/hanlull/hl6:v2.0.0
```

使用 `latest` 获取最新稳定镜像。国内镜像需要先在 [mirror.houlang.cloud](https://mirror.houlang.cloud) 注册账号，然后通过 `docker login` 登录后才能拉取，适用于无法直接访问 GHCR 的环境。

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