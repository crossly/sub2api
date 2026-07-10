---
slug: flow
title: Key flows
role: key flows
updated: "2026-07-10T20:51:52"
---

# Key flows

## 典型 OpenAI 请求链

```mermaid
sequenceDiagram
    participant C as Client
    participant M as Gin middleware
    participant H as OpenAIGatewayHandler
    participant B as Billing/Moderation
    participant S as OpenAI scheduler
    participant R as Redis snapshot/concurrency
    participant DB as PostgreSQL
    participant U as OpenAI-compatible upstream
    participant W as Usage worker

    C->>M: POST /v1/responses or /v1/messages
    M->>M: body limit, request id, ops logging
    M->>DB: resolve API key, user, group, subscription
    M->>H: authenticated request context
    H->>H: parse body, normalize endpoint/model/session
    H->>B: billing eligibility and feature gates
    B-->>H: allowed
    loop within max_account_switches
        H->>S: select(previous_response, sticky, model, exclusions)
        S->>R: read snapshot/sticky/load and acquire slot
        opt 429 over-limit mode
            S->>S: use probe bucket and recheck hard gates
            S->>DB: fresh account recheck/fallback
        end
        S-->>H: selected account + release function
        H->>U: mapped HTTP/SSE/WebSocket request
        alt success
            U-->>H: response stream/result + usage
            H-->>C: protocol-compatible response
            H->>W: enqueue usage record
            W->>DB: usage, billing and account stats
        else failover-capable error before client output
            U-->>H: 429/transport/failover error
            H->>DB: persist account/error state
            H->>H: add account ID to exclusion set
            H->>S: reschedule next account
        else response already committed or terminal error
            U-->>H: terminal failure
            H-->>C: mapped upstream error
        end
    end
```

## 429 特殊闭环

启用 [[openai-429-over-limit-routing]] 时：

1. SettingService 从数据库读取开关和 10 到 300 秒 probe interval，并用短 TTL 缓存。
2. scheduler 使用 `SchedulerModeOpenAIOverLimit` 候选 bucket。
3. request-time policy 只放松仍在 `RateLimitResetAt` 窗口内的账号级 429，并要求 `RateLimitedAt + interval` 已到。
4. 账号仍须通过 active、expiry、overload、temp-unschedulable、model、capability、channel、runtime hard block 和 concurrency gate。
5. HTTP/SSE/WS 429 先写入 rate-limit state。
6. 若客户端响应尚未提交，handler 将当前账号加入 `failedAccountIDs` 并选择下一账号。
7. 健康账号成功后进入正常 usage/billing 记录；失败账号等待下一 probe interval。

## 管理设置链

```mermaid
sequenceDiagram
    participant A as Admin browser
    participant V as Vue settings form
    participant API as Admin setting handler
    participant SS as SettingService
    participant DB as PostgreSQL
    participant G as Gateway hot path

    A->>V: toggle 429 probing / edit interval
    V->>V: normalize to 10..300 or 0 when disabled
    V->>API: update settings DTO
    API->>SS: validate and persist
    SS->>DB: write setting keys
    SS->>SS: refresh in-process cache/version
    G->>SS: read cached strategy settings
```

## 数据一致性

- PostgreSQL 账号和设置是权威状态。
- Redis scheduler snapshot 是可重建投影，outbox 和 full rebuild 负责收敛。
- 选择前的 fresh/DB recheck 用来防止 snapshot 延迟把刚刚被限流或禁用的账号重新选中。
- usage 写入在 worker pool 中异步执行，但 billing eligibility 在转发前同步检查。
