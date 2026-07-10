# OpenAI 429 Over-Limit Merge Playbook

## Source of Truth

For every upstream merge, begin with the new upstream architecture and reapply
the behavior below. Do not transplant the v0.1.126 implementation or assume its
cache, request transform, retry, and scheduler entry points still exist.

## Contract to Preserve

1. Lower numeric priority accounts are probed first when eligible.
2. A global OpenAI 429 is retried after a configurable 10-300 second interval,
   not continuously and not only after the full upstream reset.
3. A failed probe is excluded from the current retry loop so a healthy account
   can serve the request.
4. Disabled, expired, overloaded, temp-unschedulable, model-limited,
   capability-incompatible, and runtime hard-blocked accounts remain excluded.
5. 529, authentication, transport, and custom hard-block behavior remains
   independent from 429 probing.
6. Over-limit mode does not alter Codex request identity or cache fields.

## v0.1.150 Integration Points

### 1. Scheduler snapshot membership

Files:

- `backend/internal/service/scheduler_cache.go`
- `backend/internal/service/scheduler_snapshot_service.go`

Preserve `SchedulerModeOpenAIOverLimit` in startup, full, outbox, group/account,
and DB-fallback rebuild paths. Snapshot membership may include globally
rate-limited accounts; request-time policy performs final eligibility checks.

### 2. Request-time selection

Files:

- `backend/internal/service/openai_over_limit_strategy.go`
- `backend/internal/service/openai_gateway_scheduling.go`
- `backend/internal/service/openai_account_scheduler.go`
- `backend/internal/service/openai_gateway_service.go`

Both the standard and advanced schedulers must use the dedicated candidate pool
when the mode is enabled. Final selection must recheck fresh account state,
model/capability restrictions, runtime blocks, exclusions, priority, and
concurrency.

Sticky routing may yield only when a better-priority over-limit candidate is
actually probe-ready. `previous_response_id` affinity may be relaxed only when
the v150 request analysis says the continuation can move between accounts.

### 3. Runtime blocks

File:

- `backend/internal/service/openai_account_runtime_block_fastpath.go`

Keep hard blocks and 429 state separate. Normal mode observes the full reset;
over-limit mode observes the probe interval. Never let a 429 overwrite a hard
deadline or reason.

### 4. Error signals and persistence

Files:

- `backend/internal/service/openai_rate_limit_signal.go`
- `backend/internal/service/openai_gateway_upstream_errors.go`
- `backend/internal/service/openai_gateway_response_handling.go`
- `backend/internal/service/openai_gateway_passthrough.go`
- `backend/internal/service/openai_gateway_chat_completions.go`
- `backend/internal/service/openai_gateway_messages.go`
- `backend/internal/service/openai_ws_forwarder_support.go`
- `backend/internal/service/ratelimit_service.go`

Every real HTTP/SSE/WebSocket 429 signal must reach the normal account
rate-limit persistence path. SSE failures must be recorded even when an error
passthrough rule matches or output has already reached the client.

### 5. Handler retry loops

Files:

- `backend/internal/handler/openai_gateway_count_tokens.go`
- the current v150 Responses/Chat/Messages handler retry entry points

Before writing a response, a failover-capable 429 must add the current account
to the request exclusion set and reschedule within the configured switch budget.

## Frontend and Settings Ownership

The existing frontend section continues to own only:

- `openai_over_limit_mode_enabled`
- `openai_over_limit_cooldown_seconds`

Keep API DTOs, service parsing/update logic, audit fields, frontend types,
normalization, and Chinese/English copy in sync. Disabled mode serializes a zero
interval; enabled mode normalizes to 10-300 seconds.

## Merge Review Checklist

- Did upstream change scheduler bucket registration or hydration?
- Did account/group outbox rebuild paths change?
- Did account eligibility gain a new hard gate?
- Did sticky or `previous_response_id` routing move?
- Did any endpoint add a new SSE or WebSocket error path?
- Does every retry loop exclude the account that just returned 429?
- Does a 529 still remain blocked under over-limit mode?
- Are `prompt_cache_key` and `conversation_id` unchanged from upstream?
- Do simple mode and standard grouped mode both use the correct bucket?

## Minimum Regression Commands

```bash
cd backend
go test ./internal/service \
  -run 'OpenAIOverLimit|OverLimitV150|SelectAccountByPreviousResponseID|ForwardCountTokensAsAnthropic_429ReturnsFailoverBeforeWriting' \
  -count=1
go test -tags=unit ./internal/handler \
  -run 'OpenAIGatewayHandlerCountTokens_OverLimit429FallsBackToHealthyAccount' \
  -count=1
make test-unit
make test-integration

cd ../frontend
pnpm test:run \
  src/views/admin/settings/__tests__/openaiOverLimitFields.spec.ts \
  src/views/admin/__tests__/SettingsView.spec.ts
pnpm exec vue-tsc --noEmit
```
