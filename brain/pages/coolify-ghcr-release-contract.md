---
id: coolify-ghcr-release-contract
title: "Coolify 使用自有 GHCR latest 并由 tag 驱动发布"
category: decision
status: active
tags: [coolify, ghcr, release, deployment]
created: "2026-07-10T20:50:27"
updated: "2026-07-22T10:50:28"
---

## compiled_truth

## 决策

Coolify 生产部署使用仓库根部 `docker-compose.coolify.yml`，直接拉取 `ghcr.io/crossly/sub2api:latest`，不在 Coolify 主机现场构建。上游 `Dockerfile` 和 `deploy/docker-compose*.yml` 保持原样，Coolify 特有差异只维护在根目录 compose。

## Compose 契约

- 三个服务：`sub2api`、PostgreSQL 18、Redis 8。
- `sub2api` 使用 `pull_policy: always`，通过 Coolify 内部路由暴露服务，compose 不绑定宿主机端口。
- 应用、PostgreSQL 和 Redis 使用 named volumes。
- `POSTGRES_PASSWORD` 必填；Redis 密码可为空，并通过同一 `REDIS_PASSWORD` 同步服务端、应用和 healthcheck。
- healthcheck 与 `depends_on.condition: service_healthy` 保证依赖就绪后启动应用。
- `SERVER_HOST=0.0.0.0`、`SERVER_PORT=8080` 固定容器内监听；`SUB2API_IMAGE_REF` 用于日志识别实际镜像。
- compose 暴露上游标准的 database/Redis、setup migration、update token、server timing 和 image stream/concurrency 环境入口。
- 保留长 LLM 请求、OpenAI HTTP/2/WebSocket、scheduler wait、Redis pool 和数据库 pool 的高价值参数。
- PostgreSQL `command` 将 `POSTGRES_MAX_CONNECTIONS`、shared buffers、effective cache 和 maintenance memory 真正传给进程。
- Redis 多行 `sh -c` 命令使用反斜杠续行，确保 persistence 和可选 `requirepass` 参数实际生效。

## 发布契约

推送 `v*` tag 触发 `.github/workflows/release.yml`：

1. 从 tag 生成 VERSION artifact，并构建嵌入式 Vue 前端。
2. GoReleaser 生成多平台二进制和 checksum。
3. 构建 amd64/arm64 镜像并发布当前仓库 owner 对应 GHCR 的具体版本、`latest`、major.minor 和 major manifest。
4. 创建非 draft GitHub Release。
5. Release 成功后，GitHub Actions 在默认分支追加 `chore: sync VERSION to <version> [skip ci]`。

release tag 指向发布代码提交，而默认分支随后可能多一个纯 VERSION 同步提交，这是预期行为。DockerHub 仅在 secret 配置存在时发布；Coolify 的稳定依赖是 GHCR。

## 维护规则

- Coolify compose 与上游 deploy compose 分开 review。
- 每次上游 compose 修复都要判断是否应同步到 Coolify 文件，尤其是命令行参数和新增配置入口。
- 发布前确认 main/tag CI、安全扫描和 Release workflow 全绿。
- 修改镜像仓库、tag 策略或 `latest` 行为时，同时检查 compose、GoReleaser 和 release workflow。
- 本边界遵循 [[fork-upstream-merge-boundaries]]，不承载业务功能补丁。


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

- time: 2026-07-22T10:41:19
  kind: decision
  summary: "同步 v0.1.162 compose 环境入口与 PostgreSQL/Redis 启动修复"
  source: "docker-compose.coolify.yml；upstream v0.1.162 deploy/docker-compose.yml"
  affects: [coolify-ghcr-release-contract]

- time: 2026-07-22T10:50:28
  kind: evidence
  summary: "Coolify compose 已覆盖上游环境入口并同步 PostgreSQL/Redis 命令修复，YAML 解析通过"
  source: "docker-compose.coolify.yml；upstream v0.1.162 compose comparison"
  affects: [coolify-ghcr-release-contract]
