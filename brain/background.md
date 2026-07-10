---
slug: background
title: Project background
role: project background
updated: "2026-07-10T20:51:52"
---

# Project background

## 当前定位

Sub2API 是一个面向 AI 订阅与多上游账号池的 API 网关。客户端使用平台签发的 API Key 调用 OpenAI、Anthropic、Gemini、Antigravity、Grok 等兼容端点，服务负责鉴权、账号调度、协议转换、并发控制、用量记录、计费和故障切换。

本仓库 `crossly/sub2api` 是 `Wei-Shaw/sub2api` 的维护型 fork。截至 2026-07-10，运行基线是上游 `v0.1.150` 加本地 release `v0.1.150.1`。主体能力跟随上游演进，本地只保留少量明确 overlay，维护策略见 [[fork-upstream-merge-boundaries]]。

## 目标用户

- 自建 Sub2API 的管理员和运维人员。
- 通过统一 API Key 使用 Codex、Claude Code、OpenAI SDK、Gemini SDK/CLI 等客户端的开发者。
- 需要管理多账号、订阅额度、分组、计费、支付和运行监控的团队。

## 当前目标

- 保持上游主要功能与安全修复可持续合并。
- 让 OpenAI 账号级 429 在可控间隔后按优先级再次探测，并在探测失败时回退到健康账号，见 [[openai-429-over-limit-routing]]。
- 通过 Coolify 和自有 GHCR 镜像稳定部署，见 [[coolify-ghcr-release-contract]]。
- 保留 OINANCE 默认品牌、紫色主题、精简首页和移动端体验，见 [[local-branding-ui-overlay]]。
- 保留本地 OpenAI OAuth 导入和账号测试前 token 刷新能力，见 [[openai-oauth-onboarding-compat]]。

## 明确边界

- 不脱离上游重写整个网关；本地变更应保持小、可审计、可重放。
- 429 模式不绕过账号硬状态、模型/能力限制、并发限制或 529/auth/transport block。
- 429 模式不改写 Codex request identity、`prompt_cache_key` 或 usage attribution。
- Coolify 配置不侵入上游 `deploy/docker-compose*.yml`。
- 私有 token 导入数据只在本地生成，`output/` 不进入版本控制。
- `RUN_MODE=simple` 是跳过 SaaS 计费/额度检查的独立运行模式，不等同于标准生产模式。

## 证据与待确认

技术定位、当前架构和本地 overlay 均由当前源码、manifest、Git 历史及已通过的 CI 支持，置信度高。

未找到用户确认的产品商业路线图。邮件方面，上游已引入模块化通知邮件服务，但当前 fork 仍保留本地紫色 legacy fallback；其长期所有权需要在下次合并前确认。README 中部分版本号已经落后于实际 manifest，也应作为文档维护项。
