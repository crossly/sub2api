---
slug: roadmap
title: Roadmap
role: milestones
updated: "2026-07-22T11:34:10"
---

# Roadmap

## 当前状态

截至 2026-07-22：

- 上游 `v0.1.162`（`27f094e09`）已合并并发布为 fork `v0.1.162`；fork tag 解引用到 merge commit `e876a0f0c`。
- GitHub Release `v0.1.162` 已正式发布，包含五个平台归档和 `checksums.txt`。
- GHCR `0.1.162` 与 `latest` 指向同一 manifest digest `sha256:297206a026bfcd3122ed695ee3f3bd1f9b5fef249ef26f6f6368837e3b3231a2`，包含 `linux/amd64` 与 `linux/arm64`。
- `main` 已在 v0.1.162 之后采用上游 `ef3c770d9` 与 `c5971a6fc`：Axios 升至 `1.18.1`，`golang.org/x/text` 升至 `v0.39.0` 并同步兼容的 `golang.org/x/*` 模块。
- [[fork-upstream-merge-boundaries]] 继续只保留 Coolify/GHCR 与品牌 UI 两类长期应用 overlay；上述依赖变更是上游安全热修，不是新增本地功能 overlay。

## 本次验证

- 本地 `govulncheck ./...`：0 个可达漏洞；GitHub Security Scan `29888390838` 的 backend-security 与 frontend-security 均通过。
- 前端 `pnpm audit --prod --audit-level=high` 经例外检查通过，实际解析 Axios `1.18.1`。
- 后端 `make test-unit`、`make test-integration` 通过。
- `golangci-lint v2.9.0`：0 issues。
- 前端 ESLint、Vue typecheck、CI critical Vitest（95 tests）和 production build 通过。
- GitHub 常规 CI `29888390842` 的 test、frontend、shell、golangci-lint 全部通过。
- Brain link lint 通过。

## 固定维护门槛

1. 依据 [[fork-upstream-merge-boundaries]] 验证 tag、merge-base、上游 commit 与已采用的 tag 后安全热修。
2. 业务代码冲突采用上游，不能重放已退役 overlay。
3. 只恢复 [[local-branding-ui-overlay]] 与 [[coolify-ghcr-release-contract]]。
4. 反向 diff 确认业务代码与目标 tag一致；任何 tag 后上游安全热修必须单独列出并可追溯。
5. 跑后端 unit/integration、golangci、前端 CI/品牌测试、production build、compose 检查和安全扫描。
6. 所有发布门禁通过后再创建 fork tag 和 release。

## 下一步

- 下一个 release tag 必须位于 `bf9580510` 或其后，才能包含两项安全修复；现有已发布 `v0.1.162` tag 与镜像不会被静默改写。
- 后续上游 release 包含这两个提交时，依赖热修差异自然并入上游基线。
