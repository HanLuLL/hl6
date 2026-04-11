# DNS Provider API Contract (Phase 1)

## 1. 范围

定义本特性对外 HTTP 合同，覆盖：
- 管理员维护 DNS 供应商账号
- 域名绑定供应商账号
- 域名跨供应商迁移任务（创建、队列、重试、清理）
- 用户 DNS 记录统一 CRUD 与迁移只读约束
- 供应商状态总览与失败事件可视化

统一响应格式：`ApiResponse{code,message,data}`。

---

## 2. Provider Code 枚举

```text
cloudflare
dnspod
aliyun_dns
dns_com
dnsla
westcn_dns
huawei_cloud_dns
baidu_cloud_dns
aws_route53
google_cloud_dns
```

约束：
- `dnsdun` 不得在本期接口中作为合法值。

---

## 3. 管理员供应商账号接口

### 3.1 列出供应商账号

- `GET /api/v1/admin/dns-accounts`
- Auth: admin

`data[]` 字段：
- `id: number`
- `provider: string`
- `name: string`
- `credential_hint: string`
- `status: "active" | "degraded" | "disabled"`
- `last_verified_at: string | null`
- `last_error_category: string | null`
- `last_error_message: string | null`
- `created_at: string`
- `updated_at: string`

### 3.2 创建供应商账号

- `POST /api/v1/admin/dns-accounts`
- Auth: admin
- Header: `Idempotency-Key` required

Request:

```json
{
  "provider": "aws_route53",
  "name": "prod-route53",
  "credentials": {
    "access_key_id": "***",
    "access_key_secret": "***",
    "region": "us-east-1"
  }
}
```

Validation:
- `provider` 必须为支持枚举。
- `credentials` 必须满足最小字段集。

### 3.3 更新供应商账号

- `PUT /api/v1/admin/dns-accounts/{id}`
- Auth: admin
- Header: `Idempotency-Key` required

Rules:
- 修改 provider 时必须提交新 provider 的完整可用凭据。
- 禁止返回明文凭据。

### 3.4 删除供应商账号

- `DELETE /api/v1/admin/dns-accounts/{id}`
- Auth: admin
- Header: `Idempotency-Key` required

Rules:
- 若账号仍有关联域名，返回 `409`。

### 3.5 获取可绑定 Zone 列表

- `GET /api/v1/admin/dns-accounts/{id}/zones`
- Auth: admin

`data[]` 字段：
- `id: string`
- `name: string`
- `status: string`

---

## 4. 域名绑定接口（增强）

### 4.1 创建域名

- `POST /api/v1/admin/domains`
- Auth: admin
- Header: `Idempotency-Key` required

关键字段：
- `provider_account_id: number` required
- `provider_zone_id: string` required

### 4.2 更新域名

- `PUT /api/v1/admin/domains/{id}`
- Auth: admin
- Header: `Idempotency-Key` required

允许更新：
- `provider_account_id`
- `provider_zone_id`
- `is_active`
- `description`

新增响应增强字段（用于迁移状态提示）：
- `migration_state: "idle" | "running" | "partial_failed" | "queued"`
- `migration_read_only: boolean`
- `running_migration_task_id: number | null`

---

## 5. 域名迁移任务接口（新增）

### 5.1 创建迁移任务

- `POST /api/v1/admin/domains/{id}/migrations`
- Auth: admin
- Header: `Idempotency-Key` required

Request:

```json
{
  "target_provider_account_id": 12,
  "target_provider_zone_id": "Z089XXXX",
  "reason": "switch to route53"
}
```

Behavior:
- 创建任务后立即切换域名生效供应商到目标配置。
- 若同域名已有 `running` 任务，新任务入队 `pending`。
- 同域名任一时刻仅一个 `running`。

Response `data`:
- `task_id: number`
- `status: "pending" | "running"`
- `queue_position: number`
- `domain_migration_state`

### 5.2 列出域名迁移任务

- `GET /api/v1/admin/domains/{id}/migrations?status=&page=&per_page=`
- Auth: admin

`data[]` 字段：
- `id`
- `status`
- `queue_seq`
- `source_provider`
- `target_provider`
- `total_items`
- `succeeded_items`
- `failed_items`
- `started_at`
- `finished_at`

### 5.3 获取迁移任务详情

- `GET /api/v1/admin/domains/{id}/migrations/{taskId}`
- Auth: admin

`data`:
- `task`（任务元信息）
- `items[]`（可分页）
  - `dns_record_id`
  - `status`
  - `attempts`
  - `last_error_category`
  - `last_error_message`
  - `conflict_overwritten`

### 5.4 重试失败项

- `POST /api/v1/admin/domains/{id}/migrations/{taskId}/retry-failures`
- Auth: admin
- Header: `Idempotency-Key` required

Behavior:
- 仅重试 `failed` 项。
- 若目标同名同类型冲突，执行 upsert 覆盖并记录前后差异。

Response:
- `retried_items`
- `remaining_failed_items`
- `status`

### 5.5 可选清理源供应商记录

- `POST /api/v1/admin/domains/{id}/migrations/{taskId}/cleanup-source`
- Auth: admin
- Header: `Idempotency-Key` required

Request:

```json
{
  "confirm_domain_name": "example.com",
  "confirm_phrase": "CLEANUP"
}
```

Behavior:
- 仅清理平台托管记录。
- 默认不开启自动清理，必须显式调用。
- 强制二次确认：`confirm_domain_name` 必须与目标域名精确匹配，且 `confirm_phrase` 必须为固定值 `CLEANUP`。
- 任一确认字段不匹配时返回 `400 invalid_request`。

Response:
- `cleanup_total`
- `cleanup_succeeded`
- `cleanup_failed`

---

## 6. 用户 DNS 记录接口（路径不变，增加迁移只读约束）

### 6.1 列表
- `GET /api/v1/subdomains/{id}/records`

### 6.2 新增
- `POST /api/v1/subdomains/{id}/records`

### 6.3 修改
- `PUT /api/v1/subdomains/{id}/records/{recordId}`

### 6.4 删除
- `DELETE /api/v1/subdomains/{id}/records/{recordId}`

统一行为：
- 自动按域名生效供应商执行。
- 仅支持 `A/AAAA/CNAME/TXT`。
- 当域名 `migration_read_only=true` 时，写操作返回只读错误并拒绝执行。

HTTP 状态：沿用项目现有逻辑返回 `409`。

---

## 7. 供应商状态与故障总览

### 7.1 供应商状态总览

- `GET /api/v1/admin/dns-providers/status`
- Auth: admin

Response `data[]`:
- `provider`
- `accounts_total`
- `accounts_active`
- `last_verified_at`
- `last_failure_at`
- `last_failure_category`
- `failure_count_24h`
- `migration_queue_size`
- `health` (`healthy/degraded/unhealthy`)

---

## 8. 统一错误语义 Contract

接口返回应含 machine-readable 分类（建议 `data.error_category`）：

- `auth_failed`
- `permission_denied`
- `record_conflict`
- `record_not_found`
- `rate_limited`
- `provider_unavailable`
- `invalid_request`
- `domain_migration_read_only`
- `unknown`

重试建议：
- 可重试：`rate_limited`、`provider_unavailable`
- 不可重试：`auth_failed`、`permission_denied`、`invalid_request`、`record_conflict`
- 条件重试：`domain_migration_read_only`（等待任务终态后重试）

---

## 9. 凭据字段最小集 Contract

| provider | required keys |
|---|---|
| cloudflare | api_token |
| dnspod | secret_id, secret_key |
| aliyun_dns | access_key_id, access_key_secret |
| huawei_cloud_dns | ak, sk |
| aws_route53 | access_key_id, access_key_secret |
| google_cloud_dns | service_account_json |
| baidu_cloud_dns | access_key, secret_key |
| dns_com | 按 DNS.com 官方 API 签名字段 |
| dnsla | 按 DNSLA 官方 API 签名字段 |
| westcn_dns | 按西部数码官方 API 签名字段 |
