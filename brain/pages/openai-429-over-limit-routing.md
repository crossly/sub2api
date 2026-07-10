---
id: openai-429-over-limit-routing
title: "OpenAI 429 账号按间隔探测并在单次请求内回退"
category: decision
status: active
tags: [openai, 429, scheduler, failover]
created: "2026-07-10T20:50:26"
updated: "2026-07-10T20:50:27"
---

## compiled_truth

## 行为契约

管理员启用 `openai_over_limit_mode_enabled` 后，处于账号级全局 429 窗口的 OpenAI 账号不会永久退出候选池，而是在每次 429 后按 `openai_over_limit_cooldown_seconds` 定期重新探测。间隔归一化为 10 到 300 秒；关闭模式时该值序列化为 0。

目标行为是：低数值 `priority` 账号在可探测时仍优先。如果该账号再次返回 429，当前请求立即排除它并重新调度，最终可回退到仍有额度的后续账号。失败账号不能在同一个 retry loop 中被重复选中。

## 调度设计

- `SchedulerModeOpenAIOverLimit` 使用专用 snapshot bucket。bucket 包含 active OpenAI 账号，即使其 `RateLimitResetAt` 仍在未来。
- bucket 只负责扩大候选集合，不代表账号最终可用。每次选择仍从 scheduler metadata 或数据库取得新状态，并执行 request-time gate。
- over-limit 模式下，advanced scheduler 先收窄到当前最小 `priority`，再在同优先级内做 load-aware 选择。
- sticky session 只有在存在更高优先级、正处于全局 429 且已到 probe 时间的候选时才让路。
- `previous_response_id` 仅在请求分析表明 continuation 可跨账号移动时放松；不可移动的续链仍保持账号亲和。
- simple、standard、snapshot fallback 和 advanced scheduler 都必须走同一策略语义。

## 不可绕过的硬门槛

429 探测只放松“账号级全局 429 reset window”这一项。以下状态继续 fail-closed：

- disabled、非 active、`Schedulable=false`
- 已过期且启用自动暂停
- `OverloadUntil`、`TempUnschedulableUntil`
- API Key/Bedrock quota exhausted
- model unsupported 或 model-level rate limit
- endpoint capability / compact capability 不匹配
- channel restriction、shadow parent 不健康
- 并发槽位不可用
- auth、transport、529 或其他 hard runtime block

运行时 block 将 `HardUntil/HardReason` 与 `RateLimitedAt/RateLimitUntil` 分开保存。hard block 永远优先，429 不能缩短或覆盖它。正常模式遵守完整 reset；over-limit 模式只在 probe interval 到期后允许下一次探测。

## 错误与回退闭环

HTTP、SSE `response.failed` 和 WebSocket 429 都必须先进入正常的 rate-limit 持久化路径，更新账号 `RateLimitedAt` 和 `RateLimitResetAt`，再决定透传或 failover。SSE 即使命中错误透传规则或客户端已收到部分输出，也不能漏记账号状态。

在尚未向客户端提交响应时，handler 将当前 `account.ID` 加入 `failedAccountIDs`，受 `max_account_switches` 约束重新选择账号。over-limit 模式不会触发普通 OAuth 429 storm 的提前停止逻辑，因此仍可尝试健康 fallback。

该策略只改变调度和账号状态，不改写 `prompt_cache_key`、`conversation_id`、Codex request identity 或 usage attribution。

## 关键实现与回归面

- 策略：`backend/internal/service/openai_over_limit_strategy.go`
- snapshot：`scheduler_cache.go`、`scheduler_snapshot_service.go`
- request-time selection：`openai_gateway_scheduling.go`、`openai_account_scheduler.go`
- runtime block：`openai_account_runtime_block_fastpath.go`
- 429 ingestion：`openai_rate_limit_signal.go`、`ratelimit_service.go` 及各 HTTP/SSE/WS response path
- retry loops：Responses、Chat Completions、Messages、WebSocket、`/v1/messages/count_tokens`
- 设置 UI/API：`OpenAIOverLimitSection.vue` 与对应 DTO/SettingService
- 专项测试：`openai_over_limit_mode_test.go`、`openai_over_limit_v150_test.go`、count-tokens handler tests

该契约是 fork 最重要的本地行为之一，未来合并应结合 [[fork-upstream-merge-boundaries]] 重新验证，不以旧版本内部代码为准。


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
