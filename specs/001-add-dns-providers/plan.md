# Implementation Plan: 域名多 DNS 供应商支持

**Branch**: `001-add-dns-providers` | **Date**: 2026-04-11 | **Spec**: [/Users/jacup/Desktop/projects/个人开发/hl6/specs/001-add-dns-providers/spec.md](/Users/jacup/Desktop/projects/个人开发/hl6/specs/001-add-dns-providers/spec.md)
**Input**: Feature specification from `/specs/001-add-dns-providers/spec.md`

## Summary

在现有 Cloudflare 单主路径基础上扩展多供应商统一 DNS 执行层，首发一次性交付 Cloudflare、DNSPod、阿里云 DNS、DNS.com、DNSLA、西部数码、华为云 DNS、百度智能云 DNS、Amazon Route 53、Google Cloud DNS（明确排除 DnsDun）。

核心技术方案：
- 后端维持统一 `DNSProviderClient` 端口，按供应商注入官方 SDK 或官方 API 适配器。
- 域名跨供应商切换采用后台异步迁移任务，支持同域名串行队列、失败项增量重试、可选源端清理。
- 迁移任务创建后立即切换域名生效供应商；迁移运行期间域名 DNS 写操作只读，查询仍可用。
- 统一错误语义、审计事件和状态聚合接口，保证故障隔离与可观测性。

## Technical Context

**Language/Version**: Go 1.25.5（后端），TypeScript 5.9 + React 19（前端）
**Primary Dependencies**: Gin、GORM、PostgreSQL driver、cloudflare-go/v4、tencentcloud-sdk-go（dnspod）、alidns-20150109/v5、huaweicloud-sdk-go-v3、baidubce/bce-sdk-go、AWS SDK for Go v2（Route53）、google.golang.org/api/dns/v1、TanStack React Query、i18next
**Storage**: PostgreSQL 16
**Testing**: 现有 Go/前端工具链 + 手工验收路径；按项目约束本阶段不新增单元测试、不跑编译
**Target Platform**: Linux 服务器（Go API）+ 现代桌面/移动浏览器（React SPA）
**Project Type**: web（`server` + `web`）
**Performance Goals**: 管理员接入任一供应商 <= 10 分钟；90% 常规 DNS 变更 <= 1 分钟返回明确结果；迁移任务创建接口同步响应 <= 3 秒
**Constraints**: 安全优先（凭据加密/脱敏/最小暴露）；仅支持 A/AAAA/CNAME/TXT 一致能力；迁移期间域名写操作只读且沿用现有 `409` 返回；同域名迁移任务串行；`dnsdun` 不在本期范围
**Scale/Scope**: 10 家供应商统一交付；每域名单运行任务 + 任意长度队列；迁移仅处理平台托管 `dns_records`

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- 当前 `/Users/jacup/Desktop/projects/个人开发/hl6/.specify/memory/constitution.md` 仍为占位模板（`[PRINCIPLE_*]`），未定义可执行硬性条款。
- 预检查结论：**PASS（无可执行条款可违反）**。
- 治理风险记录：宪章未固化会导致后续任务缺少自动 gate 依据；本计划以项目既有工程约束（安全优先、官方文档优先、统一响应格式）作为替代基线。
- Phase 1 设计后复检结论：**PASS（设计未引入与既有工程约束冲突的新增风险）**。

## Project Structure

### Documentation (this feature)

```text
specs/001-add-dns-providers/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── dns-provider-api-contract.md
└── tasks.md
```

### Source Code (repository root)

```text
server/
├── cmd/server/
├── internal/
│   ├── config/
│   ├── handler/
│   ├── middleware/
│   ├── model/
│   ├── repository/
│   ├── router/
│   └── service/
└── pkg/

web/
├── src/
│   ├── components/
│   ├── hooks/
│   ├── i18n/
│   ├── lib/
│   ├── pages/
│   └── types/
└── public/
```

**Structure Decision**: 采用既有前后端分离单仓结构，在 `server/internal` 扩展供应商适配与迁移任务，在 `web/src/pages|hooks|types|i18n` 扩展管理与状态展示；不新增独立服务。

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | N/A | N/A |
