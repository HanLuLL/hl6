# HL6

HL6 是一个域名/子域名管理平台。用户可以在已注册域名下认领并管理子域名、维护 DNS 记录，并基于积分规则控制访问；管理员可进行全局管理。

## 核心能力

- 域名与子域名管理
- DNS 记录管理（A / AAAA / CNAME / TXT）
- 基于积分的访问控制
- 用户与管理员角色权限
- 多语言前端（i18next）

## 技术栈

- 前端：React 19 + TypeScript + Vite + Tailwind CSS 4 + TanStack React Query
- 后端：Go（Gin + GORM）+ PostgreSQL 16
- 认证：OIDC 兼容（支持 Logto、Keycloak、Authentik、Google、Zitadel 等）+ JWT
- 基础设施：Docker Compose（本地 PostgreSQL）

## 云应用一键部署
[![通过雨云一键部署](https://rainyun-apps.cn-nb1.rains3.com/materials/deploy-on-rainyun-cn.svg)](https://app.rainyun.com/apps/rca/store/7627?ref=ins_)

## 快速开始（`make dev`）

### 1. 准备依赖

- Docker（需支持 `docker compose`）
- Go（`server/go.mod` 当前为 `go 1.25.5`）
- Node.js（建议 `^20.19.0` 或 `>=22.12.0`，与当前 Vite 版本要求一致）
- npm

### 2. 初始化环境变量

在项目根目录执行：

```bash
cp .env.example .env
```

按需修改 `.env`，至少确认以下字段：

- `DATABASE_URL`
- `SERVER_PORT`
- `APP_URL`（可选；同域部署时可作为前后端公共访问地址）
- `FRONTEND_URL` / `BACKEND_URL`（可选；配置后优先级高于数据库配置）
- `ALLOWED_ORIGINS`
- `OIDC_ISSUER` / `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET`（支持任何标准 OIDC 提供商，详见 [OIDC 配置指南](docs/oidc.md)）

### 3. 安装前端依赖

```bash
cd web && npm install
```

### 4. 一键启动开发环境

回到项目根目录执行：

```bash
make dev
```

`make dev` 会自动完成以下动作：

1. `docker compose up -d` 启动 PostgreSQL
2. 启动 Go 后端（`server/cmd/server`）
3. 启动 Vite 前端开发服务器（`web`）

启动后默认访问地址：

- 前端：<http://localhost:5173>
- 后端 API：<http://localhost:8080>
- PostgreSQL：`localhost:5432`（DB: `hl6`，User: `hl6`，Password: `hl6dev`）

停止开发环境：

- 在 `make dev` 终端按 `Ctrl+C`（会结束前后端进程）
- 如需关闭数据库容器：`make db-down`

## 常用命令

```bash
# 一键开发（推荐）
make dev

# 仅启动数据库
make db-up
make db-down

# 仅启动后端
make dev-server

# 仅启动前端
make dev-web
```

## 环境变量说明

根目录 `.env.example` 提供了默认模板：

| 变量名 | 说明 |
| --- | --- |
| `DATABASE_URL` | PostgreSQL 连接串 |
| `SERVER_PORT` | 后端服务端口（默认 `8080`） |
| `APP_URL` | 可选公共地址；当 `FRONTEND_URL` 或 `BACKEND_URL` 未设置时，作为对应侧的兜底值 |
| `OIDC_ISSUER` | OIDC 提供商 Issuer URL（如 Logto、Keycloak、Google 等） |
| `OIDC_CLIENT_ID` | OIDC 应用 Client ID |
| `OIDC_CLIENT_SECRET` | OIDC 应用 Client Secret |
| `SESSION_SECRET` | 会话密钥首启种子（可留空；仅在数据库未初始化密钥时使用） |
| `FRONTEND_URL` | 可选；前端地址，设置后优先使用环境变量值（支持多地址，逗号/换行分隔） |
| `BACKEND_URL` | 可选；后端对外地址，设置后优先使用环境变量值（支持多地址，逗号/换行分隔） |
| `ALLOWED_ORIGINS` | CORS 白名单（逗号分隔） |
| `ENCRYPTION_KEY` | 可选；64 位十六进制字符串（32 字节），有值则加密 Cloudflare Token，无值则明文存储 |

> 注意：Vite 配置使用 `envDir: ".."`，前端会读取项目根目录 `.env`。
> 注意：系统中首个注册用户会自动成为管理员，后续注册用户默认是普通用户。
> 注意：运行时 URL 优先级为 `FRONTEND_URL/BACKEND_URL`（含 `APP_URL` 兜底）> 数据库配置 > 自动探测并落库。管理员首次登录或地址变化时需要在面板确认一次当前生效地址。
> 注意：服务会在数据库中维护内部会话密钥（`_internal_session_secret`）。首次启动如果该键不存在，会使用 `SESSION_SECRET` 作为一次性种子；若 `SESSION_SECRET` 为空则自动生成随机密钥。后续启动以数据库值为准。
> 注意：如果数据库数据丢失，内部会话密钥会重建，所有已登录会话将失效并需要重新登录。

## 容器部署

仓库根目录新增了一个全栈 `Dockerfile`，会构建前端并把静态资源打进最终镜像，由 Go 服务同端口托管前端页面和 `/api/v1` 接口。

- 镜像启动后默认监听 `8080`
- 同域部署时，只需设置 `APP_URL`，并让 `ALLOWED_ORIGINS` 包含该地址
- 仍然需要独立的 PostgreSQL 实例，`DATABASE_URL` 必填

雨云商店上架建议见 [docs/rainyun.md](docs/rainyun.md)。

## 项目结构

```text
.
├── server/                 # Go 后端
│   ├── cmd/server/         # 程序入口
│   ├── internal/           # handler/service/repository/router 等
│   └── pkg/                # 公共包（response/validator）
├── web/                    # React 前端
│   └── src/                # pages/components/hooks/lib/i18n/types
├── docker-compose.yml      # 本地 PostgreSQL
├── Makefile                # 开发命令入口
└── .env.example            # 环境变量模板
```

## 许可证

本项目采用 GNU Affero General Public License v3.0（AGPL-3.0）开源协议，详见根目录 `LICENSE` 文件。
