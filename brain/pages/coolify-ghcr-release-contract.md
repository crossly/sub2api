---
id: coolify-ghcr-release-contract
title: "Coolify 使用自有 GHCR latest 并由 tag 驱动发布"
category: decision
status: active
tags: [coolify, ghcr, release, deployment]
created: "2026-07-10T20:50:27"
updated: "2026-07-10T20:50:27"
---

## compiled_truth

## 决策

Coolify 生产部署使用仓库根部 `docker-compose.coolify.yml`，直接拉取 `ghcr.io/crossly/sub2api:latest`，不在 Coolify 主机现场构建。上游 `deploy/docker-compose*.yml` 保持原样，避免后续上游合并持续冲突。

## Compose 契约

- 三个服务：`sub2api`、PostgreSQL、Redis。
- `sub2api` 使用 `pull_policy: always`，通过 Coolify 内部路由暴露服务，compose 本身不绑定宿主机端口。
- 应用、PostgreSQL 和 Redis 使用 named volumes。
- `POSTGRES_PASSWORD` 是必填项；Redis 密码可为空，并通过同一 `REDIS_PASSWORD` 同步服务端和客户端。
- healthcheck 和 `depends_on.condition: service_healthy` 保证数据库、Redis 就绪后启动应用。
- compose 保留长 LLM 请求、OpenAI HTTP/2/WebSocket、scheduler wait、Redis pool 和数据库 pool 的高价值参数。
- `SUB2API_IMAGE_REF` 用于启动日志识别实际镜像来源。

## 发布契约

推送 `v*` tag 触发 `.github/workflows/release.yml`：

1. 从 tag 生成 VERSION artifact，并构建嵌入式 Vue 前端。
2. GoReleaser 生成多平台二进制和 checksum。
3. 构建 amd64/arm64 镜像并发布 GHCR 的具体版本、`latest`、major.minor 和 major manifest。
4. 创建非 draft GitHub Release。
5. Release 成功后，GitHub Actions 在默认分支追加 `chore: sync VERSION to <version> [skip ci]`。

因此 release tag 指向发布代码提交，而默认分支随后可能比 tag 多一个纯 VERSION 同步提交，这是预期行为。DockerHub 仅在 secret 配置存在时发布；Coolify 的稳定依赖是 GHCR。

## 维护规则

- Coolify compose 与上游 deploy compose 分开 review。
- 发布前必须确认 main/tag CI、安全扫描和 Release workflow 全绿。
- 修改镜像仓库、tag 策略或 `latest` 行为时，必须同时检查 compose、`.goreleaser.yaml` 和 release workflow。
- 429 或 UI 改动不应顺带修改 Coolify 部署契约；边界见 [[fork-upstream-merge-boundaries]]。


## timeline

- time: 2026-07-10T20:50:27
  kind: decision
  summary: "Created this page: Coolify 使用自有 GHCR latest 并由 tag 驱动发布"
  source: "docker-compose.coolify.yml; release workflow; GoReleaser config; git history"
  affects: [coolify-ghcr-release-contract]

- time: 2026-07-10T20:50:27
  kind: decision
  summary: "记录 Coolify 三服务部署和 tag 到 GHCR/Release 的运维契约"
  source: "docker-compose.coolify.yml; release workflow; GoReleaser config; git history"
  affects: [coolify-ghcr-release-contract]
