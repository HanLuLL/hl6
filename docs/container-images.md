# 容器镜像

HL6 只在 GitHub Container Registry 构建一次正式镜像，国内地址是该镜像的拉取代理，不是第二套来源代码或独立构建产物。两种地址应解析到相同版本标签。

| 网络环境 | 镜像地址 |
| --- | --- |
| 国际网络 | `ghcr.io/hanlull/hl6:latest` |
| 中国大陆网络 | `ghcr.milu.moe/hanlull/hl6:latest` |
| 开发分支 | 分别使用上述地址的 `dev` 标签 |
| 固定提交 | 使用 `sha-<commit>`（生产）或 `dev-<commit>`（开发）标签 |

## 拉取与验证

```bash
# 国际网络
docker pull ghcr.io/hanlull/hl6:latest

# 中国大陆网络
docker pull ghcr.milu.moe/hanlull/hl6:latest
```

拉取后应固定使用已经验证的标签，不建议长期依赖 `latest`。可用以下命令检查镜像摘要：

```bash
docker image inspect ghcr.io/hanlull/hl6:latest --format '{{index .RepoDigests 0}}'
docker image inspect ghcr.milu.moe/hanlull/hl6:latest --format '{{index .RepoDigests 0}}'
```

## Compose 选择

`docker-compose.prod.yml` 读取 `HL6_IMAGE`。默认是国际 GHCR；在中国大陆环境将其改为代理地址：

```dotenv
# 国际网络
HL6_IMAGE=ghcr.io/hanlull/hl6:latest

# 中国大陆网络
# HL6_IMAGE=ghcr.milu.moe/hanlull/hl6:latest
```

然后执行：

```bash
docker compose -f docker-compose.prod.yml --env-file .env pull app
docker compose -f docker-compose.prod.yml --env-file .env up -d app
```

## PostgreSQL 镜像

生产 Compose 默认使用 `postgres:16-alpine`。当 Docker Hub 在部署网络不可达时，可在 `.env` 中单独指定已审批的 PostgreSQL 镜像地址：

```dotenv
POSTGRES_IMAGE=your-approved-registry.example/library/postgres:16-alpine
```

该变量只影响数据库服务；不要因为 HL6 应用镜像拉取问题而全局修改 Docker daemon 的镜像源。

## 自动发布规则

- `main` 推送发布国际镜像 `latest` 和 `sha-<commit>`。
- `dev` 推送发布国际镜像 `dev` 和 `dev-<commit>`，不会覆盖生产 `latest`。
- `v*` 标签发布对应版本标签；国内代理按需读取同一 GHCR 镜像。

国内代理不可用时，先确认 `docker pull ghcr.milu.moe/hanlull/hl6:latest` 的错误信息；代理恢复前可使用国际地址或从当前源码本地构建：

```bash
docker build -t hl6:local .
```

不要把未验证的第三方 registry mirror 写入 Docker daemon 全局配置，也不要将部署配置指向不存在的独立国内仓库。
