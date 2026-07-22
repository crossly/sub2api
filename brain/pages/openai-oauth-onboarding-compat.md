---
id: openai-oauth-onboarding-compat
title: "保留 OpenAI OAuth 本地导入与测试前 token 刷新"
category: decision
status: archived
tags: [openai, oauth, import, account-test]
created: "2026-07-10T20:50:27"
updated: "2026-07-22T10:34:24"
---

## compiled_truth

## 退役决定

自合并上游 `v0.1.162` 起，本 fork 不再把 OpenAI OAuth 导入转换器、账号测试前 token refresh 或相关 Wire/test 改动作为长期维护 overlay。账号导入、OAuth refresh、连接测试和 Codex onboarding 语义全部采用上游目标版本。

## 合并要求

- 不在后续上游合并中重放 `AccountTestService` 的本地 refresh hook。
- 本地 `codex_txt_to_sub2api` 转换命令若不属于上游版本，应从运行时代码树移除；私有数据转换需求可在仓库外单独处理。
- 导入 schema、required fields 和 token metadata 以上游实现为准。
- 未来如需新增本地 onboarding 能力，必须形成新的明确决策，不能依赖本页历史实现。

## 历史说明

本页此前记录了 `v0.1.146` 之后保留的本地 OAuth onboarding 兼容层。该实现仅作为 Git 历史存在，不再构成当前维护边界。


## timeline

- time: 2026-07-10T20:50:27
  kind: decision
  summary: "Created this page: 保留 OpenAI OAuth 本地导入与测试前 token 刷新"
  source: "ba048f556; account_test_service.go; codex_txt_to_sub2api command"
  affects: [openai-oauth-onboarding-compat]

- time: 2026-07-10T20:50:27
  kind: decision
  summary: "记录 OpenAI OAuth 账号导入和连接测试兼容层的本地所有权"
  source: "ba048f556; account_test_service.go; codex_txt_to_sub2api command"
  affects: [openai-oauth-onboarding-compat]

- time: 2026-07-22T10:34:24
  kind: decision
  summary: "将 OAuth onboarding 兼容层移出长期 overlay，v0.1.162 起采用上游实现"
  source: "用户决策 2026-07-22；仅保留 Coolify/GHCR 与品牌 UI"
  affects: [openai-oauth-onboarding-compat]

- time: 2026-07-22T10:34:24
  kind: reversal
  summary: "OAuth onboarding 本地 overlay 已退役；v0.1.162 起采用上游实现"
  source: brain archive-page
  affects: [openai-oauth-onboarding-compat]
