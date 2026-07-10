---
slug: mindmap
title: Feature mindmap
role: feature mindmap
updated: "2026-07-10T20:51:52"
---

# Feature mindmap

## 功能地图

```mermaid
mindmap
  root((Sub2API fork))
    API 网关
      OpenAI Responses
      Chat Completions
      Anthropic Messages
      Gemini v1beta
      Antigravity
      Grok
      HTTP SSE WebSocket
      模型映射与协议转换
    账号与调度
      OAuth 和 API Key 账号
      分组与渠道
      priority 与 load-aware
      粘性会话
      previous_response_id
      并发槽位与等待队列
      scheduler snapshot 和 outbox
      429 probe 与 failover
    用户与商业能力
      用户和 API Key
      订阅与余额
      用量和精确计费
      支付与 webhook
      兑换码与推广
      TOTP 和外部身份
    管理与运维
      账号测试和额度探测
      Dashboard 与 Ops
      错误透传规则
      渠道监控
      备份与清理
      批量图片任务
    数据层
      PostgreSQL
      Redis
      Ent schema
      migrations
      异步 worker
    前端
      Vue SPA
      管理后台
      用户控制台
      i18n
      OINANCE 品牌 overlay
    Fork 维护
      上游 tag 合并
      OpenAI 429 本地策略
      OAuth onboarding 兼容
      Coolify compose
      GHCR release
      本地品牌 UI
```

## 关键知识页

- [[fork-upstream-merge-boundaries]]：哪些能力跟随上游，哪些属于 fork。
- [[openai-429-over-limit-routing]]：429 账号如何重新探测并回退。
- [[coolify-ghcr-release-contract]]：Coolify 与 tag-driven release。
- [[local-branding-ui-overlay]]：默认品牌、主题和响应式前端。
- [[openai-oauth-onboarding-compat]]：本地导入和测试前 token 刷新。

## 维护视角

功能地图中的大部分业务能力由上游维护。本 fork 的主要风险集中在“Fork 维护”分支和它们与 scheduler、handler、release workflow 的交点。新需求应先判断属于上游通用能力还是本地 overlay，再选择落点。
