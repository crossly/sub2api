---
id: fork-upstream-merge-boundaries
title: "按上游版本演进并隔离维护本地 overlay"
category: decision
status: active
tags: [fork, upstream, merge, maintenance]
created: "2026-07-10T20:50:26"
updated: "2026-07-22T11:34:10"
---

## compiled_truth

## 决策

本仓库是 `Wei-Shaw/sub2api` 的维护型 fork。当前代码基线为上游 `v0.1.162`（`27f094e09`），加上游 tag 后安全修复 `ef3c770d9`（Axios）与 `c5971a6fc`（`golang.org/x/text`），以及少量本地 overlay。主线通过合并上游 release/tag 获取架构、安全修复和业务功能，不把 fork 演变成独立产品分支。

应用层长期保留的本地 overlay 仅有两类：

- Coolify 部署与自有 GHCR 发布，见 [[coolify-ghcr-release-contract]]。
- OINANCE 品牌、主题、首页和响应式 UI，见 [[local-branding-ui-overlay]]。

Project Brain 文件是维护知识，不属于运行时功能 overlay。tag 后采用的两个安全提交属于上游热修，不是新的本地功能 overlay；后续目标 tag 包含这些提交后，应自然收敛回上游基线。

## 上游所有权

除上述两类外，后端和前端功能均以上游目标版本为准，包括：

- OpenAI 429、账号调度、sticky、failover、runtime block 和设置界面。
- OpenAI OAuth、账号导入、连接测试、token refresh 和 Codex 兼容。
- provider adapter、HTTP/SSE/WebSocket transport、模型能力和错误映射。
- 数据库模型、计费、支付、监控、安全审计、异步图片、邮件和通知模块。
- `Dockerfile`、`deploy/docker-compose*.yml` 及上游标准部署文件。
- Go 与前端依赖安全修复；发现阻断 Security Scan 的上游已修漏洞时，可以在下一个 release tag 前采用对应上游提交。

历史本地 429、OAuth converter/test-refresh 和 legacy email overlay 已退役，不在后续上游合并中重放。

## 当前允许的代码差异

相对上游 `v0.1.162` tag，运行时代码差异只应出现在：

- 临时采用的上游安全热修：`backend/go.mod`、`backend/go.sum`、`frontend/package.json`、`frontend/pnpm-lock.yaml`。
- 根目录 `docker-compose.coolify.yml`。
- 品牌 logo、Tailwind token、landing locale、首页、认证背景、看板图表和响应式 shell/test。
- `.gitignore` 中允许版本控制 Project Brain 的规则。

`BRAIN.md`、`AGENTS.md` 和 `brain/` 是项目维护资料。任何其他运行时代码差异都需要明确的新决策。

## 合并原则

1. 获取并验证目标上游 tag 的 commit，确认与当前分支的 merge-base。
2. 先合并目标 tag；业务代码冲突默认采用上游版本。
3. 只对 Coolify/GHCR 与品牌 UI 两个功能桶恢复和审阅本地差异。
4. 合并后反向比较目标 tag，确认 `backend/`、`Dockerfile` 和 `deploy/` 与上游一致；已明确采用的上游 tag 后安全热修单独列出。
5. 搜索并删除上游不存在的历史 overlay 文件，不能只处理文本冲突。
6. 后端 unit/integration、`golangci-lint`、前端 typecheck/Vitest、compose 解析和 Security Scan 通过后，才发布 fork tag。
7. 新增任何第三类长期 overlay 必须先形成明确决策，不能因历史 commit 自动延续。

## 理由

上游架构持续快速演进。把调度、OAuth、邮件等功能维持为本地补丁会扩大冲突面，并容易让旧行为覆盖上游的新安全与兼容修复。将运行时功能交还上游，只保留部署和视觉差异，可以显著降低升级成本并让行为更可预测。安全热修直接采用上游提交，可在不形成长期 fork 差异的前提下保持 release 门禁可用。


## timeline

- time: 2026-07-10T20:50:26
  kind: decision
  summary: "Created this page: 按上游版本演进并隔离维护本地 overlay"
  source: "git first-parent history; upstream v0.1.150 merge; docs/plans"
  affects: [fork-upstream-merge-boundaries]

- time: 2026-07-10T20:50:26
  kind: decision
  summary: "从当前源码、上游合并历史和维护文档提炼 fork 的长期维护边界"
  source: "git first-parent history; upstream v0.1.150 merge; docs/plans"
  affects: [fork-upstream-merge-boundaries]

- time: 2026-07-22T10:34:24
  kind: decision
  summary: "将长期本地 overlay 收敛为 Coolify/GHCR 与品牌 UI，429、OAuth 等功能完全跟随上游"
  source: "用户决策 2026-07-22；upstream v0.1.162 升级"
  affects: [fork-upstream-merge-boundaries]

- time: 2026-07-22T10:41:19
  kind: decision
  summary: "记录 v0.1.162 基线、允许的最终 delta 与反向审计规则"
  source: "upstream v0.1.162 commit 27f094e09；merge tree audit 2026-07-22"
  affects: [fork-upstream-merge-boundaries]

- time: 2026-07-22T10:50:28
  kind: evidence
  summary: "v0.1.162 合并后 backend、Dockerfile、deploy 与上游一致，允许 delta 仅为 Brain、Coolify 和品牌 UI"
  source: "git diff against 27f094e09；local CI-equivalent validation 2026-07-22"
  affects: [fork-upstream-merge-boundaries, coolify-ghcr-release-contract, local-branding-ui-overlay]

- time: 2026-07-22T11:34:10
  kind: decision
  summary: "在 v0.1.162 基线上纳入上游 Axios 与 x/text 安全热修，恢复 Security Scan 全绿"
  source: "upstream ef3c770d9、c5971a6fc；Security Scan 29888390838；CI 29888390842"
  affects: [fork-upstream-merge-boundaries]
