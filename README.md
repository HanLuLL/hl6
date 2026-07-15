# HL6

HL6 是一个面向个人与团队的域名、子域名和 DNS 管理平台。管理员统一维护可认领域名、DNS 提供商、用户组和风控策略，用户通过 OIDC 登录后认领子域名、维护 DNS 记录、使用积分与支付能力，并可在网页端或 Android 客户端完成同一套操作。

> 原项目：[厚浪开发组 / HL6](https://git.houlang.cloud/houlangcloud/hl6)
>
> 当前项目：[HanLuLL/hl6](https://github.com/HanLuLL/hl6)
>
> 许可证：[GNU AGPL-3.0](LICENSE)

## 正式版本

- 服务端与网页端：`v1.0.0`
- Android 客户端：`1.0.0`，包名 `cc.lxii.domain`
- [GitHub Releases](https://github.com/HanLuLL/hl6/releases/latest)
- [直接下载最新 APK](https://hanlull.github.io/hl6/android/cc.lxii.domain/latest.apk)

## 核心能力

- 域名与子域名：域名配置、保留前缀、可用性检查、子域名认领、释放和管理员接管。
- DNS：A、AAAA、CNAME、TXT 记录，Cloudflare、阿里云 DNS、DNSPod、华为云等多提供商账号，批量任务和域名迁移。
- 用户与权限：OIDC 单点登录、资料持久化、用户组、角色、额度、封禁时间、封禁通知和申诉。
- 积分与支付：余额、交易记录、每日签到、推荐奖励、管理员发放、易支付与码支付订单。
- 内容安全：规则审查、定时扫描、手动复扫、自动处置、AI 模型与提示词配置、二次审核。
- 站点运营：品牌名称、Logo、Favicon、公告、通知、邮件记录、友情链接、SEO、页脚与统计配置。
- Android 客户端：复用网页端 React UI，通过 API 通信，支持原生 OIDC、Keystore 会话、通讯密钥、版本检查和强制更新。
- 运维：单镜像部署、PostgreSQL、可选 Redis 队列、健康检查、备份恢复、国际与中国大陆镜像地址。

## 镜像与下载

```bash
# 国际网络
docker pull ghcr.io/hanlull/hl6:latest

# 中国大陆网络
docker pull ghcr.milu.moe/hanlull/hl6:latest
```

正式版本可将 `latest` 替换为 `v1.0.0`。中国大陆地址是 GHCR 代理；镜像内容与 `ghcr.io/hanlull/hl6` 保持一致。

## 快速部署

准备 Docker Engine 24+ 与 Docker Compose v2，然后执行：

```bash
git clone https://github.com/HanLuLL/hl6.git
cd hl6
cp .env.example .env
```

在 `.env` 中至少补充生产数据库参数，并设置公网 URL：

```dotenv
POSTGRES_DB=hl6
POSTGRES_USER=hl6
POSTGRES_PASSWORD=请替换为高强度随机密码
APP_URL=https://domain.example.com
```

启动服务：

```bash
docker compose --env-file .env -f docker-compose.prod.yml up -d
docker compose --env-file .env -f docker-compose.prod.yml ps
```

访问 `APP_URL`。当数据库中还没有用户且 OIDC 环境变量为空时，可先通过网页首配向导写入 OIDC 配置；首个完成登录的用户自动成为管理员。完整步骤、安全配置和反向代理示例见[生产部署与升级](docs/deployment.md)。

## 文档

| 目标 | 文档 |
| --- | --- |
| 选择阅读路径 | [文档中心](docs/README.md) |
| 部署、升级、回滚与备份 | [生产部署与升级](docs/deployment.md) |
| 配置管理后台全部功能 | [管理后台配置](docs/administration.md) |
| 对接身份提供商 | [OIDC 认证配置](docs/oidc.md) |
| 构建和发布 Android | [Android 客户端](docs/android-client.md) |
| 搭建开发环境 | [开发指南](docs/development.md) |
| 理解组件与数据流 | [系统架构](docs/architecture.md) |
| 调用服务端接口 | [API 集成指南](docs/api.md) |
| 排查生产问题 | [运维与故障排查](docs/operations.md) |

## 技术栈

| 层 | 技术 |
| --- | --- |
| 前端 | React 19、TypeScript、Vite、Tailwind CSS 4、TanStack Query、Shadcn UI、i18next |
| 后端 | Go、Gin、GORM、PostgreSQL 16、可选 Redis Streams |
| 认证 | OIDC、服务端签发会话令牌、Android 原生授权码交换 |
| DNS | Cloudflare、阿里云 DNS、DNSPod、华为云及提供商抽象层 |
| Android | Capacitor、Android SDK 36、JDK 21、Android Keystore |
| 交付 | Docker、GHCR、GitHub Actions、GitHub Pages、GitHub Releases |

## 开发命令

```bash
make dev           # PostgreSQL + Go 服务端 + Vite
make dev-server    # 仅 Go 服务端
make dev-web       # 仅 Vite 前端
make db-up         # 启动开发数据库
make db-down       # 停止开发数据库
```

贡献前请阅读[开发指南](docs/development.md)和[Android 客户端适配规范](docs/agent.md)。任何影响 Android 展示、接口或登录的改动必须在同一变更中完成客户端适配。
