# Quickstart: 多 DNS 供应商 + 域名迁移队列

## 1. 前置条件

- 分支：`001-add-dns-providers`
- 数据库：PostgreSQL 16 已启动（`make db-up`）
- 目标：一次性交付 10 家供应商统一读写能力（排除 `dnsdun`）
- 原则：官方 SDK 优先；无官方 Go SDK 时使用官方 API 适配器

---

## 2. 后端实施步骤

### Step 1：扩展供应商枚举与工厂

- 更新 `server/internal/model/dns_provider.go`
  - 增加：`dns_com`、`dnsla`、`westcn_dns`、`baidu_cloud_dns`、`aws_route53`、`google_cloud_dns`
- 更新 `server/internal/service/provider_factory.go`
  - 增加凭据解析与 `BuildProviderClient` 分支

### Step 2：新增 provider adapter

在 `server/internal/service/` 新增或补齐：
- `route53.go`
- `google_cloud_dns.go`
- `baidu_cloud_dns.go`
- `dns_com.go`
- `dnsla.go`
- `westcn_dns.go`

统一实现 `DNSProviderClient`：
- `ListZones`
- `CreateRecord`
- `UpdateRecord`
- `DeleteRecord`
- `FindRecord`

### Step 3：迁移数据模型与执行器

- 新增模型与仓储：
  - `DomainDNSMigrationTask`
  - `DomainDNSMigrationItem`
- 新增迁移执行服务：
  - 创建任务（立即切换生效供应商）
  - 队列串行调度（同域名单运行）
  - 失败项重试（仅失败项）
  - 可选源端清理（仅平台托管记录）

### Step 4：写路径只读保护

- 在 DNS 写接口入口（Create/Update/Delete）检查域名迁移状态。
- `running` 时拒绝写操作并返回统一错误分类 `domain_migration_read_only`，HTTP 状态沿用现有项目逻辑 `409`。

### Step 5：统一错误语义与审计

- 将供应商错误映射到统一类别。
- 迁移创建、重试、覆盖、清理动作写入审计日志。
- 在 `dns_operation_events` 中记录失败分类与关键信息。

### Step 6：新增管理接口

- `POST /api/v1/admin/domains/{id}/migrations`
- `GET /api/v1/admin/domains/{id}/migrations`
- `GET /api/v1/admin/domains/{id}/migrations/{taskId}`
- `POST /api/v1/admin/domains/{id}/migrations/{taskId}/retry-failures`
- `POST /api/v1/admin/domains/{id}/migrations/{taskId}/cleanup-source`
- `GET /api/v1/admin/dns-providers/status`

`cleanup-source` 必须带强制确认参数（域名精确匹配 + 固定确认短语），避免误删。

---

## 3. 前端实施步骤

### Step 1：管理员供应商账号页

- 文件：`web/src/pages/admin/dns-provider-accounts.tsx`
- 补齐新增供应商选项、凭据字段渲染、状态展示文案。

### Step 2：域名管理页迁移能力

- 文件：`web/src/pages/admin/domains.tsx`
- 新增“发起迁移”入口、队列状态、失败项重试、源端清理按钮。
- 在列表中展示 `migration_state` 与只读标识。

### Step 3：用户记录页只读提示

- 文件：`web/src/pages/subdomain-detail.tsx`、`web/src/components/dns/*`
- 迁移期间禁用新增/编辑/删除按钮并显示状态提示。

### Step 4：API 与类型

- 文件：`web/src/lib/api.ts`、`web/src/types/*`
- 增加迁移任务相关请求与类型定义。

### Step 5：i18n

- 文件：`web/src/i18n/*.json`
- 补齐供应商名称、迁移状态、统一错误语义文案。

---

## 4. 手工验收（不跑编译/单测）

1. 管理员创建 10 家供应商账号并完成 zone 拉取。
2. 创建域名并绑定任一供应商；用户完成 A/AAAA/CNAME/TXT 的增删改查。
3. 对有现存记录的域名发起迁移：
- 任务创建后确认域名生效供应商已切换。
- 迁移 `running` 时用户写操作被拒绝、查询可用。
4. 构造失败项并执行“重试失败项”：
- 仅失败项被重试；冲突场景执行 upsert 覆盖并可审计。
5. 执行“源端清理”：
- 仅平台托管记录被清理，非托管记录不受影响。
6. 同域名连续提交两个迁移任务：
- 验证后提交任务进入队列并在前序终态后自动执行。
7. 查看 `dns-providers/status`：
- 能看到失败事件与健康状态聚合。

---

## 5. 实施边界

- 本期不实现 DnsDun。
- 本期不扩展 SRV/CAA/MX 等记录类型。
- 本期不新增独立任务调度服务，先在现有服务进程内实现可恢复 worker。
