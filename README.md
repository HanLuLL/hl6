# HL6

HL6 是一个域名/子域名管理平台，支持在已注册域名下认领和管理子域名、维护 DNS 记录，并基于积分规则控制访问。

> **原项目地址**：[https://git.houlang.cloud/houlangcloud/hl6](https://git.houlang.cloud/houlangcloud/hl6)
>
> **本项目地址**：[https://github.com/HanLuLL/hl6](https://github.com/HanLuLL/hl6)
>
> **容器镜像**：`ghcr.milu.moe/hanlull/hl6:latest`

---

## 文档导航

| 文档 | 说明 |
|------|------|
| [快速部署](docs/deployment.md) | Docker Compose 生产环境部署指南、环境变量说明、升级流程 |
| [开发环境搭建](docs/development.md) | 从零搭建本地开发环境，包含全平台依赖安装指南 |
| [OIDC 认证配置](docs/oidc.md) | 10+ 种 OIDC 提供商的配置方法（Logto、Keycloak、Authing、Google 等） |
| [架构设计](docs/architecture.md) | 系统架构、目录结构、数据模型、API 设计 |
| [故障排查](docs/troubleshooting.md) | 常见问题诊断与解决方案 |

## 核心功能

- **域名与子域名管理**：管理员添加域名，用户认领子域名
- **DNS 记录管理**：支持 A / AAAA / CNAME / TXT，多 DNS 提供商（Cloudflare、阿里云、DNSPod、华为云等）
- **积分系统**：签到、推荐奖励、充值（支付宝/微信/QQ，支持易支付和码支付网关）
- **内容审核**：自动化扫描、规则引擎、违规处置
- **用户管理**：OIDC 认证、用户组、封禁管理
- **SEO 优化**：sitemap.xml、robots.txt、元数据管理、Google Analytics 集成
- **站点定制**：品牌 Logo/Favicon、公告系统、页脚备案信息
- **多语言**：中文简繁体、英语、日语、西班牙语、俄语

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | React 19 + TypeScript + Vite + Tailwind CSS 4 + TanStack React Query |
| 后端 | Go (Gin + GORM) + PostgreSQL 16 |
| 认证 | OIDC 兼容 (Logto / Keycloak / Authentik / Authing / Google / Zitadel 等) + JWT |
| 基础设施 | Docker Compose, Cloudflare DNS API |

## 快速开始

```bash
# 克隆仓库
git clone https://github.com/HanLuLL/hl6.git
cd hl6

# 配置环境变量
cp .env.example .env
# 编辑 .env 填写必要配置

# 启动开发环境 (PostgreSQL + Go 后端 + Vite 前端)
make dev
```

详细部署指南见 [docs/deployment.md](docs/deployment.md)。

## 容器部署

```bash
# 使用预构建镜像
docker pull ghcr.milu.moe/hanlull/hl6:latest

# 或使用 Docker Compose
# 参考 docs/deployment.md
```

## 环境变量

| 变量名 | 必填 | 说明 |
|--------|------|------|
| `DATABASE_URL` | 是 | PostgreSQL 连接串 |
| `OIDC_ISSUER` | 否* | OIDC 提供商 Issuer URL |
| `OIDC_CLIENT_ID` | 否* | OIDC 应用 Client ID |
| `OIDC_CLIENT_SECRET` | 否* | OIDC 应用 Client Secret |
| `SESSION_SECRET` | 否 | 会话密钥种子（留空自动生成） |
| `FRONTEND_URL` | 否 | 前端地址（支持多地址，逗号分隔） |
| `BACKEND_URL` | 否 | 后端对外地址 |
| `ALLOWED_ORIGINS` | 否 | CORS 白名单 |
| `ENCRYPTION_KEY` | 否 | AES-256-GCM 密钥（64 位十六进制） |
| `EPAY_URL` / `EPAY_PID` / `EPAY_KEY` | 否 | 易支付网关配置 |
| `CODEPAY_URL` / `CODEPAY_ID` / `CODEPAY_KEY` | 否 | 码支付网关配置 |

> *OIDC 三项可留空，首次启动时通过 Web UI 初始化向导配置（仅限系统无用户时）。
> 系统首个注册用户自动成为管理员。

完整环境变量说明见 `.env.example` 和 [docs/deployment.md](docs/deployment.md)。

## 项目结构

```
.
├── server/                 # Go 后端
│   ├── cmd/server/         # 程序入口
│   ├── internal/           # handler / service / repository / router / model
│   └── pkg/                # response / validator / crypto / queue
├── web/                    # React 前端
│   └── src/                # pages / components / hooks / lib / i18n / types
├── docs/                   # 项目文档
├── Dockerfile              # 多阶段构建（前端 + 后端）
├── Makefile                # 开发命令入口
└── .env.example            # 环境变量模板
```

## 开发命令

```bash
make dev           # 启动完整开发栈 (DB + Go server + Vite)
make dev-server    # 仅启动 Go 后端
make dev-web       # 仅启动前端开发服务器
make db-up         # 启动 PostgreSQL
make db-down       # 停止 PostgreSQL
```

## 致谢

本项目基于 [厚浪开发组](https://houlang.cloud) 的 [HL6](https://git.houlang.cloud/houlangcloud/hl6) 项目，遵循 AGPL-3.0 开源协议。

## 许可证

[GNU Affero General Public License v3.0 (AGPL-3.0)](LICENSE)
