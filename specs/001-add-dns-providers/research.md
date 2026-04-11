# Phase 0 Research: 域名多 DNS 供应商支持

## 决策 1：交付范围按 FR 一次性上线 10 家供应商，DnsDun 明确排除

### Decision

- 首发范围固定为：Cloudflare、DNSPod、阿里云 DNS、DNS.com、DNSLA、西部数码、华为云 DNS、百度智能云 DNS、Amazon Route 53、Google Cloud DNS。
- `dnsdun` 不出现在枚举、UI 选项、后端校验允许列表。
- 不采用“按供应商分批上线”或“部分只读能力上线”。

### Rationale

- 与 FR-010、FR-011 完全一致，避免需求边界漂移。
- 统一上线有利于前后端能力矩阵一次性定稿，减少二次迁移成本。

### Alternatives considered

- 分批交付：与 FR-010 冲突。
- 临时灰度只开放查询能力：会破坏“统一完整读写能力”目标。

---

## 决策 2：供应商接入采用“官方 SDK 优先；无 SDK 时官方 API 适配器”

### Decision

- 保持统一端口 `DNSProviderClient`（`ListZones/CreateRecord/UpdateRecord/DeleteRecord/FindRecord`）。
- 官方 SDK 路径：Cloudflare、DNSPod、阿里云、华为云、Route53、Google Cloud DNS、百度智能云。
- 官方 API 适配路径：DNS.com、DNSLA、西部数码。
- DNSLA 接入资料来源固定为公开文档，不依赖私有对接材料。

接入矩阵：

| 供应商 | 结论 | 实现路径 |
|---|---|---|
| Cloudflare | 官方 Go SDK 可用 | `cloudflare-go/v4` |
| DNSPod | 官方 Go SDK 可用 | `tencentcloud-sdk-go` |
| 阿里云 DNS | 官方 Go SDK 可用 | `alidns-20150109/v5` |
| 华为云 DNS | 官方 Go SDK 可用 | `huaweicloud-sdk-go-v3` |
| Amazon Route 53 | 官方 Go SDK 可用 | `aws-sdk-go-v2/service/route53` |
| Google Cloud DNS | 官方 Go SDK/官方 API Client 可用 | `google.golang.org/api/dns/v1` |
| 百度智能云 DNS | 官方 Go SDK 可用 | `baidubce/bce-sdk-go` |
| DNS.com | 可用官方 API 文档 | HTTP 适配器 |
| DNSLA | 采用官方公开 API 文档 | HTTP 适配器 |
| 西部数码 | 可用官方 API 文档 | HTTP 适配器 |

### Rationale

- SDK 方案可复用官方签名和错误结构，降低接入风险。
- API 适配器方案保证无 SDK 场景仍可纳入统一接口。

### Alternatives considered

- 全部自研 HTTP 客户端：签名细节重复实现多，风险和维护成本更高。
- 社区 SDK：可维护性与兼容性不可控。

---

## 决策 3：迁移能力采用独立“任务 + 任务项”模型，不复用 `dns_bulk_jobs`

### Decision

新增迁移专用模型：
- `domain_dns_migration_tasks`
- `domain_dns_migration_items`

不直接复用 `dns_bulk_jobs`，仅复用其“异步执行 + 可恢复 worker”实现经验。

### Rationale

- 迁移语义包含 source/target provider 快照、队列顺序、任务终态与重试策略，语义明显重于“批量删除”。
- 独立模型更易支撑 FR-013~FR-023 的状态可追踪要求。

### Alternatives considered

- 直接复用 `dns_bulk_jobs`：字段语义不匹配，后续扩展会产生大量补丁字段。

---

## 决策 4：迁移任务创建成功后立即切换域名生效供应商，任务按域名串行执行

### Decision

- 创建迁移任务成功时，立即更新 `domains.provider/provider_account_id/provider_zone_id` 到目标供应商（FR-016）。
- 同域名仅允许一个运行中任务，其余任务入队 `pending`，前序任务终态后自动启动（FR-021、FR-023）。
- 任务执行读取任务内 source/target 快照，不依赖 `domains` 当前值，避免后续排队任务覆盖上下文。

### Rationale

- 满足“即时切换”与“队列串行”两项要求并存。
- 快照机制避免运行时读到被后续任务改写的域名绑定。

### Alternatives considered

- 任务完成后才切换供应商：与 FR-016 冲突。
- 队列中任务按最新域名配置动态读取：会导致任务语义漂移，不可审计。

---

## 决策 5：迁移期间域名写操作只读；任务结束即解除只读

### Decision

- 当域名存在 `running` 迁移任务时：允许查询，拒绝新增/修改/删除 DNS 记录（FR-015）。
- 阻断点放在 DNS 写 handler 的统一入口（Create/Update/Delete），返回统一 machine-readable 错误类别。
- 任务终态（成功或部分失败）后立即解除只读并继续使用目标供应商（FR-020）。

### Rationale

- 防止迁移窗口内双写导致平台状态与目标供应商状态失配。
- 与已有统一写路径兼容，改动面可控。

### Alternatives considered

- 迁移期间允许写入并做变更日志回放：复杂度高，且一致性风险大。

---

## 决策 6：失败项重试按“仅失败项 + upsert 覆盖”执行

### Decision

- 手动重试只处理当前失败项集合（FR-018）。
- 若目标供应商存在同名同类型记录且内容不一致，采用 upsert 覆盖为平台期望值（FR-022）。
- 重试动作记录覆盖前后关键字段（旧值、新值、操作者、时间）。

### Rationale

- 直接落实澄清结论，避免重试扩大影响面。
- upsert 能最小化人工对齐成本，并保证最终一致。

### Alternatives considered

- 全量重跑：成本更高且可能重复覆盖已成功项。
- 冲突即失败不覆盖：会增加人工干预和工单量。

---

## 决策 7：源供应商记录默认不清理，提供显式管理员清理动作

### Decision

- 迁移流程默认不删除源端记录（FR-019）。
- 提供管理员触发的“源端清理”接口，仅清理平台托管记录集合。
- 清理过程必须写审计日志并保留失败项结果。

### Rationale

- 保守策略更安全，避免误删源端非托管记录。
- 显式操作可审计，便于运维追责与回溯。

### Alternatives considered

- 迁移成功后自动清理：误删风险高，不符合澄清结论。

---

## 决策 8：统一错误语义与重试分类，支撑跨供应商前端一致提示

### Decision

统一错误分类：
- `auth_failed`
- `permission_denied`
- `record_conflict`
- `record_not_found`
- `rate_limited`
- `provider_unavailable`
- `invalid_request`
- `domain_migration_read_only`
- `unknown`

### Rationale

- 满足 FR-006，支持前端统一提示与重试策略。
- 保证单供应商异常时可观测、可隔离（FR-008、FR-009）。

### Alternatives considered

- 透传厂商原始错误：前端需按供应商分支处理，体验和可维护性差。

---

## 决策 9：供应商状态可视化基于“账号状态 + 操作事件 + 迁移任务”聚合

### Decision

- 复用 `dns_operation_events` 作为调用事件事实源。
- 增加迁移任务维度（最近任务状态、失败项数量、排队长度）用于管理页可视化。
- 新增管理员状态总览接口返回 provider 聚合健康度。

### Rationale

- 满足 FR-007/FR-009 的可观测性目标。
- 在现有审计体系上增量实现，降低引入新系统风险。

### Alternatives considered

- 单独搭建监控子系统：超出当前特性范围。

---

## 决策 10：凭据与安全策略沿用加密存储，按供应商最小字段校验

### Decision

- `dns_provider_accounts.credentials` 持续使用加密后 JSON 存储。
- 每家供应商定义最小必填键集，创建/更新时强校验。
- API 返回统一使用脱敏摘要字段，禁止回传明文凭据。

### Rationale

- 与项目“安全优先”约束一致。
- 复用现有 `encryptRawCredentials/decryptRawCredentials` 机制，减少改造风险。

### Alternatives considered

- 放宽凭据校验、运行时再失败：会增加生产错误与运维成本。

---

## 参考资料（官方文档/官方仓库）

- Cloudflare Go SDK: https://developers.cloudflare.com/api/go/
- Cloudflare SDK repo: https://github.com/cloudflare/cloudflare-go
- 腾讯云 DNSPod 文档: https://cloud.tencent.com/document/product/1427
- Tencent Cloud Go SDK: https://github.com/TencentCloud/tencentcloud-sdk-go
- 阿里云 DNS OpenAPI: https://next.api.aliyun.com/
- AliDNS Go SDK: https://github.com/alibabacloud-go/alidns-20150109
- 华为云 DNS API: https://support.huaweicloud.com/api-dns/
- 华为云 Go SDK: https://github.com/huaweicloud/huaweicloud-sdk-go-v3
- Route 53 API Reference: https://docs.aws.amazon.com/Route53/latest/APIReference/Welcome.html
- AWS SDK for Go v2 Route53: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/route53
- Google Cloud DNS 文档: https://cloud.google.com/dns/docs
- Google DNS API Go Client: https://pkg.go.dev/google.golang.org/api/dns/v1
- 百度智能云 DNS 文档: https://cloud.baidu.com/doc/DNS/
- 百度 BCE Go SDK: https://github.com/baidubce/bce-sdk-go
- DNS.com API 文档: https://www.dns.com/supports/api/
- 西部数码 API 文档入口: https://www.west.cn/docs/
