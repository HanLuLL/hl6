# 客户端适配入口

所有影响打包 Android 客户端的工作必须遵循 [docs/agent.md](docs/agent.md)。

Android 包基于本地共享 React UI 构建，通过 API 与 HL6 通信。不得加载远程网站作为主 UI，不得复制服务端业务逻辑，不得绕过服务端授权。

对于 v2，原生认证使用直接邮箱/密码 API、Android Keystore 会话存储和服务端通讯密钥请求头。任何影响客户端的更改必须在同一变更集中更新 `docs/agent.md`。