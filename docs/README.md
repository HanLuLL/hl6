# HL6 文档中心

本目录是 HL6 的长期维护文档。内容按读者角色拆分，每个主题只有一个主文档；Release Notes 只描述正式版本能力，不承担部署教程或提交日志的职责。

## 第一次使用

1. 阅读[生产部署与升级](deployment.md)，选择国际或中国大陆镜像并启动 PostgreSQL 与 HL6。
2. 阅读[OIDC 认证配置](oidc.md)，登记唯一的网页回调地址并完成首配。
3. 登录管理后台后，按[管理后台配置](administration.md)添加 DNS 提供商、域名、用户组和运营配置。
4. 需要 Android 时，再阅读[Android 客户端构建与发布](android-client.md)。

## 按角色阅读

| 角色 | 推荐文档 | 解决的问题 |
| --- | --- | --- |
| 部署人员 | [生产部署与升级](deployment.md) | Compose、镜像、环境变量、HTTPS、升级、回滚和备份 |
| 系统管理员 | [管理后台配置](administration.md) | 域名、用户、积分、支付、审核、品牌和客户端策略 |
| 身份平台管理员 | [OIDC 认证配置](oidc.md) | Issuer、Client、回调、首配和 Android 登录 |
| Android 维护者 | [Android 客户端](android-client.md) | 构建参数、签名、通讯密钥、APK、更新和故障排查 |
| 开发者 | [开发指南](development.md) | 本地环境、命令、代码边界和提交检查 |
| 架构评审者 | [系统架构](architecture.md) | 组件、数据模型、认证、DNS、审核和支付流程 |
| API 集成者 | [API 集成指南](api.md) | 响应结构、鉴权、幂等、路由族和错误处理 |
| 值班人员 | [运维与故障排查](operations.md) | 健康检查、日志、队列、数据库和常见事故处理 |

## 配置来源

HL6 的配置分为三类：

- 启动配置：数据库、监听端口、外部 URL、加密密钥和可选 Redis，通过环境变量注入。
- 动态系统配置：品牌、SEO、支付、公告、客户端版本、OIDC 等由管理后台写入 `SystemConfig`。
- 构建配置：Android 域名、通讯密钥、名称、图标、版本、包名和签名 Secret 由客户端工作流注入。

当同一字段同时存在于环境变量和数据库时，环境变量优先。生产部署应在文档中明确记录哪些字段由环境变量锁定，避免后台修改后看似未生效。

## 安全约定

- 不提交 `.env`、数据库密码、OIDC Client Secret、DNS API 凭据、Android keystore 或通讯密钥。
- 生产环境使用 HTTPS，并准确配置 `APP_URL`、`BACKEND_URL`、`FRONTEND_URL` 与 `ALLOWED_ORIGINS`。
- 设置 32 字节 `ENCRYPTION_KEY`，保护数据库中的 OIDC、DNS 和 AI API 密钥。
- Android 通讯密钥是可轮换的客户端构建标识，不是用户权限；用户操作仍必须通过会话和服务端授权。
- 正式 Release 只在人工确认版本、说明和 APK 后创建，不由普通提交自动触发。

## 文档维护

代码改动若影响配置、API、部署、用户界面或 Android 行为，必须同步更新对应主文档。客户端相关变更还必须遵守[Android 客户端适配规范](agent.md)，并在其变更记录中追加一条简洁记录。
