# HL6 与 Authing 集成指南

## 概述

本指南将详细介绍如何将 Authing 作为 OIDC 提供商与 HL6 集成。HL6 支持所有标准 OIDC 提供商，包括 Authing。

## 概述

[Authing](https://authing.cn/) 是国内领先的、基于云原生架构的 IDaaS 产品，兼容 OAuth 2.0, OIDC, SAML, AD/LDAP, WS-Fed 等所有主流认证/授权协议，可以帮助你快速实现任何 Web、App 和企业软件的身份认证和用户管理。

## 兼容性说明

HL6 针对 Authing 的 OIDC 实现进行了特别兼容性优化，包括：
- 支持多种用户信息字段映射（name, username, nickname, given_name, family_name 等）
- 支持多种头像字段（picture, avatar, avatar_url 等）
- 支持多种邮箱字段（email, email_address 等）
- 支持无kid（Key ID）的JWT令牌验证（Authing特有问题）
- 支持灵活的 JWT 验证机制
- 支持多种 HTTP 状态码响应

## 配置步骤

### 1. 在 Authing 控制台创建应用

1. 登录 [Authing 控制台](https://console.authing.cn/)
2. 进入 **应用** → **应用列表**，点击「添加应用」
3. 填写应用信息：
   - **应用名称**：填写你的 HL6 应用名称
   - **认证地址**：填写一个域名，作为这个应用在 Authing 的唯一标识
   - **回调链接**：填写 `https://your-hl6-domain.com/api/v1/auth/callback`

### 2. 配置应用安全设置

在应用详情页面的「高级配置」选项卡中：

1. 在「安全性」卡片中，将 **id_token 签名算法** 选择为 **RS256**，然后点击「保存」
2. 配置以下身份验证方式为 **none**：
   - **换取 token 身份验证方式**
   - **检验 token 身份验证方式** 
   - **撤回 token 身份验证方式**
3. 点击「保存」

### 3. 配置回调地址

在应用详情页面的「登录配置」中：

1. **登录回调地址**：添加 `https://your-hl6-domain.com/api/v1/auth/callback`
2. **登出回调地址**：添加 `https://your-hl6-domain.com`（用于登出后跳转）

### 4. 记录应用信息

在应用详情页面记录以下信息：

- **应用 ID** (Client ID)
- **应用密钥** (Client Secret) 
- **应用域名** (用于构建 Issuer URL)

## 在 HL6 中配置 Authing

### 方法一：环境变量配置（推荐）

在 HL6 的 `.env` 文件中配置以下环境变量：

```env
# Authing 配置
OIDC_ISSUER=https://<your-app-domain>.authing.cn/oidc
OIDC_CLIENT_ID=<your-app-id>
OIDC_CLIENT_SECRET=<your-app-secret>
```

**注意**：将 `<your-app-domain>` 替换为你的 Authing 应用域名，`<your-app-id>` 和 `<your-app-secret>` 替换为从 Authing 控制台获取的实际值。

### 方法二：通过管理界面配置

1. 登录 HL6 管理员账户
2. 进入 **系统设置** → **OIDC 配置**
3. 填写以下信息：
   - **OIDC Issuer**: `https://<your-app-domain>.authing.cn/oidc`
   - **OIDC Client ID**: 从 Authing 获取的应用 ID
   - **OIDC Client Secret**: 从 Authing 获取的应用密钥

## Authing 用户信息字段映射

Authing OIDC 返回的用户信息字段映射如下：

| HL6 需求字段 | Authing 对应字段 | 备注 |
|-------------|----------------|------|
| 用户名 | `name` 或 `username` | 优先使用 `name` |
| 邮箱 | `email` | - |
| 手机号 | `phone` | - |
| 头像 | `picture` | - |

## 验证配置

配置完成后，重启 HL6 服务，然后：

1. 访问 HL6 登录页面
2. 点击登录按钮，应该会跳转到 Authing 认证页面
3. 使用 Authing 中的用户登录，验证是否能成功返回 HL6

## 故障排除

### 常见问题

1. **回调地址不匹配错误**
   - 确认在 Authing 控制台中已正确配置回调地址
   - 确保回调地址与 HL6 部署的域名完全一致

2. **认证失败**
   - 检查 `OIDC_ISSUER`、`OIDC_CLIENT_ID`、`OIDC_CLIENT_SECRET` 是否正确
   - 确认 Authing 应用的安全配置是否正确（特别是身份验证方式设为 `none`）

3. **用户信息获取失败**
   - 确认 Authing 应用的授权范围包含了 `openid`、`email`、`profile`

### 调试方法

1. 检查 Authing 控制台中的应用日志
2. 确认 `.well-known/openid-configuration` 端点是否可访问：
   ```
   https://<your-app-domain>.authing.cn/oidc/.well-known/openid-configuration
   ```

## 注意事项

- Authing 的 Issuer URL 格式为 `https://<app-domain>.authing.cn/oidc`
- 确保在 Authing 中配置的回调地址与 HL6 部署的域名完全一致
- 如果使用自定义域名，请确保 DNS 解析正确配置
- Authing 支持 RS256 签名算法，确保在应用配置中选择了正确的签名算法
- Authing 支持 `end_session_endpoint`，HL6 会自动处理登出流程