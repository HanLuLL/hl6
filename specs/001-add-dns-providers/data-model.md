# Phase 1 Data Model: 域名多 DNS 供应商支持

## 1. DNSProvider（代码枚举实体）

说明：沿用代码枚举（`server/internal/model/dns_provider.go`），不新增独立表。

| 字段 | 类型 | 说明 | 约束 |
|---|---|---|---|
| `code` | string | 供应商唯一标识 | 全局唯一、小写蛇形 |
| `display_name` | string | 管理端展示名称 | 必填 |
| `adapter_mode` | enum | `official_sdk` / `official_api_http` | 必填 |
| `enabled` | bool | 是否允许绑定新域名 | 默认 `true` |
| `supported_record_types` | string[] | 可用记录类型 | 本期固定 `A/AAAA/CNAME/TXT` |

本期 code：
- `cloudflare`
- `dnspod`
- `aliyun_dns`
- `dns_com`
- `dnsla`
- `westcn_dns`
- `huawei_cloud_dns`
- `baidu_cloud_dns`
- `aws_route53`
- `google_cloud_dns`

`dnsdun` 不纳入本期枚举。

---

## 2. DNSProviderAccount（`dns_provider_accounts` 扩展）

| 字段 | 类型 | 说明 | 约束 |
|---|---|---|---|
| `id` | uint | 主键 | PK |
| `provider` | varchar(32) | 供应商 code | 必填，必须在支持枚举内 |
| `name` | varchar | 管理员可读名称 | 必填 |
| `credentials` | text/json | 加密后的凭据 | 必填，禁止明文回传 |
| `status` | varchar(16) | `active/degraded/disabled` | 默认 `active` |
| `last_verified_at` | timestamptz | 最近校验时间 | 可空 |
| `last_error_category` | varchar(32) | 最近失败分类 | 可空 |
| `last_error_message` | text | 最近失败摘要（脱敏） | 可空 |
| `created_at` | timestamptz | 创建时间 | 自动 |
| `updated_at` | timestamptz | 更新时间 | 自动 |

校验规则：
- 创建/更新时按 provider 校验最小字段集。
- 若账号被域名引用，不允许硬删除（返回冲突）。

---

## 3. Domain（`domains` 扩展为“生效绑定 + 迁移态”）

| 字段 | 类型 | 说明 | 约束 |
|---|---|---|---|
| `id` | uint | 主键 | PK |
| `name` | varchar | 域名 | 全局唯一 |
| `provider` | varchar(32) | 当前生效供应商 | 必填 |
| `provider_account_id` | uint | 当前生效账号 | 必填，FK -> dns_provider_accounts.id |
| `provider_zone_id` | varchar | 当前生效 zone | 必填 |
| `migration_state` | varchar(24) | `idle/running/partial_failed/queued` | 默认 `idle` |
| `migration_read_only` | bool | 迁移期间写保护 | 默认 `false` |
| `last_migration_task_id` | uint | 最近迁移任务 | 可空 |
| `is_active` | bool | 域名开关 | 默认 `true` |
| `description` | text | 描述 | 可空 |
| `created_at` | timestamptz | 创建时间 | 自动 |
| `updated_at` | timestamptz | 更新时间 | 自动 |

业务规则：
- 一个域名任一时刻仅一个生效供应商绑定。
- 迁移任务创建成功立即更新 `provider/provider_account_id/provider_zone_id` 到目标值。

---

## 4. DomainDNSMigrationTask（新增：`domain_dns_migration_tasks`）

| 字段 | 类型 | 说明 | 约束 |
|---|---|---|---|
| `id` | uint | 主键 | PK |
| `domain_id` | uint | 域名 ID | 必填，FK -> domains.id |
| `status` | varchar(24) | `pending/running/succeeded/partial_failed/failed/cancelled` | 必填，索引 |
| `queue_seq` | bigint | 同域名队列序号 | `(domain_id, queue_seq)` 唯一 |
| `triggered_by` | uint | 发起管理员 | 必填，FK -> users.id |
| `source_provider` | varchar(32) | 源供应商快照 | 必填 |
| `source_account_id` | uint | 源账号快照 | 必填 |
| `source_zone_id` | varchar(191) | 源 zone 快照 | 必填 |
| `target_provider` | varchar(32) | 目标供应商快照 | 必填 |
| `target_account_id` | uint | 目标账号快照 | 必填 |
| `target_zone_id` | varchar(191) | 目标 zone 快照 | 必填 |
| `total_items` | int | 计划迁移条目数 | 默认 0 |
| `succeeded_items` | int | 成功条目数 | 默认 0 |
| `failed_items` | int | 失败条目数 | 默认 0 |
| `retried_items` | int | 已重试条目数 | 默认 0 |
| `last_error_category` | varchar(32) | 任务级最近失败分类 | 可空 |
| `last_error_message` | text | 任务级最近失败信息 | 可空 |
| `started_at` | timestamptz | 启动时间 | 可空 |
| `finished_at` | timestamptz | 完成时间 | 可空 |
| `created_at` | timestamptz | 创建时间 | 自动 |
| `updated_at` | timestamptz | 更新时间 | 自动 |

关键索引：
- `(domain_id, status)`：快速定位运行中和排队任务。
- `status`：后台 worker 抢占待执行任务。

关键约束：
- 同域名最多一个 `running`（可通过部分唯一索引或事务锁保证）。
- 前序任务到达任一终态后，后续 `pending` 自动转 `running`。

---

## 5. DomainDNSMigrationItem（新增：`domain_dns_migration_items`）

| 字段 | 类型 | 说明 | 约束 |
|---|---|---|---|
| `id` | uint | 主键 | PK |
| `task_id` | uint | 所属迁移任务 | 必填，FK -> domain_dns_migration_tasks.id |
| `dns_record_id` | uint | 平台记录 ID | 必填，FK -> dns_records.id |
| `record_type` | varchar(16) | 记录类型 | 仅 `A/AAAA/CNAME/TXT` |
| `name` | varchar(255) | 记录名 | 必填 |
| `content` | text | 记录值快照 | 必填 |
| `ttl` | int | TTL 快照 | 必填 |
| `proxied` | bool | 代理快照 | 必填 |
| `source_provider_record_id` | varchar(191) | 源端记录 ID | 可空 |
| `target_provider_record_id` | varchar(191) | 目标端记录 ID | 可空 |
| `status` | varchar(24) | `pending/running/succeeded/failed/skipped` | 必填，索引 |
| `attempts` | int | 已尝试次数 | 默认 0 |
| `last_error_category` | varchar(32) | 最近错误分类 | 可空 |
| `last_error_message` | text | 最近错误消息 | 可空 |
| `conflict_overwritten` | bool | 是否发生 upsert 覆盖 | 默认 `false` |
| `overwrite_before` | jsonb | 覆盖前目标记录摘要 | 可空 |
| `overwrite_after` | jsonb | 覆盖后目标记录摘要 | 可空 |
| `finished_at` | timestamptz | 完成时间 | 可空 |
| `created_at` | timestamptz | 创建时间 | 自动 |
| `updated_at` | timestamptz | 更新时间 | 自动 |

关键约束：
- `retry-failures` 仅选择 `status=failed` 项。
- 成功项不可被重跑覆盖。

---

## 6. DNSRecord（`dns_records` 迁移相关约束）

| 字段 | 类型 | 说明 | 约束 |
|---|---|---|---|
| `id` | uint | 主键 | PK |
| `subdomain_id` | uint | 所属子域名 | 必填 |
| `type` | varchar(16) | 记录类型 | 本期仅 `A/AAAA/CNAME/TXT` |
| `name` | varchar(255) | 记录名 | 必填 |
| `content` | text | 记录值 | 必填 |
| `ttl` | int | TTL | 供应商归一化后存储 |
| `proxied` | bool | 代理标记 | 非支持供应商固定 false |
| `provider_record_id` | varchar(191) | 当前生效供应商记录 ID | 可空（未写入成功时） |
| `created_at` | timestamptz | 创建时间 | 自动 |
| `updated_at` | timestamptz | 更新时间 | 自动 |

迁移范围：仅处理平台数据库已托管记录（FR-017）。

---

## 7. DNSOperationEvent / AuditLog（沿用并增强）

- `dns_operation_events`：记录供应商调用步骤与失败分类。
- `audit_logs`：记录管理员发起迁移、失败项重试、源端清理等动作。

建议增强字段：
- `error_category`（标准分类）
- `retryable`（是否可重试）
- `latency_ms`（调用耗时）

---

## 8. 关系与状态流转

关系：
- `DNSProviderAccount (1) -> (N) Domain`
- `Domain (1) -> (N) DomainDNSMigrationTask`
- `DomainDNSMigrationTask (1) -> (N) DomainDNSMigrationItem`
- `Domain (1) -> (N) Subdomain -> (N) DNSRecord`

迁移任务状态流转：
- `pending -> running -> succeeded`
- `pending -> running -> partial_failed`
- `pending -> running -> failed`
- `failed/partial_failed --(retry-failures)--> running`

域名写保护状态：
- 创建任务后：`migration_read_only=true`
- 任务终态后：`migration_read_only=false`

---

## 9. 一致性与并发控制

- 创建迁移任务时对域名行加锁（`SELECT ... FOR UPDATE`），保证队列序号单调递增。
- worker 启动任务时做“同域名单运行”校验，避免并发执行。
- 迁移任务执行使用 source/target 快照，避免被后续域名绑定更新污染。
- 写入 DNS 记录接口在事务前检查 `migration_read_only`，统一拒绝写操作。
