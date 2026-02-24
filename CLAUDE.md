# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

HL6 是一个域名/子域名管理平台，用户可以在已注册域名下认领和管理子域名，并管理 DNS 记录。包含基于积分的访问控制和管理员功能。

## Tech Stack

- **Frontend**: React 19 + TypeScript, Vite, Tailwind CSS 4, React Router DOM, TanStack React Query, i18next (6 语言), OIDC 兼容认证, Shadcn UI
- **Backend**: Go (Gin + GORM), PostgreSQL 16, Cloudflare API (DNS 管理), JWT 认证 (lestrrat-go/jwx)
- **Infra**: Docker Compose (PostgreSQL), OIDC 兼容认证, Cloudflare (DNS)

## Development Commands

```bash
make dev           # 启动完整开发栈 (DB + Go server + Vite dev server)
make dev-server    # 仅启动 Go 后端 (cd server && go run ./cmd/server)
make dev-web       # 仅启动前端开发服务器 (cd web && npm run dev)
make db-up         # 启动 PostgreSQL (docker compose)
make db-down       # 停止 PostgreSQL

# 前端
cd web && npm run dev      # 开发服务器
cd web && npm run build    # 构建 (tsc + vite build)
cd web && npm run lint     # ESLint
```

环境变量从根目录 `.env` 读取（参考 `.env.example`）。Vite 通过 `envDir: ".."` 读取根目录 `.env`。

## Architecture

### Frontend (`web/src/`)

- `pages/` — 路由页面（dashboard、domains、subdomains、admin 等）
- `components/ui/` — Shadcn/Radix 基础组件
- `components/domain/`, `components/dns/`, `components/credits/` — 业务组件
- `components/layout/` — 布局组件（RootLayout, PageTransition）
- `hooks/` — 自定义 hooks（use-auth, use-subdomains, use-dns-records, use-credits 等），封装 TanStack React Query 的数据获取和 mutation
- `lib/api.ts` — REST API 客户端，使用 Bearer token 认证
- `lib/prefetch.ts` — React Query 预取逻辑
- `i18n/` — i18next 配置及语言文件（en, zh, zh-Hant, es, ru, ja）
- `types/` — TypeScript 接口定义

路径别名: `@/*` 映射到 `web/src/*`。Vite 开发服务器代理 `/api` 到 `localhost:8080`。

### Backend (`server/`)

- `cmd/server/` — 入口点（自动迁移、数据种子）
- `internal/handler/` — HTTP handlers（auth, domain, subdomain, dns, credit, admin）
- `internal/middleware/` — 中间件（auth, CORS, rate limiting, admin 权限）
- `internal/repository/` — GORM 数据访问层
- `internal/service/` — Cloudflare DNS 操作服务
- `internal/router/` — Gin 路由配置
- `internal/config/` — 环境变量配置加载
- `internal/model/` — GORM 模型（User, Domain, Subdomain, DNSRecord, CreditBalance 等）
- `pkg/response/` — 标准化 API 响应格式
- `pkg/validator/` — DNS 记录验证

### Key Patterns

- **API 响应格式**: 统一使用 `ApiResponse{code, message, data}` 包装，列表接口使用 `PaginatedResponse`
- **认证流程**: 前端 OIDC OAuth → JWT token → 后端 middleware 验证
- **数据获取**: 前端通过自定义 hooks 使用 React Query，mutation 后自动 invalidate 相关 query keys
- **Toast 通知**: 使用 Sonner，mutation 成功/失败时通过 i18n 展示国际化消息
- **DNS 记录类型**: 支持 A, AAAA, CNAME, TXT，有重复记录拦截和 CNAME 共存规则
- **角色控制**: 用户 vs 管理员，后端 admin middleware 保护管理路由

# 最高准则

## 开发要求

- 网络搜索 API 文档，而非猜测
- 除非明确提到，禁止写单元测试、跑编译
- 探索代码库，思考可否复用之前代码

## 对话要求

- 主动追问用户，确保思想对齐：目标用户、核心功能、不做的功能、未提及的可能问题