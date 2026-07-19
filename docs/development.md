# 开发

## 本地环境

```bash
make db-up
make dev-server
make dev-web
```

Web 开发服务器将 `/api` 代理到 Go 服务器。Vite 通过 `envDir: ".."` 读取根目录 `.env`。

## 必需的本地变量

```dotenv
DATABASE_URL=postgres://hl6:hl6dev@localhost:5433/hl6?sslmode=disable
AUTH_PASSWORD_PEPPER_ID=dev-v1
AUTH_PASSWORD_PEPPER=仅开发用的长随机值
FRONTEND_URL=http://localhost:5174
BACKEND_URL=http://localhost:8081
ALLOWED_ORIGINS=http://localhost:5174,https://localhost
```

不要在本地开发中使用生产密码胡椒、SMTP 凭证、支付密钥、DNS 密钥、签名密钥或通讯密钥。

## 检查

```bash
cd web
pnpm run lint
pnpm run build

cd ../server
go test ./internal/auth ./internal/migration ./internal/service ./internal/repository ./internal/middleware -count=1
go build ./cmd/server
go build ./cmd/hl6-admin
```

需要 PostgreSQL 的 Handler 集成测试使用 `HL6_TEST_DATABASE_URL`。使用隔离的数据库。维护恢复测试绝不能以共享或生产数据库为目标。

## 页脚版本号

Web 端页脚会自动显示当前构建的版本号、Git 分支和提交哈希。这些信息通过 Vite 构建时注入的全局常量获取：

- `__APP_VERSION__` — 来自 `package.json` 的 `version` 字段
- `__APP_GIT_BRANCH__` — 当前 Git 分支名
- `__APP_GIT_COMMIT__` — 当前 Git 提交的短哈希

当值为 `unknown`（分支/提交）或 `dev`（版本）时，对应的徽标会自动隐藏。发布构建时应确保这些值被正确注入，以便用户在页脚看到版本信息。

## Android

```bash
cd web
pnpm run build
pnpm exec cap sync android
cd android
./gradlew :app:assembleDebug --no-daemon
```

在任何 UI、认证、API、客户端版本或打包更改之前查看 [agent.md](agent.md)。