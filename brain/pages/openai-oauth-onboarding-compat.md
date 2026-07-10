---
id: openai-oauth-onboarding-compat
title: "保留 OpenAI OAuth 本地导入与测试前 token 刷新"
category: decision
status: active
tags: [openai, oauth, import, account-test]
created: "2026-07-10T20:50:27"
updated: "2026-07-10T20:50:27"
---

## compiled_truth

## 决策

本 fork 保留 OpenAI OAuth 账号 onboarding 的两个兼容层：本地文本转导入 JSON，以及管理员账号连接测试前刷新 token。它们是独立于 429 调度的账号维护能力。

## 本地导入工具

`backend/cmd/codex_txt_to_sub2api` 将四列 tab-separated 文本转换为 `sub2api-data` version 1：

1. `access_token`
2. email
3. `id_token`
4. `refresh_token|client_id`

输出始终包含 `proxies: []`，满足导入 schema；账号类型是 `platform=openai`、`type=oauth`。工具从 `id_token` 尝试补全 `chatgpt_account_id`、`chatgpt_user_id`、`plan_type` 和 `organization_id`，并将 `expires_at` 设为过去时间，促使服务导入后立即刷新，而不是先使用可能过期的 access token。

该工具处理私有 token 数据，应只在本地运行。输出目录 `output/` 已被忽略，不应把生成的账号 JSON 提交进仓库。

## 账号测试前刷新

`AccountTestService` 对带 `refresh_token` 的 OpenAI OAuth 账号，在普通和 compact probe 前调用 `OpenAIOAuthService.RefreshAccountToken`：

- 刷新成功后用新 credential 更新本次内存账号，再发测试请求。
- 刷新失败时停止 probe，避免拿已知失效 token 请求上游并产生误导结果。
- 无 refresh token 或非 OpenAI OAuth 账号保持上游默认测试行为。

## 维护影响

- Wire 依赖注入和所有 test stub 必须跟随 `AccountTestService` 构造函数签名。
- 导入 schema 变化时必须同步 converter 和 tests，尤其不能再次遗漏 required `proxies`。
- 该兼容层不应改写 429 策略；调度行为由 [[openai-429-over-limit-routing]] 独立负责。
- 上游合并时按 [[fork-upstream-merge-boundaries]] 单独 reapply 和验证。


## timeline

- time: 2026-07-10T20:50:27
  kind: decision
  summary: "Created this page: 保留 OpenAI OAuth 本地导入与测试前 token 刷新"
  source: "ba048f556; account_test_service.go; codex_txt_to_sub2api command"
  affects: [openai-oauth-onboarding-compat]

- time: 2026-07-10T20:50:27
  kind: decision
  summary: "记录 OpenAI OAuth 账号导入和连接测试兼容层的本地所有权"
  source: "ba048f556; account_test_service.go; codex_txt_to_sub2api command"
  affects: [openai-oauth-onboarding-compat]
