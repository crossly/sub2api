---
slug: stack
title: Tech stack
role: tech-stack choices
updated: "2026-07-10T20:51:52"
---

# Tech stack

## 当前技术选择

以下以实际 manifest 和部署文件为准；README 中 Go/PostgreSQL/Redis 版本描述存在滞后。

| 领域 | 当前选择 | 依据与作用 |
|---|---|---|
| 后端语言 | Go 1.26.5 | `backend/go.mod` 和 CI 固定版本；单二进制、并发友好 |
| HTTP 框架 | Gin 1.9.1 | 路由、中间件、流式响应和协议入口 |
| 依赖注入 | Google Wire | 编译期组装 repositories/services/handlers/workers |
| ORM / schema | Ent 0.14.5 + SQL migrations | 业务实体、生成代码和增量数据库迁移 |
| 主数据库 | PostgreSQL，Coolify 当前使用 18-alpine | 权威业务状态、用量、计费、支付、账号和设置 |
| 缓存与协调 | Redis 8-alpine，go-redis/v9 | scheduler snapshot、粘性会话、并发、队列、缓存和跨实例通知 |
| 上游 HTTP | `req/v3`、标准 `net/http`、uTLS | provider transport、代理、HTTP/2 和 TLS profile |
| WebSocket | `coder/websocket` + Gorilla WebSocket | OpenAI Responses WS 与兼容 relay |
| 配置 | Viper + YAML + environment | 二进制、Docker/Coolify 和运行模式配置 |
| 认证 | JWT、TOTP、OAuth/OIDC provider | 用户会话、管理员认证和外部身份 |
| 支付 | Stripe、支付宝、微信及 provider abstraction | 内置充值、订阅和 webhook |
| 前端 | Vue 3.4、TypeScript 5.6、Vite 5 | SPA 和嵌入式管理后台 |
| 前端状态/路由 | Pinia、Vue Router、vue-i18n | 会话、设置、页面与中英文文案 |
| 样式与图表 | TailwindCSS 3.4、Chart.js | 管理界面与 [[local-branding-ui-overlay]] |
| 测试 | Go unit/integration/e2e、Testcontainers、Vitest、vue-tsc | 后端状态路径、数据库/Redis、前端关键交互 |
| 静态检查 | golangci-lint v2.9、govulncheck、pnpm audit | CI 和安全扫描 |
| 构建 | pnpm 10.33、multi-stage Docker、embedded frontend | 可复现前端和单镜像交付 |
| 发布 | GitHub Actions + GoReleaser + QEMU/Buildx | 多平台 archive、GHCR multi-arch 和 GitHub Release |

## 运行模式

- `standard`：完整分组、订阅、计费、额度和 SaaS 能力。
- `simple`：面向个人/内部快速使用，跳过计费流程；release 环境还需要显式确认。
- gateway 同时暴露 OpenAI/Anthropic/Gemini 兼容路径，并依据 API Key 的 group platform 路由到对应 service。

## 本地 stack overlay

- [[coolify-ghcr-release-contract]] 固定生产 compose 使用自有 GHCR。
- [[openai-429-over-limit-routing]] 在现有 scheduler/Redis/PostgreSQL 模型上扩展行为，不引入第二套调度基础设施。
- [[openai-oauth-onboarding-compat]] 复用现有 OpenAI OAuth service 和 import schema。
- 前端只在现有 Vue/Tailwind 栈上应用 [[local-branding-ui-overlay]]。

## 文档漂移

当前 `README_CN.md` 仍写 Go 1.25.7、PostgreSQL 15+、Redis 7+，而代码和 Coolify manifest 已是 Go 1.26.5、PostgreSQL 18、Redis 8。未来更新文档时应以 `go.mod`、CI、Dockerfile 和 compose 为事实源。
