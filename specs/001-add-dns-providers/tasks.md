# Tasks: 域名多 DNS 供应商支持

**Input**: Design documents from `/specs/001-add-dns-providers/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/, quickstart.md

**Tests**: 本特性未显式要求 TDD/自动化测试任务；以下任务以实现与手工验收为主。

**Organization**: 任务按用户故事分组，确保每个故事可独立实现和独立验收。

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 补齐跨阶段共用的配置与骨架文件。

- [X] T001 扩展迁移队列/worker 配置读取与环境变量说明在 `server/internal/config/config.go` 和 `.env.example`
- [X] T002 [P] 创建迁移领域后端骨架文件 `server/internal/model/domain_dns_migration.go`、`server/internal/repository/domain_migration.go`、`server/internal/service/domain_migration_service.go`、`server/internal/handler/domain_migration.go`
- [X] T003 [P] 创建迁移领域前端骨架与占位接口在 `web/src/types/index.ts`、`web/src/lib/api.ts`、`web/src/hooks/use-domain-migrations.ts`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 所有用户故事共享且阻塞后续开发的核心基础能力。

**⚠️ CRITICAL**: 本阶段完成前，不进入任何用户故事实现。

- [X] T004 扩展供应商枚举并明确排除 `dnsdun` 在 `server/internal/model/dns_provider.go`
- [X] T005 [P] 实现官方 SDK 适配器 `server/internal/service/route53.go`、`server/internal/service/google_cloud_dns.go`、`server/internal/service/baidu_cloud_dns.go`
- [X] T006 [P] 实现官方 API HTTP 适配器 `server/internal/service/dns_com.go`、`server/internal/service/dnsla.go`、`server/internal/service/westcn_dns.go`
- [X] T007 接入多供应商工厂分支与凭据解析在 `server/internal/service/provider_factory.go` 和 `server/internal/service/provider_helpers.go`
- [X] T008 建立统一错误语义映射与可重试判定在 `server/internal/service/provider_errors.go`、`server/internal/service/dns_operation_service.go`、`server/internal/handler/operation_result.go`
- [X] T009 注册新增模型迁移与关键索引约束在 `server/cmd/server/main.go`
- [X] T010 初始化迁移服务并完成路由层注入准备在 `server/internal/router/router.go`

**Checkpoint**: Foundation ready - 可以开始用户故事实现。

---

## Phase 3: User Story 1 - 管理员接入多供应商 (Priority: P1) 🎯 MVP

**Goal**: 管理员可新增、更新、删除、验证 10 家 DNS 供应商账号并看到可用状态。

**Independent Test**: 仅交付本阶段时，管理员可完成任一受支持供应商账号接入并得到可用/不可用反馈。

### Implementation for User Story 1

- [X] T011 [US1] 扩展供应商账号模型与视图字段（status/last_verified/last_error）在 `server/internal/model/models.go`
- [X] T012 [US1] 实现供应商账号状态读写与校验仓储方法在 `server/internal/repository/dns_provider_account.go`
- [X] T013 [US1] 在账号创建/更新/列表/zone 查询中接入凭据最小字段校验与连通性验证在 `server/internal/handler/dns_provider_account.go`
- [X] T014 [US1] 在域名创建与更新时阻断禁用/无效账号绑定在 `server/internal/handler/domain.go` 和 `server/internal/repository/domain.go`
- [X] T015 [P] [US1] 扩展管理端供应商账号 API 类型定义在 `web/src/types/index.ts` 和 `web/src/lib/api.ts`
- [X] T016 [US1] 更新供应商账号管理页支持 10 家供应商动态凭据表单与状态列在 `web/src/pages/admin/dns-provider-accounts.tsx`
- [X] T017 [P] [US1] 补齐供应商名称与账号状态多语言文案在 `web/src/i18n/zh.json`、`web/src/i18n/en.json`、`web/src/i18n/zh-Hant.json`、`web/src/i18n/es.json`、`web/src/i18n/ru.json`、`web/src/i18n/ja.json`

**Checkpoint**: US1 可独立验收并可作为 MVP 演示。

---

## Phase 4: User Story 2 - 用户按域名选择供应商管理记录 (Priority: P1)

**Goal**: 用户在绑定供应商的域名下完成统一 DNS 记录 CRUD，且支持后台异步迁移、失败项重试与源端清理。

**Independent Test**: 在至少接入 1 家非 Cloudflare 供应商时，用户可完成 A/AAAA/CNAME/TXT 全流程；迁移期间写操作被 409 拒绝、结束后恢复。

### Implementation for User Story 2

- [X] T018 [US2] 定义迁移任务/任务项模型与域名迁移态字段在 `server/internal/model/domain_dns_migration.go` 和 `server/internal/model/models.go`
- [X] T019 [US2] 实现迁移任务仓储（创建、串行抢占、详情分页、失败项筛选）在 `server/internal/repository/domain_migration.go`
- [X] T020 [US2] 实现迁移编排服务（创建即切换、队列串行执行）在 `server/internal/service/domain_migration_service.go`
- [X] T021 [US2] 实现失败项增量重试与冲突 upsert 覆盖审计在 `server/internal/service/domain_migration_service.go` 和 `server/internal/service/dns_operation_service.go`
- [X] T022 [US2] 实现源供应商清理与强制确认校验（`confirm_domain_name` + `confirm_phrase`）在 `server/internal/handler/domain_migration.go` 和 `server/internal/service/domain_migration_service.go`
- [X] T023 [US2] 在域名更新流程接入“有记录即创建异步迁移任务”逻辑在 `server/internal/handler/domain.go`
- [X] T024 [US2] 在 DNS 写接口增加迁移只读拦截并返回 `409` + `domain_migration_read_only` 在 `server/internal/handler/dns.go`
- [X] T025 [US2] 注册迁移接口路由（create/list/detail/retry/cleanup）在 `server/internal/router/router.go`
- [X] T026 [P] [US2] 扩展前端迁移任务类型与 API 方法在 `web/src/types/index.ts` 和 `web/src/lib/api.ts`
- [X] T027 [P] [US2] 实现迁移 React Query hooks 在 `web/src/hooks/use-domain-migrations.ts` 和 `web/src/hooks/use-dns-records.ts`
- [X] T028 [US2] 在域名管理页实现迁移发起、队列展示、失败项重试、cleanup 二次确认弹窗在 `web/src/pages/admin/domains.tsx`
- [X] T029 [US2] 在子域名详情和 DNS 组件实现迁移只读禁写与提示在 `web/src/pages/subdomain-detail.tsx`、`web/src/components/dns/record-form.tsx`、`web/src/components/dns/record-table.tsx`
- [X] T030 [P] [US2] 补齐迁移状态、只读错误、cleanup 确认多语言文案在 `web/src/i18n/zh.json`、`web/src/i18n/en.json`、`web/src/i18n/zh-Hant.json`、`web/src/i18n/es.json`、`web/src/i18n/ru.json`、`web/src/i18n/ja.json`

**Checkpoint**: US2 可独立验收（含迁移队列与只读策略）。

---

## Phase 5: User Story 3 - 运维可观测与故障隔离 (Priority: P2)

**Goal**: 管理员可查看供应商健康状态与最近失败事件，验证单供应商故障不会影响其他供应商域名操作。

**Independent Test**: 构造单供应商失败后，可在状态页定位失败分类/时间；其他供应商域名操作继续成功。

### Implementation for User Story 3

- [X] T031 [US3] 扩展 DNS 操作事件字段（error_category/retryable/latency_ms）并写入持久化在 `server/internal/model/dns_operation.go`、`server/internal/repository/dns_operation.go`、`server/internal/service/dns_operation_service.go`
- [X] T032 [US3] 实现供应商状态聚合查询（账号状态、最近失败、24h 失败数、迁移队列）在 `server/internal/repository/dns_provider_status.go`
- [X] T033 [US3] 新增状态总览接口 `GET /api/v1/admin/dns-providers/status` 在 `server/internal/handler/admin.go` 和 `server/internal/router/router.go`
- [X] T034 [US3] 对接状态总览前端类型与 API 在 `web/src/types/index.ts` 和 `web/src/lib/api.ts`
- [X] T035 [US3] 在管理员供应商账号页展示健康状态与最近失败事件在 `web/src/pages/admin/dns-provider-accounts.tsx`
- [X] T036 [P] [US3] 补齐错误分类与健康状态多语言文案在 `web/src/i18n/zh.json`、`web/src/i18n/en.json`、`web/src/i18n/zh-Hant.json`、`web/src/i18n/es.json`、`web/src/i18n/ru.json`、`web/src/i18n/ja.json`

**Checkpoint**: US3 可独立验收并支撑运维排障。

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 收尾、文档与跨故事一致性加固。

- [X] T037 [P] 更新产品说明与运维文档中的多供应商/迁移队列/清理确认指南在 `README.md`
- [X] T038 根据落地实现回写 quickstart 的手工验收步骤与注意事项在 `specs/001-add-dns-providers/quickstart.md`
- [X] T039 清理 Cloudflare-only 文案与命名，统一为 provider-neutral 在 `web/src/i18n/en.json`、`web/src/i18n/zh.json`、`server/internal/handler/dns.go`
- [X] T040 对齐计划文档与合同文档中的最终接口字段/状态定义在 `specs/001-add-dns-providers/plan.md` 和 `specs/001-add-dns-providers/contracts/dns-provider-api-contract.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: 无依赖，可立即开始。
- **Phase 2 (Foundational)**: 依赖 Phase 1，且阻塞所有用户故事。
- **Phase 3 (US1)**: 依赖 Phase 2。
- **Phase 4 (US2)**: 依赖 Phase 2，且建议在 US1 完成后开展联调验收。
- **Phase 5 (US3)**: 依赖 Phase 2，且依赖 US2 产出操作事件与迁移状态数据源。
- **Phase 6 (Polish)**: 依赖目标用户故事完成。

### User Story Dependencies

- **US1 (P1)**: 无用户故事前置依赖（仅依赖 Foundational）。
- **US2 (P1)**: 技术上可在 Foundational 后启动；独立验收建议依赖 US1 完成供应商账号接入链路。
- **US3 (P2)**: 依赖 US2 提供迁移与调用事件数据，建议在 US1+US2 后完成。

### Within Each User Story

- 先完成后端模型/仓储/服务，再完成 handler 与 router。
- 前端类型/API 先于页面交互改造。
- i18n 文案在页面功能稳定后统一收口。

---

## Parallel Opportunities

- Setup 阶段：`T002` 与 `T003` 可并行。
- Foundational 阶段：`T005` 与 `T006` 可并行；完成后再做 `T007`。
- US1 阶段：`T015` 与 `T017` 可并行。
- US2 阶段：`T026` 与 `T027` 可并行；完成后执行 `T028`/`T029`。
- US3 阶段：`T034` 与 `T036` 可并行。
- Polish 阶段：`T037` 可与 `T038` 并行。

---

## Parallel Example: User Story 1

```bash
Task: "T015 [US1] 扩展前端类型与 API（web/src/types/index.ts, web/src/lib/api.ts）"
Task: "T017 [US1] 补齐 US1 多语言文案（web/src/i18n/*.json）"
```

## Parallel Example: User Story 2

```bash
Task: "T026 [US2] 扩展迁移 API 与类型（web/src/types/index.ts, web/src/lib/api.ts）"
Task: "T027 [US2] 实现迁移 hooks（web/src/hooks/use-domain-migrations.ts, web/src/hooks/use-dns-records.ts）"
```

## Parallel Example: User Story 3

```bash
Task: "T034 [US3] 对接状态总览前端 API（web/src/types/index.ts, web/src/lib/api.ts）"
Task: "T036 [US3] 补齐 US3 多语言文案（web/src/i18n/*.json）"
```

---

## Implementation Strategy

### MVP First (建议)

1. 完成 Phase 1 + Phase 2。
2. 仅交付 Phase 3 (US1)。
3. 按 US1 Independent Test 做手工验收并演示。

### Incremental Delivery

1. 在 MVP 基础上交付 Phase 4 (US2) 并完成迁移链路验收。
2. 最后交付 Phase 5 (US3) 的可观测性与故障隔离可视化。
3. Phase 6 收尾文档与一致性修正。

### Parallel Team Strategy

1. 团队共同完成 Phase 1/2。
2. 分工并行：
- 开发 A：US1 管理端账号链路
- 开发 B：US2 迁移服务与用户写路径只读
- 开发 C：US3 状态聚合与可视化
3. 在每个故事 checkpoint 合并并做独立验收。
