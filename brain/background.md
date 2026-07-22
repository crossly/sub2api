---
slug: background
title: Project background
role: project background
updated: "2026-07-22T10:41:19"
---

# Project background

## 当前定位

Sub2API 是面向 AI 订阅与多上游账号池的 API 网关。客户端使用平台签发的 API Key 调用 OpenAI、Anthropic、Gemini、Antigravity、Grok 等兼容端点，服务负责鉴权、账号调度、协议转换、并发控制、用量记录、计费和故障切换。

本仓库 `crossly/sub2api` 是 `Wei-Shaw/sub2api` 的维护型 fork。截至 2026-07-22，代码基线为上游 `v0.1.162`（`27f094e09`）加本地 Coolify/GHCR 与 OINANCE 品牌 UI overlay，维护策略见 [[fork-upstream-merge-boundaries]]。

## 目标用户

- 通过 Coolify 自建 Sub2API 的管理员和运维人员。
- 使用 Codex、Claude Code、OpenAI SDK、Gemini SDK/CLI 等客户端的开发者。
- 管理多账号、订阅额度、分组、计费、支付和运行监控的团队。

## 当前目标

- 持续吸收上游功能、安全修复和协议兼容更新。
- 通过 Coolify 与 `ghcr.io/crossly/sub2api:latest` 稳定部署，见 [[coolify-ghcr-release-contract]]。
- 保留 OINANCE 默认品牌、紫色主题、精简首页和移动端体验，见 [[local-branding-ui-overlay]]。
- 把本地运行时差异控制在两个可审计 overlay 内。

## 明确边界

- OpenAI 429、OAuth、scheduler、transport、计费、邮件和其他业务功能完全跟随上游，不维护 fork 专属行为。
- `backend/`、`Dockerfile` 和 `deploy/` 在上游合并后应与目标 tag 一致。
- Coolify 配置不侵入上游 `deploy/docker-compose*.yml`。
- 品牌 UI 不改变 API、账号状态或计费语义。
- Project Brain 保存持久决策，但不参与运行时。
- `RUN_MODE=simple` 是跳过部分 SaaS 计费流程的上游运行模式，不等同于标准生产模式。

## 版本说明

上游 tag `v0.1.162` 内的 `backend/cmd/server/VERSION` 仍可能是前一 release 的值；正式 fork tag 发布时，release workflow 从 tag 生成版本并在成功后同步默认分支 VERSION 文件。判断发布版本应以 tag 和 release artifact 为准。
