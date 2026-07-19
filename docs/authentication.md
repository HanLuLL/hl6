# 邮箱认证

## 概述

HL6 v2 使用自主邮箱/密码认证。密码由服务端验证，使用 Argon2id 加上服务端胡椒进行哈希，永不返回、记录、邮件发送或以可逆形式存储。

Web 应用使用安全的 HttpOnly、SameSite=Lax 会话 Cookie。打包的 Android 客户端在提供通讯密钥后接收短期 Bearer 会话，并使用 Android Keystore 安全存储保存该会话。

## 新注册

1. 打开 `/register` 并提交邮箱地址
2. HL6 评估注册可用性和管理员的精确域名策略：如果邮箱域名被策略拒绝（黑名单拦截或不在白名单中），将返回明确的错误提示告知用户该邮箱域名不允许注册；如果通过策略检查，则发送一次性验证链接，不暴露账户是否已存在
3. 验证链接打开 `/set-password`
4. 设置 8 到 128 个 Unicode 码点的密码
5. HL6 原子化创建用户、本地凭证、默认组、积分余额、推荐状态和会话

原始验证令牌是 URL 安全、随机、短期、单次使用、仅存储为 SHA-256 哈希，并在页面读取后立即从浏览器地址栏移除。

注册、账户激活和密码重置请求可携带 `locale`。Web 与 Android 共享界面会提交当前 i18n 语言；旧客户端未提交时，服务端回退到 `Accept-Language`，再回退到英文。认证邮件的主题、正文、按钮、备用链接说明和页脚支持 `en`、`zh`、`zh-Hant`、`es`、`ru`、`ja` 六种语言。

## 现有 v1 账户

硬切换后，现有用户打开 `/activate-account` 或 `/login`，提交历史邮箱地址，按照激活链接并设置新密码。关联的 `users.id` 不变。现有的资料字段、自定义头像、积分余额、域名、DNS 记录、封禁、申诉、通知、支付和审核链接保持完整。

切换后不接受任何旧会话。每个现有用户必须一次性建立本地密码。

## 密码恢复

`/forgot-password` 始终返回中性响应。已激活的账户收到一次性重置链接。完成重置会更新 Argon2id 哈希并递增凭证会话版本，使其他会话失效。

密码设置在开始昂贵的哈希推导之前消耗其一次性链接。成功的密码更改也会消耗该用户所有其他未完成的激活、重置和恢复链接。如果两个不同的链接被并发消耗，凭证锁会在第一次密码更改后拒绝较旧的兄弟链接，因此它无法覆盖新密码。登出会递增相同的会话版本，因此复制的浏览器或 Android Bearer 令牌在登出后无法继续使用。

认证请求按规范化邮箱地址和隐私保护的客户端 IP 哈希独立限速。更改任一值不会绕过另一个限制。

## 邮箱域名策略

管理员在 **管理后台 -> 访问与注册** 配置：

- `unrestricted`：任何语法有效的地址都可以注册
- `allowlist`：只有列表中的精确规范化域名可以注册
- `blocklist`：除了列表中的精确规范化域名外，有效地址都可以注册

通配符域名、部分后缀、格式错误的 IDN、重复条目和模糊的域名拼写会被拒绝。此策略仅适用于新注册。现有用户仍然可以激活和重置密码。

## 密码胡椒

在 PostgreSQL 之外设置这些部署密钥：

```dotenv
AUTH_PASSWORD_PEPPER_ID=v1
AUTH_PASSWORD_PEPPER=高熵随机密钥
AUTH_PREVIOUS_PASSWORD_PEPPER_ID=
AUTH_PREVIOUS_PASSWORD_PEPPER=
```

要轮换，设置新的当前 ID/值并将之前的对放入 previous 变量。使用之前胡椒成功登录会透明地用当前胡椒重新哈希密码。仅在用户有足够时间登录后移除之前的对。

## v1 切换

切换特意设计为仅控制台操作。不会从浏览器或 Android 客户端运行。

1. 部署带有 `AUTH_PASSWORD_PEPPER`、公共 `FRONTEND_URL` 和工作发件人所需的 `SMTP_BOOTSTRAP_*` 值的 v2 镜像。启动配置仅填充缺失的 SMTP 数据库设置。现有用户数据库在显式切换完成之前仍然故意禁用本地登录
2. 从 v2 容器发送并验证 SMTP 测试。这记录了预检所需的时间戳，无需浏览器会话：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin mail test --recipient administrator@example.com
```

3. 从 PostgreSQL 识别现有管理员 ID，然后通过 v2 控制台命令创建并保留服务器记录的导出：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec postgres \
  sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  -c "SELECT id, email FROM users WHERE role = '\''admin'\'' ORDER BY id;"'

docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin maintenance export --created-by-user-id 1
```

导出命令打印已验证的备份记录。复制其 `id`，从维护卷下载或保留存档，并在继续之前保留独立的加密副本。

4. 运行预检命令：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app /app/hl6-admin auth preflight
```

5. 解决每个阻碍项，特别是空白/无效邮箱、规范化的重复邮箱、缺失的默认组、SMTP 测试状态、公共 HTTPS 前端 URL 和已验证的备份
6. 使用导出的备份 ID 执行不可逆切换：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin auth cutover --confirm --backup-id 123
```

7. 重启应用。现有用户随后使用激活流程。

对于邮件投递恢复，部署操作员可以创建新的一次性激活链接，而不将其写入数据库或日志：

```bash
docker compose --env-file .env -f docker-compose.prod.yml exec app \
  /app/hl6-admin auth issue-activation --email user@example.com
```

仅通过安全渠道传递该输出。

## QQ 头像回退

如果用户没有自定义头像且其规范化地址匹配数字 `@qq.com`，客户端渲染：

```text
https://q.qlogo.cn/headimg_dl?dst_uin=<QQ_NUMBER>&spec=640&img_type=jpg
```

该 URL 是 HTTPS 以避免混合内容失败。自定义 `avatar_url` 始终优先，登录时永不覆盖它。
