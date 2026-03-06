# HL6 雨云云应用商店上架建议

本文基于当前仓库结构整理，目标是让 HL6 在雨云中以一个应用模板完成安装，推荐使用双容器方案：

- `main`：HL6 全栈容器，单端口同时提供前端页面和后端 API
- `postgres`：PostgreSQL 16 数据库容器

这样用户只需要安装一个应用，不需要再单独拼接前端、后端和反向代理。

如果你准备使用雨云版本页的“从 Docker 导入 / Compose 导入”功能，仓库里已经提供了可直接作为导入起点的文件：

- [docker-compose.rainyun.yml](../docker-compose.rainyun.yml)

这个文件使用标准 Docker Compose 语义描述 `main + postgres` 双容器结构，更适合直接喂给导入器。
同时它避免了“引用了未定义环境变量”的写法，兼容雨云导入器对 `${ENV_NAME}` 的校验规则。

## 推荐镜像

- `main` 镜像：`<你的镜像仓库>/hl6-app:latest`
- 如果使用当前 GitLab CI：`$CI_REGISTRY_IMAGE/app:latest`
- `postgres` 镜像：`postgres:16-alpine`

仓库根目录 `Dockerfile` 已经适配这个部署形态，会把 `web/dist` 打进最终镜像，由 Go 服务直接托管前端。

使用 `docker-compose.rainyun.yml` 导入前，只需要先把里面的 `main.image` 改成你真实可拉取的镜像地址。
导入成功后，再把文件里的示例默认值改成你模板里的真实默认值或 Options 引用。

## 版本基本信息

- 版本号：建议使用项目发布版本号，例如 `0.1.0`
- 镜像：`main` 容器填写全栈镜像地址
- 最小 CPU：建议 `1` 核
- 最小内存：建议 `1024` MB

以上 CPU 和内存是按当前技术栈给出的保守建议值，便于商店用户开箱即用。

## 容器设计

### `main` 容器

- 镜像：`<你的镜像仓库>/hl6-app:latest`
- Command：留空
- Args：留空
- 启动端口：`8080`

### `postgres` 容器

- 镜像：`postgres:16-alpine`
- Command：留空
- Args：留空

## Compose 导入说明

雨云在 2025 年 12 月已经支持从 Docker Compose 导入容器定义。[官方更新公告](https://forum.rainyun.com/t/topic/12843) 说明可以通过版本编辑页的“从 Docker 导入”入口，把 Compose 快速转换成云应用容器。

当前仓库的 [docker-compose.rainyun.yml](../docker-compose.rainyun.yml) 采用的是标准 Compose 写法：

- `main` 对外暴露 `8080:8080`
- `postgres` 只通过 `expose: 5432` 提供内部连接
- `main` 默认使用 `postgres:5432` 作为数据库主机名，避免导入时特殊变量未替换导致启动失败
- `main` 里所有被 `${...}` 引用的变量，都先在同一个容器的环境变量列表里显式定义

导入后请检查一项内容：

- 如果你改了数据库容器名，不是 `postgres`，同步修改 `DATABASE_URL` 里的主机名

常见报错：

- `invalid character "{" in host name`

这表示 `DATABASE_URL` 里的 `${rca_svc_...}` 没有被平台替换。可用两种方式修复：

- 方式 A（推荐，最稳）：直接用容器服务名，例如 `@postgres:5432`
- 方式 B：确认容器名和服务名完全匹配后，再用 `${rca_svc_[容器名]_[服务名]}` 格式

这是因为在标准 Docker Compose 中多容器一般直接通过服务名互连，而雨云 RCA 官方说明里推荐在模板层使用 `${rca_svc_[容器名]_[服务名]}` 这种特殊环境变量。

## Env 环境变量

### `main` 容器

建议在模板中配置以下环境变量：

| Key | Value |
| --- | --- |
| `SERVER_PORT` | `8080` |
| `APP_URL` | `${APP_URL}` |
| `FRONTEND_URL` | `${APP_URL}` |
| `BACKEND_URL` | `${APP_URL}` |
| `ALLOWED_ORIGINS` | `${APP_URL}` |
| `DATABASE_URL` | `postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable` |
| `OIDC_ISSUER` | `${OIDC_ISSUER}` |
| `OIDC_CLIENT_ID` | `${OIDC_CLIENT_ID}` |
| `OIDC_CLIENT_SECRET` | `${OIDC_CLIENT_SECRET}` |
| `SESSION_SECRET` | `${SESSION_SECRET}` |
| `ENCRYPTION_KEY` | `${ENCRYPTION_KEY}` |
| `ADMIN_EMAILS` | `${ADMIN_EMAILS}` |

说明：

- `APP_URL` 填应用的公网访问地址，建议使用 `http://` 或 `https://` 的完整 URL。
- 当前实现是同域部署，前端和 API 共用一个公网地址，所以 `FRONTEND_URL`、`BACKEND_URL` 和 `ALLOWED_ORIGINS` 都可以直接复用 `APP_URL`。
- `DATABASE_URL` 默认按 Compose 服务名互连，这里假设 PostgreSQL 容器名是 `postgres`。
- 如果你改了数据库容器名，同步修改 `DATABASE_URL` 主机名部分。
- `SESSION_SECRET` 不能为空，否则 JWT 会话签发和校验会失败。
- `ENCRYPTION_KEY` 建议必填，并限制为 64 位十六进制字符串；不填时 Cloudflare Token 会以明文形式保存在数据库中。

### `postgres` 容器

| Key | Value |
| --- | --- |
| `POSTGRES_DB` | `${POSTGRES_DB}` |
| `POSTGRES_USER` | `${POSTGRES_USER}` |
| `POSTGRES_PASSWORD` | `${POSTGRES_PASSWORD}` |

## Services 服务配置

### `main`

- 服务名称：`http`
- 显示名称：`Web`
- 服务类型：`外部访问`
- 内部端口：`8080`
- 外部端口：`8080`
- 协议：`tcp`

### `postgres`

- 服务名称：`db`
- 显示名称：`PostgreSQL`
- 服务类型：`内部访问`
- 内部端口：`5432`
- 外部端口：留空
- 协议：`tcp`

## VolumeMounts 持久化卷挂载

### `postgres`

- 名称：`postgres-data`
- 挂载路径：`/var/lib/postgresql/data`
- 子路径：`postgres`
- 内容类型：`目录`

### `main`

当前项目运行时没有必须挂载到本地磁盘的业务文件：

- 通知图片存储在 PostgreSQL `bytea`
- Cloudflare 账号、系统配置、积分等都在 PostgreSQL 中

因此 `main` 容器可以不配置持久化卷。

## Options 选项建议

推荐把下面这些值暴露给用户填写：

| 标签 | 环境变量键 | 类型 | 默认值 | 必填 | 备注 |
| --- | --- | --- | --- | --- | --- |
| 应用访问地址 | `APP_URL` | 文本 | 空 | 是 | 必须是完整 URL |
| 管理员邮箱 | `ADMIN_EMAILS` | 文本 | 空 | 是 | 支持逗号分隔多个邮箱 |
| OIDC Issuer | `OIDC_ISSUER` | 文本 | 空 | 是 | 例如 `https://your-provider.example.com/oidc` |
| OIDC Client ID | `OIDC_CLIENT_ID` | 文本 | 空 | 是 | OIDC 应用 ID |
| OIDC Client Secret | `OIDC_CLIENT_SECRET` | 文本 | 空 | 是 | OIDC 应用密钥 |
| Session Secret | `SESSION_SECRET` | 文本 | `hl6` | 是 | 建议启用随机生成 |
| Encryption Key | `ENCRYPTION_KEY` | 文本 | 空 | 是 | 验证规则建议为 `^[a-fA-F0-9]{64}$` |
| PostgreSQL 数据库名 | `POSTGRES_DB` | 文本 | `hl6` | 是 | 数据库名 |
| PostgreSQL 用户名 | `POSTGRES_USER` | 文本 | `hl6` | 是 | 数据库用户名 |
| PostgreSQL 密码 | `POSTGRES_PASSWORD` | 文本 | `hl6db` | 是 | 建议自行修改 |

补充建议：

- `SESSION_SECRET` 可以启用“随机生成”，这样安装时会自动带随机后缀。
- `ENCRYPTION_KEY` 不建议用雨云的“随机生成”直接代替，因为项目要求固定长度的 64 位十六进制字符串。
- 如果雨云选项支持正则校验，`APP_URL` 建议校验完整 URL 格式，`ADMIN_EMAILS` 可保留文本输入，由用户按逗号分隔填写。

## 上架说明建议

商店描述里建议明确写出以下前置条件：

- 需要一个可访问的 OIDC 提供商，HL6 不内置账号密码登录
- 首次安装后，管理员需要登录后台配置 Cloudflare 账号和域名
- 建议为应用配置 HTTPS，再将 HTTPS 地址填入 `APP_URL`

## 已完成的仓库适配

为了适配这种部署形态，仓库已经补上：

- 根目录全栈 `Dockerfile`
- 后端静态文件托管逻辑，存在 `web/dist` 时自动提供 SPA
- `APP_URL` 作为同域部署的简化入口
- GitLab CI 的 `app:docker` 任务
