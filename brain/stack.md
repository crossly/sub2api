---
slug: stack
title: Tech stack
role: tech-stack choices
updated: "2026-07-22T10:41:19"
---

# Tech stack

## 当前技术选择

以下以上游 `v0.1.162` manifest、CI 和部署文件为准。

| 领域 | 当前选择 | 依据与作用 |
|---|---|---|
| 后端语言 | Go 1.26.5 | `backend/go.mod` 与 CI 固定版本 |
| HTTP 框架 | Gin 1.9.1 | 路由、中间件、流式响应和协议入口 |
| 依赖注入 | Google Wire 0.7 | 编译期组装 repositories/services/handlers |
| ORM / schema | Ent 0.14.5 + SQL migrations | 业务实体、生成代码和数据库演进 |
| 主数据库 | PostgreSQL 18 | 权威业务状态、账号、用量、计费和支付 |
| 缓存与协调 | Redis 8 + go-redis/v9 | 调度投影、粘性会话、并发、缓存和队列 |
| 上游 HTTP | req/v3、net/http、uTLS | provider transport、代理和 TLS profile |
| WebSocket | coder/websocket + Gorilla WebSocket | OpenAI Responses WS 与兼容 relay |
| 配置 | Viper + YAML + environment | 二进制、Docker/Coolify 和运行模式 |
| 认证 | JWT、TOTP、OAuth/OIDC | 用户会话、管理员认证和外部身份 |
| 前端 | Vue 3.4、TypeScript、Vite 5 | SPA 和嵌入式管理后台 |
| 前端状态/路由 | Pinia、Vue Router、vue-i18n | 会话、设置、页面与多语言 |
| 样式与图表 | TailwindCSS 3.4、Chart.js | 管理界面与 [[local-branding-ui-overlay]] |
| CI 前端工具链 | Node 20 + pnpm 9 | backend-ci、security scan 和 release workflow |
| Docker 前端构建 | Node 24 Alpine | 多阶段 Dockerfile 的 native build stage |
| 测试 | Go unit/integration、Testcontainers、Vitest、vue-tsc | 后端状态路径和前端交互 |
| 静态检查 | golangci-lint v2.9、govulncheck、pnpm audit | CI 与安全扫描 |
| 发布 | GitHub Actions + GoReleaser + Buildx | 多平台 archive、GHCR manifest 和 Release |

## 运行模式

- `standard`：完整分组、订阅、计费、额度和 SaaS 能力。
- `simple`：面向个人或内部快速使用，跳过部分计费流程。
- gateway 同时暴露多种兼容路径，并依据 API Key 的 group/platform 路由到上游 service。

## 本地 stack overlay

- [[coolify-ghcr-release-contract]] 使用 PostgreSQL 18、Redis 8、named volumes 和 `ghcr.io/crossly/sub2api:latest`。
- [[local-branding-ui-overlay]] 只在现有 Vue/Tailwind 栈上应用，不引入第二套前端框架。
- 所有后端依赖与业务实现跟随上游，边界见 [[fork-upstream-merge-boundaries]]。
