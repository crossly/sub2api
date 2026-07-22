---
slug: flow
title: Key flows
role: key flows
updated: "2026-07-22T10:41:19"
---

# Key flows

## 典型网关请求

```mermaid
sequenceDiagram
    participant C as Client
    participant M as Gin middleware
    participant H as Gateway handler
    participant S as Upstream scheduler
    participant R as Redis
    participant DB as PostgreSQL
    participant U as Provider upstream
    participant W as Async workers

    C->>M: API request
    M->>DB: resolve key, user, group, subscription
    M->>H: authenticated context
    H->>H: parse and normalize protocol/model
    loop upstream retry policy
        H->>S: select account with exclusions
        S->>R: cache/sticky/concurrency coordination
        S->>DB: authoritative state when required
        S-->>H: account and release function
        H->>U: mapped HTTP/SSE/WebSocket request
        alt success
            U-->>H: response and usage
            H-->>C: protocol-compatible response
            H->>W: enqueue usage/audit work
            W->>DB: usage, billing and state
        else upstream-defined failover
            U-->>H: retryable failure
            H->>H: update state and exclusion set
        else terminal or committed response
            H-->>C: mapped terminal error
        end
    end
```

429、OAuth、账号切换和错误分类全部采用上游目标版本；fork 不再定义独立探测或优先级语义。

## 上游升级流程

```mermaid
flowchart LR
    Fetch["Fetch and verify upstream tag"] --> Base["Confirm merge-base"]
    Base --> Merge["Merge tag without immediate commit"]
    Merge --> Resolve["Use upstream for business conflicts"]
    Resolve --> Restore["Reapply Coolify and branding overlays"]
    Restore --> Audit["Diff result against upstream tag"]
    Audit --> Test["Backend, frontend, lint, compose checks"]
    Test --> Commit["Create merge commit"]
```

反向审计必须确认 `backend/`、`Dockerfile` 和 `deploy/` 与目标 tag 一致，剩余差异只属于 [[coolify-ghcr-release-contract]]、[[local-branding-ui-overlay]] 和 Project Brain。

## 发布与 Coolify

```mermaid
sequenceDiagram
    participant D as Developer
    participant G as GitHub Actions
    participant R as GitHub Release
    participant C as GHCR
    participant F as Coolify

    D->>G: push v* tag
    G->>G: test/build/version artifact
    G->>R: publish binaries and checksums
    G->>C: publish version and latest manifests
    G->>D: sync VERSION on default branch
    F->>C: pull ghcr.io/crossly/sub2api:latest
    F->>F: restart app with PostgreSQL and Redis
```
