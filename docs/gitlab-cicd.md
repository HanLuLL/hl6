# GitLab CI/CD

GitLab 使用 `/.gitlab-ci.yml` 执行后端镜像构建与前端 Pages 发布。

## 触发分支

- `main`

## 打包与发布

CI 文件：`/.gitlab-ci.yml`

包含两个 Job：

- `backend:docker`：使用 Kaniko 仅构建后端镜像（`server/Dockerfile`），推送到 `CI_REGISTRY_IMAGE/server`
- `pages`：构建 `web` 并发布到 GitLab Pages（产物目录 `public/`）

默认镜像标签：

- 分支标签：`$CI_COMMIT_REF_SLUG`
- 最新标签：`latest`

## 容器仓库认证（GitLab 最佳实践）

`backend:docker` job 使用 GitLab 预定义变量登录容器仓库：

- `CI_REGISTRY`
- `CI_REGISTRY_USER`
- `CI_REGISTRY_PASSWORD`
