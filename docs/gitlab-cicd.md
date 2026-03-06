# GitLab CI/CD

GitLab 使用 `/.gitlab-ci.yml` 执行后端镜像构建与前端 Pages 发布。

## 触发分支

- `main`

## 打包与发布

CI 文件：`/.gitlab-ci.yml`

包含三个 Job：

- `backend:docker`：使用 Kaniko 仅构建后端镜像（`server/Dockerfile`），推送到 `CI_REGISTRY_IMAGE/server`
- `app:docker`：使用 Kaniko 构建全栈镜像（根目录 `Dockerfile`），推送到 `CI_REGISTRY_IMAGE/app`
- `pages`：构建 `web` 并发布到 GitLab Pages（产物目录 `public/`）

默认镜像标签：

- 分支标签：`$CI_COMMIT_REF_SLUG`
- 最新标签：`latest`

如果要给雨云云应用商店提供单镜像部署，直接使用 `CI_REGISTRY_IMAGE/app:latest` 即可；这个镜像会同时包含前端静态资源和 Go 后端。

## 容器仓库认证（GitLab 最佳实践）

`backend:docker` job 使用 GitLab 预定义变量登录容器仓库：

- `CI_REGISTRY`
- `CI_REGISTRY_USER`
- `CI_REGISTRY_PASSWORD`

## 前端后端地址配置

`pages` job 支持在构建时注入 `VITE_API_BASE_URL`：

- 未配置时默认：`/api/v1`（同域反向代理场景）
- 跨域部署时建议设置为完整地址，例如：`https://api.houlang.cloud/api/v1`

请在 GitLab `Settings -> CI/CD -> Variables` 中添加：

- `VITE_API_BASE_URL`（按你的后端地址填写）

后端运行环境还需要与 Pages 域名对齐：

- `BACKEND_URL`：后端对外访问地址（用于 OIDC callback，例如 `https://api.example.com`）
- `FRONTEND_URL`：设置为你的 Pages 访问地址
- `ALLOWED_ORIGINS`：包含你的 Pages 访问地址（可逗号分隔多个）
