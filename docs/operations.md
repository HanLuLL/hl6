# 运维

## 健康与日志

```bash
curl -fsS https://domain.example.com/health
docker compose --env-file .env -f docker-compose.prod.yml logs -f app
```

日志可能包含请求路径、状态和清理后的错误。不得包含密码值、原始邮箱令牌、通讯密钥、会话令牌、SMTP 密钥、带密码的数据库 URL、支付密钥或私有 DNS 凭证。

## 认证事件

| 症状 | 操作 |
| --- | --- |
| 注册邮件未发送 | 在管理后台验证 SMTP，发送测试邮件，检查邮件日志，并验证公共前端 URL 确认 |
| 现有用户无法激活 | 使用 `hl6-admin auth preflight` 检查规范化邮箱冲突；仅在无法恢复邮件投递时使用控制台恢复 |
| 所有用户被登出 | 验证胡椒/会话配置和数据库健康。密码重置或会话版本更改会故意使旧会话失效 |
| 轮换后密码哈希失败 | 将之前的胡椒恢复为 `AUTH_PREVIOUS_PASSWORD_PEPPER` 并保留直到成功登录重新哈希账户 |

## 数据库导出与恢复

导出存档是短期保留的操作文件，不是您唯一的灾难恢复副本。下载它们，验证记录的 SHA-256，独立加密，并在隔离环境中测试恢复。

恢复期间，维护模式阻止 API 访问（恢复作业状态除外）。如果 `pg_restore` 完全无法启动（例如可执行文件不可用），失败的作业会释放维护模式。一旦 `pg_restore` 已启动，无论命令成功还是失败，维护模式都保持活动。仅在操作员验证恢复的数据库或从保留的恢复前存档恢复后重启；永不针对部分恢复的数据库重新开放正常流量。

## 切换恢复

在 v2 切换之前，在服务器外保留已验证的存档。如果切换在完成前失败，解决预检错误并仅在验证数据库状态后重试。如果必须放弃已完成的切换，恢复已验证的切换前存档并使用匹配的应用镜像；永不将旧镜像指向部分迁移的数据库。

对于 v1 数据库，在运行 `auth preflight` 之前使用 v2 容器命令创建所需的已验证切换前存档：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin maintenance export --created-by-user-id <administrator-id>
```

该命令特意仅控制台操作，验证操作员是管理员，并记录后续切换使用的备份 ID。

如果迁移使用 SMTP 启动配置，在切换前验证投递并记录所需的预检时间戳：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin mail test --recipient administrator@example.com
```

## Android 运维

在轮换通讯密钥之前构建新 APK。在启用强制更新之前验证直接 APK URL、SHA-256、应用包名、签名证书和版本策略。保持之前的可选更新路径可用直到确认新 APK 可安装。