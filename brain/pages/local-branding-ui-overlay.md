---
id: local-branding-ui-overlay
title: "保留 OINANCE 品牌与紫色响应式 UI overlay"
category: decision
status: active
tags: [frontend, branding, ui, overlay]
created: "2026-07-10T20:50:27"
updated: "2026-07-10T20:50:27"
---

## compiled_truth

## 决策

本 fork 保留一层独立于上游业务功能的前端品牌/UI overlay。它不改变 API 或调度语义，主要负责默认视觉、首页表达和移动端体验。

## 当前 overlay

- 默认 fallback 资产从 `/logo.png` 改为 `/logo.svg`，当前 SVG 是 OINANCE 的紫色 `OI` 标记。
- Tailwind primary palette、登录页背景、管理看板和图表色彩使用统一的紫色商业主题。
- 首页采用精简 hero 结构，核心文案是 `AI API Gateway`、`Native Direct`、`Long-Term Stable`，中文对应“AI API 网关”“原生直连”“长效稳定”。
- provider 展示明确包含 ChatGPT。
- `AppHeader` 和首页操作在窄屏下使用更紧凑的布局与菜单，并有 source-based responsive tests 保护。
- `site_name` 和 `site_logo` 运行时设置仍可覆盖默认品牌；本 overlay 定义的是 fallback 和默认体验，不锁死租户品牌。

## 维护边界

- 品牌 copy 只改 landing locale key，429 管理设置只改 `admin.settings.openaiOverLimitMode`，避免在 locale 文件中混杂 concern。
- logo/fallback、主题 token、首页结构、响应式 shell 应作为不同小批次 review，但共同构成本 overlay。
- 上游重写页面时先恢复语义和布局，再恢复颜色与品牌，不盲目保留旧 class 字符串。
- 响应式行为改动必须同步调整对应 guard tests。
- 该 overlay 与 [[openai-429-over-limit-routing]]、[[coolify-ghcr-release-contract]] 分开维护，并遵循 [[fork-upstream-merge-boundaries]]。

## 当前不确定项

源码中的 legacy email fallback 也使用紫色 business template，但邮件发送已由上游模块化服务优先处理。邮件是否继续作为品牌 overlay 的一部分尚未确认，当前不纳入本决策的稳定范围。


## timeline

- time: 2026-07-10T20:50:27
  kind: decision
  summary: "Created this page: 保留 OINANCE 品牌与紫色响应式 UI overlay"
  source: "frontend diff against upstream v0.1.150; git history; maintenance map"
  affects: [local-branding-ui-overlay]

- time: 2026-07-10T20:50:27
  kind: decision
  summary: "记录需要跨上游合并保留的品牌、主题和响应式 UI 边界"
  source: "frontend diff against upstream v0.1.150; git history; maintenance map"
  affects: [local-branding-ui-overlay]
