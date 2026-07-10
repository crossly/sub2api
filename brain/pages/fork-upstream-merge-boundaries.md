---
id: fork-upstream-merge-boundaries
title: "按上游版本演进并隔离维护本地 overlay"
category: decision
status: active
tags: [fork, upstream, merge, maintenance]
created: "2026-07-10T20:50:26"
updated: "2026-07-10T20:50:26"
---

## compiled_truth

## 决策

本仓库是 `Wei-Shaw/sub2api` 的维护型 fork。主线通过合并上游 release/tag 获取主体架构和功能，本地差异按独立 overlay 保留，不把整个 fork 当作脱离上游的长期分叉。

截至 2026-07-10，当前代码基线是上游 `v0.1.150` 加本地 `v0.1.150.1`。后续合并必须从新上游版本的实际架构重新审视 hook 点，尤其不能把 `v0.1.126` 的 429 实现直接搬到新版本。

## 本地长期边界

- OpenAI 429 超限探测与 failover 由 [[openai-429-over-limit-routing]] 维护。
- Coolify 部署和自有 GHCR 发布由 [[coolify-ghcr-release-contract]] 维护。
- 品牌、主题、首页和响应式 UI 由 [[local-branding-ui-overlay]] 维护。
- OpenAI OAuth 账号导入与测试前刷新由 [[openai-oauth-onboarding-compat]] 维护。
- `deploy/docker-compose*.yml` 保持上游版本；Coolify 特有配置只放在仓库根部 `docker-compose.coolify.yml`。
- 上游拥有网关主体、provider adapter、数据库模型、支付、监控和邮件模块的总体演进方向。

## 合并原则

1. 先确认新上游 tag 的实际运行路径和合并提交第二父节点。
2. 按功能桶检查本地 delta，不用一个旧补丁覆盖新的 scheduler、handler 或 transport。
3. 先完成上游合并，再逐个恢复 Coolify、品牌、OAuth 和 429 overlay；429 最后处理并做端到端回归。
4. 对 scheduler membership、request-time eligibility、SSE/WebSocket 错误路径、重试 exclusion set 和设置 API 做专项审计。
5. 合并后必须跑后端 unit/integration、`golangci-lint`、前端 typecheck/critical vitest 和 release workflow。

## 理由

上游在 0.1.126 之后多次重构了 scheduler snapshot、advanced scheduler、OpenAI transport、邮件发送和大型 service 文件。按行为契约重新落地比沿用旧内部实现更能避免隐蔽回归，也能让本地差异继续保持可审计。

## 当前待确认

当前源码仍保留本地紫色 legacy email fallback 模板，但上游已经引入模块化 `NotificationEmailService`，且历史需求倾向于不再本地维护邮件模板。下一次上游合并前应明确该 fallback 是继续保留、删除，还是完全交给上游模块；在确认前不要把它视为稳定的 fork 所有权。


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
