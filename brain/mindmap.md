---
slug: mindmap
title: Feature mindmap
role: feature mindmap
updated: "2026-07-22T10:41:19"
---

# Feature mindmap

## 功能地图

```mermaid
mindmap
  root((Sub2API fork))
    上游网关能力
      OpenAI Responses
      Chat Completions
      Anthropic Messages
      Gemini v1beta
      Antigravity
      Grok
      HTTP SSE WebSocket
      模型与协议转换
    账号与调度
      OAuth 与 API Key
      分组与渠道
      priority 与 load
      sticky 与 previous response
      并发与等待队列
      429 与 failover
    用户与商业能力
      用户和 API Key
      订阅与余额
      用量与计费
      支付与 webhook
      TOTP 与外部身份
    管理与运维
      账号测试
      Dashboard 与 Ops
      安全审计
      异步图片任务
      对象存储与备份
      邮件与通知
    数据层
      PostgreSQL
      Redis
      Ent 与 migrations
      后台 workers
    本地品牌 UI
      OINANCE logo
      紫色主题
      Landing 文案
      响应式 header
    本地部署
      Coolify compose
      crossly GHCR latest
      tag-driven release
    项目维护
      上游 tag 合并
      反向 delta 审计
      Project Brain
```

## 活动知识页

- [[fork-upstream-merge-boundaries]]：上游所有权和只保留两个 overlay 的规则。
- [[coolify-ghcr-release-contract]]：Coolify、GHCR 与 tag-driven release。
- [[local-branding-ui-overlay]]：默认品牌、主题和响应式前端。

429 与 OAuth 历史页面已归档。它们只解释过去版本，不代表当前 fork 行为。

## 维护视角

大部分业务能力由上游维护。本 fork 的审阅重点只在部署和视觉差异，以及它们与上游 release workflow、Docker 构建和前端组件结构的交点。
