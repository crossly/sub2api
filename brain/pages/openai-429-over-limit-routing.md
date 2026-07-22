---
id: openai-429-over-limit-routing
title: "OpenAI 429 账号按间隔探测并在单次请求内回退"
category: decision
status: archived
tags: [openai, 429, scheduler, failover]
created: "2026-07-10T20:50:26"
updated: "2026-07-22T10:34:24"
---

## compiled_truth

## 退役决定

自合并上游 `v0.1.162` 起，本 fork 不再维护 OpenAI 429 超限探测、调度优先级或 fallback 的本地实现。当前 429 行为、设置项、账号状态持久化、HTTP/SSE/WebSocket 错误处理和账号切换语义全部以上游目标版本为准。

## 合并要求

- 不重放历史 `openai_over_limit_*` 策略、DTO、设置 UI、scheduler bucket、runtime block 或专项测试补丁。
- 429 相关文件发生冲突时采用上游版本。
- 对上游未包含的本地 429 文件执行删除，避免 merge 因“上游未触碰文件”而错误保留旧逻辑。
- 不再承诺旧版“定期探测超限账号并回退”的本地行为；上游实际行为是唯一事实源。
- 未来若重新提出 fork 专属 429 行为，必须基于当时上游架构建立新的决策与测试，不能复活本页旧契约。

## 历史说明

本页此前记录了基于 `v0.1.150` 架构的本地 429 overlay。该实现保留在 Git 历史中，仅用于解释过去版本，不再代表当前或未来版本。


## timeline

- time: 2026-07-10T20:50:26
  kind: decision
  summary: "Created this page: OpenAI 429 账号按间隔探测并在单次请求内回退"
  source: "v0.1.150 runtime code; e8c36962f; over-limit tests and plans"
  affects: [openai-429-over-limit-routing]

- time: 2026-07-10T20:50:27
  kind: decision
  summary: "记录 v0.1.150 第一性原理实现的 429 探测、优先级和回退契约"
  source: "v0.1.150 runtime code; e8c36962f; over-limit tests and plans"
  affects: [openai-429-over-limit-routing]

- time: 2026-07-22T10:34:24
  kind: decision
  summary: "撤销本地 429 行为契约，v0.1.162 起完全采用上游实现"
  source: "用户决策 2026-07-22；upstream v0.1.162 升级"
  affects: [openai-429-over-limit-routing]

- time: 2026-07-22T10:34:24
  kind: reversal
  summary: "本地 429 overlay 已退役；v0.1.162 起完全采用上游行为"
  source: brain archive-page
  affects: [openai-429-over-limit-routing]
