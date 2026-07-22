---
slug: roadmap
title: Roadmap
role: milestones
updated: "2026-07-22T11:09:11"
---

# Roadmap

## 当前状态

截至 2026-07-22：

- 上游 `v0.1.162`（`27f094e09`）已合并并发布为 fork `v0.1.162`；fork tag 解引用到 merge commit `e876a0f0c`。
- GitHub Release `v0.1.162` 已正式发布，包含五个平台归档和 `checksums.txt`。
- GHCR `0.1.162` 与 `latest` 指向同一 manifest digest `sha256:297206a026bfcd3122ed695ee3f3bd1f9b5fef249ef26f6f6368837e3b3231a2`，包含 `linux/amd64` 与 `linux/arm64`。
- release workflow 已在 `main` 追加 `625fc91d7 chore: sync VERSION to 0.1.162 [skip ci]`。
- [[fork-upstream-merge-boundaries]] 已收敛为只保留 Coolify/GHCR 与品牌 UI 两类应用 overlay；历史本地 429 和 OAuth onboarding overlay 已归档，运行时完全采用上游实现。

## 本次验证

- 后端 `make test-unit`、`make test-integration` 通过。
- `golangci-lint v2.9.0`：0 issues。
- 前端 ESLint、Vue typecheck、CI critical Vitest、品牌响应式专项 Vitest和 production build 通过。
- Apple container shell/lifecycle tests 通过。
- Coolify compose 通过 YAML 结构解析，且环境键覆盖上游标准 compose 的全部入口。
- main 与 tag 的常规 GitHub CI 全绿；Release workflow 全绿。
- Security Scan 因上游基线中的 `golang.org/x/text v0.37.0`（修复版 `v0.39.0`）和 Axios `1.16.x`（修复版 `1.18.1`）告警失败；该依赖修复已存在于 `upstream/main`，尚未包含在 `v0.1.162` tag。
- Brain link lint 通过。

## 固定维护门槛

1. 依据 [[fork-upstream-merge-boundaries]] 验证 tag、merge-base 和上游 commit。
2. 业务代码冲突采用上游，不能重放已退役 overlay。
3. 只恢复 [[local-branding-ui-overlay]] 与 [[coolify-ghcr-release-contract]]。
4. 反向 diff 确认 `backend/`、`Dockerfile`、`deploy/` 与目标 tag 一致。
5. 跑后端 unit/integration、golangci、前端 CI/品牌测试、production build、compose 检查和安全扫描。
6. 验证通过后再 push fork main、创建 fork tag 和 release。

## 下一步

- 后续上游合并时纳入 Axios 与 `golang.org/x/text` 的安全升级，使 Security Scan 恢复全绿。
- Coolify 更新时拉取 `ghcr.io/crossly/sub2api:latest`，并通过 `SUB2API_IMAGE_REF` 与运行日志确认实际镜像版本。
