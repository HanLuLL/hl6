# HL6 文档

## 从这里开始

1. [部署](deployment.md)：生产环境、Docker 镜像、反向代理、SMTP、v1 切换和回滚
2. [认证](authentication.md)：注册、激活、密码重置、域名策略、会话和密码胡椒轮换
3. [管理后台](administration.md)：分组管理区域、Android 控制、导出和恢复
4. [Android 客户端](android-client.md)：本地 UI 包模型、通讯密钥、签名密钥和工作流输入

## 参考

| 需求 | 文档 |
| --- | --- |
| 端点、负载和错误行为 | [API](api.md) |
| 组件、数据所有权和安全边界 | [架构](architecture.md) |
| 监控、备份恢复和事件响应 | [运维](operations.md) |
| 本地开发和测试命令 | [开发](development.md) |
| 强制 Android 适配流程 | [Agent 约定](agent.md) |

## 版本范围

本文档描述 `v2.0.0`。认证模型为 HL6 自主邮箱/密码认证；提供商登录端点和回调配置不属于此版本。