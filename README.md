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
- 认证：Logto（OIDC）+ JWT
- 基础设施：Docker Compose（本地 PostgreSQL）

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
- `FRONTEND_URL`
- `ALLOWED_ORIGINS`
- `LOGTO_ENDPOINT` / `LOGTO_APP_ID` / `LOGTO_APP_SECRET`（需要您自部署 logto）

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
| `ADMIN_EMAILS` | 管理员邮箱列表（逗号分隔） |
| `LOGTO_ENDPOINT` | Logto OIDC 地址 |
| `LOGTO_APP_ID` | Logto 应用 ID |
| `LOGTO_APP_SECRET` | Logto 应用密钥 |
| `SESSION_SECRET` | 会话签名密钥 |
| `FRONTEND_URL` | 前端地址（默认 `http://localhost:5173`） |
| `ALLOWED_ORIGINS` | CORS 白名单（逗号分隔） |
| `ENCRYPTION_KEY` | 64 位十六进制字符串（32 字节） |

> 注意：Vite 配置使用 `envDir: ".."`，前端会读取项目根目录 `.env`。

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

## 常见问题

### 1) `make dev` 启动失败，提示 `npm` 相关错误

先执行：

```bash
cd web && npm install
```

### 2) 端口冲突（`5173` / `8080` / `5432`）

- 修改 `.env` 中 `SERVER_PORT`
- 调整前端端口（Vite 启动参数或配置）
- 修改 `docker-compose.yml` 的数据库端口映射

### 3) 登录相关接口不可用

请确认 `.env` 中 Logto 配置已填写正确：

- `LOGTO_ENDPOINT`
- `LOGTO_APP_ID`
- `LOGTO_APP_SECRET`

