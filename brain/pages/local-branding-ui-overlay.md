---
id: local-branding-ui-overlay
title: "保留 OINANCE 品牌与紫色响应式 UI overlay"
category: decision
status: active
tags: [frontend, branding, ui, overlay]
created: "2026-07-10T20:50:27"
updated: "2026-07-22T10:34:24"
---

## compiled_truth

## 决策

本 fork 保留一层独立于上游业务功能的前端品牌/UI overlay。它不改变 API、账号调度、认证、计费或 provider 语义，主要负责默认视觉、首页表达和移动端体验。

## 当前 overlay

- 默认 fallback 资产使用 OINANCE 紫色 `OI` SVG 标记；运行时 `site_name` 和 `site_logo` 仍可覆盖默认品牌。
- Tailwind primary palette、登录页、管理看板和图表色彩保持统一的紫色商业主题。
- 首页使用精简 hero 和本地中英文品牌文案，并明确展示 ChatGPT 等支持渠道。
- `AppHeader` 与首页操作在窄屏下保持紧凑、可用的响应式布局。
- 品牌和响应式行为由对应前端测试保护。

## 维护边界

- 只维护 logo/fallback、主题 token、landing 文案、首页结构和响应式 shell。
- 上游重写页面时先采用新页面语义和组件结构，再以小范围差异恢复品牌与布局，不盲目保留旧 class。
- 429、OAuth、账号设置和其他业务 UI 完全跟随上游，不属于品牌 overlay。
- 邮件模板和通知邮件模块完全跟随上游；紫色 legacy email fallback 不再作为品牌资产保留。
- 本 overlay 与 [[coolify-ghcr-release-contract]] 分开维护，并遵循 [[fork-upstream-merge-boundaries]]。


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

- time: 2026-07-22T10:34:24
  kind: decision
  summary: "将品牌 overlay 与已退役功能解耦，并明确邮件模板归上游"
  source: "用户决策 2026-07-22；upstream v0.1.162 升级"
  affects: [local-branding-ui-overlay]
