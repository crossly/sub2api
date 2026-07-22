---
slug: roadmap
title: Roadmap
role: milestones
updated: "2026-07-22T10:50:28"
---

# Roadmap

## 当前状态

截至 2026-07-22：

- 上游 `v0.1.162`（`27f094e09`）已合并到本地 `main`，等待用户决定何时 push/tag/release。
- [[fork-upstream-merge-boundaries]] 已收敛为只保留 Coolify/GHCR 与品牌 UI 两类应用 overlay。
- 历史本地 429 和 OAuth onboarding overlay 已归档，运行时完全采用上游实现。
- [[coolify-ghcr-release-contract]] 已同步 v0.1.162 新环境入口及 PostgreSQL/Redis 启动修复。
- [[local-branding-ui-overlay]] 已在新版前端上保留。

## 本次验证

- 后端 `make test-unit` 通过。
- 后端 `make test-integration` 通过。
- `golangci-lint v2.9.0`：0 issues。
- 前端 ESLint、Vue typecheck 和 CI critical Vitest：6 files / 95 tests 通过。
- 品牌响应式专项 Vitest：2 files / 2 tests 通过。
- 前端 production build 通过。
- Apple container shell/lifecycle tests 通过。
- Coolify compose 通过 YAML 结构解析，且环境键覆盖上游标准 compose 的全部入口。
- Brain link lint 通过。

## 固定维护门槛

1. 依据 [[fork-upstream-merge-boundaries]] 验证 tag、merge-base 和上游 commit。
2. 业务代码冲突采用上游，不能重放已退役 overlay。
3. 只恢复 [[local-branding-ui-overlay]] 与 [[coolify-ghcr-release-contract]]。
4. 反向 diff 确认 `backend/`、`Dockerfile`、`deploy/` 与目标 tag 一致。
5. 跑后端 unit/integration、golangci、前端 CI/品牌测试、production build 和 compose 检查。
6. 验证通过后再 push fork main、创建 fork tag 和 release。

## 下一步

当前合并只在本地完成，尚未 push、打 tag 或发布 release。发布动作应由用户明确指定版本号后执行；release 成功后确认 GHCR `latest` 和具体版本 manifest，再由 Coolify 拉取。
