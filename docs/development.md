# 开发指南

本文面向 HL6 贡献者，说明本地环境、目录边界、常用命令和提交前检查。生产配置见[生产部署与升级](deployment.md)，Android 特殊要求见[客户端适配规范](agent.md)。

## 1. 技术要求

| 工具 | 建议版本 | 用途 |
| --- | --- | --- |
| Go | 与 `server/go.mod` 兼容的稳定版本 | 后端 API、Worker、迁移 |
| Node.js | 22 | React/Vite 与构建脚本 |
| pnpm | 11.7.0 | 前端依赖 |
| PostgreSQL | 16 | 主数据库 |
| Docker / Compose | Engine 24+ / Compose v2 | 开发数据库和生产构建 |
| Redis | 可选 | 多实例审核队列 |
| JDK / Android SDK | JDK 21 / SDK 36 | 仅 Android 构建 |

中国大陆开发网络可配置组织批准的 Go、npm 和 Docker 镜像源，但不要把个人代理地址或凭据提交到仓库。

## 2. 初始化

```bash
git clone https://github.com/HanLuLL/hl6.git
cd hl6
cp .env.example .env
```

开发默认数据库：

```dotenv
DATABASE_URL=postgres://hl6:hl6dev@localhost:5433/hl6?sslmode=disable
SERVER_PORT=8081
VITE_DEV_PORT=5174
BACKEND_URL=http://localhost:8081
FRONTEND_URL=http://localhost:5174
ALLOWED_ORIGINS=http://localhost:5174
```

OIDC 三项可以指向测试租户，也可以全部留空后使用首配向导。不要使用生产 Client Secret、DNS Token、支付密钥或 Android 通讯密钥进行普通本地开发。

## 3. 常用命令

```bash
make dev           # 启动开发数据库、Go 服务端和 Vite
make dev-server    # cd server && go run ./cmd/server
make dev-web       # cd web && pnpm run dev
make db-up         # 启动开发 PostgreSQL
make db-down       # 停止开发 PostgreSQL
```

前端：

```bash
cd web
pnpm install --frozen-lockfile
pnpm run dev
pnpm run build
pnpm run lint
```

后端：

```bash
cd server
go run ./cmd/server
go test ./...
```

Vite 从仓库根目录读取 `.env`，并将 `/api` 代理到 Go 服务端。跨域调试时显式设置 `VITE_API_BASE_URL`，同时更新服务端 CORS。

## 4. 目录边界

```text
server/
  cmd/server/             启动、迁移、种子
  internal/config/        环境变量与运行时配置
  internal/handler/       HTTP 输入输出
  internal/middleware/    认证、CORS、管理员权限
  internal/model/         GORM 数据模型
  internal/repository/    数据访问
  internal/router/        路由注册与依赖组装
  internal/service/       DNS、审核、迁移等业务编排
  internal/worker/        审核调度与消费
  pkg/                    响应、验证、加密、队列基础能力

web/
  src/pages/              路由页面
  src/components/ui/      Shadcn/Radix 基础组件
  src/components/         领域组件和布局
  src/hooks/              React Query 查询与 mutation
  src/lib/api.ts          统一 REST 客户端
  src/i18n/               六种语言资源
  src/types/              API/页面类型
  android/                Capacitor Android 工程
  scripts/                客户端构建配置脚本

docs/                     长期维护文档
.github/workflows/        镜像、客户端和正式 Release 流程
```

Handler 不直接承载复杂 DNS 或审核流程；业务编排放入 Service，持久化放入 Repository。前端页面通过 Hook 和统一 API 客户端访问后端，不在组件内散落 Fetch 和鉴权逻辑。

## 5. 后端约定

- 使用 Gin 路由组明确公开、登录和管理员边界。
- 使用统一 `response.Response`、分页响应和 `message_key`。
- 结构化 JSON 使用绑定和模型，不使用字符串拼接解析。
- 管理员判定同时支持用户角色和管理员用户组。
- DNS 写操作进入幂等服务和提供商抽象，不绕过审计。
- 外部 URL、凭据和回调必须经过验证；敏感字段使用 AES-256-GCM。
- 数据库迁移保持向前兼容，破坏性迁移必须提供明确升级/回滚说明。
- 日志不输出凭据、会话、支付签名或通讯密钥。

新增路由时同步更新[API 集成指南](api.md)。修改数据流或组件边界时同步更新[系统架构](architecture.md)。

## 6. 前端约定

- React Query 负责服务端状态、缓存、预取和 mutation 失效。
- `web/src/lib/api.ts` 负责 API 基址、Bearer 会话、原生通讯头、超时和错误转换。
- 操作成功/失败通过 Sonner 和 i18n Key 展示。
- 页面和组件不复制服务端权限、积分、DNS 或审核判定。
- 新文案同步维护 `en`、`zh`、`zh-Hant`、`es`、`ru`、`ja`。
- 延续现有 Shadcn、Tailwind 和图标体系，兼顾桌面与移动布局。
- 资料字段以服务端持久化值为准，登录刷新不得重置用户自定义姓名或头像。

## 7. API 与幂等

所有 API 位于 `/api/v1`。网页会话通过 `Authorization: Bearer` 传递。Android 请求还携带 `X-HL6-Client-Key`。

非安全方法由统一客户端生成 `X-Idempotency-Key`。Handler 需要幂等时通过公共提取函数读取，Service 以业务 Scope 和 Key 持久化结果。不要在页面重试中生成第二个键，除非用户明确发起新操作。

## 8. 数据库与迁移

服务启动时自动迁移模型。新增字段应：

1. 选择兼容旧数据的默认/可空策略。
2. 更新模型、Repository 和响应类型。
3. 检查管理员列表、导出和筛选。
4. 评估索引、唯一约束和并发写入。
5. 更新架构文档和升级说明。

禁止为了修复本地开发数据而提交会删除生产数据的自动迁移。

## 9. Android 同步要求

Android 复用整个 `web/` UI。任何影响页面、路由、API、字段、错误码、OIDC、会话或版本行为的修改，都必须检查 Capacitor 构建和原生运行环境。

客户端变更至少检查：

- `https://localhost` CORS。
- `X-HL6-Client-Key` 与 `X-Idempotency-Key`。
- 深链与包名。
- Keystore 会话恢复。
- `pnpm run build` 与 `cap sync android`。
- 普通/强制更新。
- `docs/agent.md` 变更记录。

## 10. 提交前检查

根据改动范围运行最小充分验证：

```bash
cd web && pnpm run lint
cd web && pnpm run build
cd server && go test ./...
git diff --check
```

纯文档修改至少检查本地链接、示例命令、环境变量名称和 `git diff --check`。工作流修改还需由 GitHub Actions 实际解析并执行关键路径。

提交内容保持聚焦。不要把无关格式化、生成文件、APK、keystore、`.env` 或个人工具配置混入变更。

## 11. 安全评审

变更涉及以下内容时提高评审级别：

- 认证、会话、OIDC、管理员判断或 CORS。
- DNS 写操作、幂等、批量任务或域名迁移。
- 支付签名、积分入账或人工补偿。
- 文件上传、外部 URL、HTML/Markdown 内容。
- 加密密钥、API Key、通讯密钥和日志。
- Release、Docker 推送、Android 签名和公开下载。
