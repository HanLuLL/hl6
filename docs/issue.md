# 常见问题

## 1) `make dev` 启动失败，提示 `npm` 相关错误

先执行：

```bash
cd web && npm install
```

## 2) 端口冲突（`5173` / `8080` / `5432`）

- 修改 `.env` 中 `SERVER_PORT`
- 调整前端端口（Vite 启动参数或配置）
- 修改 `docker-compose.yml` 的数据库端口映射

## 3) 登录相关接口不可用

请确认 `.env` 中 OIDC 配置已填写正确（各提供商的具体配置方法见 [OIDC 配置指南](docs/oidc.md)）：

- `OIDC_ISSUER`
- `OIDC_CLIENT_ID`
- `OIDC_CLIENT_SECRET`