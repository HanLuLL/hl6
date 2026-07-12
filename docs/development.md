# HL6 开发环境搭建

本指南说明如何从源码搭建 HL6 的本地开发环境。

- **GitHub 仓库**：[https://github.com/HanLuLL/hl6](https://github.com/HanLuLL/hl6)
- **原项目仓库**：[https://git.houlang.cloud/houlangcloud/hl6](https://git.houlang.cloud/houlangcloud/hl6)

---

## 目录

- [1. 前置条件](#1-前置条件)
- [2. 克隆与环境配置](#2-克隆与环境配置)
- [3. 启动开发](#3-启动开发)
- [4. 项目结构](#4-项目结构)
- [5. 代码规范与约定](#5-代码规范与约定)
- [6. 生产构建](#6-生产构建)

---

## 1. 前置条件

| 依赖 | 版本要求 | 说明 |
|------|---------|------|
| Go | ≥ 1.25 | 后端语言 |
| Node.js | ≥ 22（推荐 LTS） | 前端构建 |
| Docker & Docker Compose | 任意现代版本 | 运行 PostgreSQL 16 |
| Make | 任意版本 | 任务编排 |
| Git | 任意版本 | 版本控制 |

### 中国大陆网络优化

```bash
# Go 模块代理
go env -w GOPROXY=https://goproxy.cn,direct

# npm 镜像
npm config set registry https://registry.npmmirror.com
```

---

## 2. 克隆与环境配置

### 2.1 克隆仓库

```bash
git clone https://github.com/HanLuLL/hl6.git
cd hl6
```

### 2.2 配置环境变量

```bash
cp .env.example .env
```

编辑 `.env`，关键配置项：

```env
# 数据库（Docker 默认配置，通常无需修改）
DATABASE_URL=postgres://hl6:hl6dev@localhost:5433/hl6?sslmode=disable
SERVER_PORT=8081

# OIDC 认证（必填，或留空使用 Web UI 首配向导）
OIDC_ISSUER=https://your-oidc-provider.example.com
OIDC_CLIENT_ID=your-client-id
OIDC_CLIENT_SECRET=your-client-secret

# 前端地址
FRONTEND_URL=http://localhost:5174
ALLOWED_ORIGINS=http://localhost:5174

# 会话密钥（留空自动生成）
SESSION_SECRET=

# 加密密钥（可选，用于加密敏感配置）
ENCRYPTION_KEY=
```

OIDC 配置指南见 [OIDC 提供商配置](./oidc.md)。

生成 `ENCRYPTION_KEY`（可选）：

```bash
openssl rand -hex 32
```

### 2.3 安装前端依赖

```bash
cd web && npm install && cd ..
```

---

## 3. 启动开发

### 3.1 一键启动

```bash
make dev
```

该命令并行执行：

1. `docker compose up -d --wait` — 启动 PostgreSQL 容器
2. `go run ./cmd/server` — 编译并启动 Go 后端（端口 `SERVER_PORT`，默认 8081）
3. `npm run dev` — 启动 Vite 前端开发服务器（端口 `VITE_DEV_PORT`，默认 5174）

启动成功标志：

```
✔ Container hl6-postgres  Started
Database migrated successfully
Server starting on :8081
VITE vX.x.x  ready
  ➜  Local:   http://localhost:5174/
```

访问 `http://localhost:5174`。

### 3.2 分离启动

如需单独启动各组件：

```bash
make db-up         # 仅启动 PostgreSQL
make dev-server    # 仅启动 Go 后端
make dev-web       # 仅启动前端
```

### 3.3 停止服务

在 `make dev` 终端按 `Ctrl+C` 停止前后端。

停止数据库：

```bash
make db-down
```

> `make db-down` 不会删除数据。彻底清除数据：`docker compose down -v`。

### 3.4 Vite 代理

Vite 开发服务器将 `/api` 请求代理到 `localhost:8081`（由 `SERVER_PORT` 控制）。跨域部署时可在 `.env` 中设置 `VITE_API_BASE_URL`。

---

## 4. 项目结构

```
hl6/
├── .env.example              # 环境变量模板
├── .env                      # 本地配置（不提交 Git）
├── Makefile                  # 开发命令
├── docker-compose.yml        # PostgreSQL 容器定义
├── Dockerfile                # 多阶段构建（前端 + 后端）
├── server/                   # Go 后端
│   ├── cmd/server/main.go    # 入口点
│   ├── internal/
│   │   ├── config/           # 环境变量加载
│   │   ├── handler/          # HTTP 处理器
│   │   ├── middleware/       # 认证、CORS、管理员权限
│   │   ├── model/            # GORM 数据模型
│   │   ├── repository/       # 数据访问层
│   │   ├── router/           # 路由配置
│   │   ├── service/          # 业务逻辑（DNS 提供商、审计、支付等）
│   │   ├── worker/           # 审计扫描 worker
│   │   └── oidc/             # OIDC Discovery
│   └── pkg/
│       ├── audit/            # 审计抓取器
│       ├── crypto/           # AES-256-GCM 加解密
│       ├── queue/            # 任务队列
│       ├── response/         # 统一 API 响应
│       └── validator/        # DNS 记录验证
└── web/                      # React 前端
    ├── src/
    │   ├── pages/            # 路由页面
    │   ├── components/       # UI 与业务组件
    │   ├── hooks/            # 自定义 hooks（React Query 封装）
    │   ├── lib/              # API 客户端、工具函数
    │   ├── i18n/             # 国际化（6 种语言）
    │   └── types/            # TypeScript 类型
    └── vite.config.ts        # Vite 配置（envDir 读取根 .env）
```

详细架构设计见 [架构文档](./architecture.md)。

---

## 5. 代码规范与约定

### 5.1 后端（Go）

- **框架**：Gin (HTTP) + GORM (ORM)
- **分层**：`handler` → `service` → `repository` → `model`
- **响应格式**：统一使用 `pkg/response`，返回 `ApiResponse{code, message, data}`
- **错误处理**：使用 i18n key（`message_key`），前端匹配翻译
- **认证**：JWT (HS256) 存储在 `hl6_session` Cookie
- **配置**：环境变量优先，部分支持数据库配置（OIDC、URL）
- **日志**：`log` 用于启动信息，`log/slog` 用于业务日志

### 5.2 前端（TypeScript / React）

- **状态管理**：TanStack React Query，封装在 `hooks/` 中
- **UI 组件**：Shadcn UI + Radix Primitives + Tailwind CSS 4
- **国际化**：i18next，语言文件在 `i18n/`
- **API 客户端**：`lib/api.ts`，Bearer token 认证
- **通知**：Sonner toast，mutation 成功/失败时展示国际化消息
- **路径别名**：`@/*` 映射到 `web/src/*`

### 5.3 通用约定

- **API 响应**：`code: 0` 成功，`-1` 失败
- **DNS 记录类型**：A / AAAA / CNAME / TXT
- **积分单位**：内部 int64，1 显示积分 = 10 内部单位
- **首个注册用户自动成为管理员**

### 5.4 Lint

```bash
cd web && npm run lint   # ESLint
```

---

## 6. 生产构建

### 6.1 Docker 镜像构建

```bash
docker build -t hl6:local .
```

Dockerfile 采用三阶段构建：

1. `node:22-alpine` — 构建前端 (`npm run build`)
2. `golang:1.25-alpine` — 构建后端二进制（CGO 启用）
3. `alpine:3.22` — 最终镜像，内嵌前端 dist + 后端二进制

构建参数：

```bash
docker build \
  --build-arg APP_GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD) \
  --build-arg APP_GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg GOPROXY=https://goproxy.cn,direct \
  -t hl6:local .
```

### 6.2 前端单独构建

```bash
cd web && npm run build
# 产物在 web/dist/
```

### 6.3 后端单独构建

```bash
cd server
CGO_ENABLED=1 go build -o hl6-server ./cmd/server
```

> 后端依赖 CGO（PostgreSQL 驱动需要），构建时需确保 C 编译器可用。

### 6.4 生产部署

生产环境部署使用预构建镜像 `ghcr.milu.moe/hanlull/hl6:latest`，详细流程见 [部署指南](./deployment.md)。
