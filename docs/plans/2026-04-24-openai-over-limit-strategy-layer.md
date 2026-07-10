# OpenAI 429 Over-Limit Strategy on v0.1.150

## Baseline

This implementation is designed against the unmodified upstream `v0.1.150`
architecture. Older implementations are behavioral history only; their
account/model cooldown map, request identity rewrites, and scheduler query
paths are not part of this design.

## Required Behavior

1. A lower numeric account priority remains preferred.
2. An OpenAI account inside a global 429 reset window may be probed again only
   after the configured probe interval.
3. If that probe returns 429, the current request excludes that account and
   falls back to another eligible account.
4. Hard account state still wins: disabled, expired, temp-unschedulable,
   overloaded, unsupported-model, model-rate-limited, and unusable API-key
   quota states are never bypassed.
5. HTTP 529 and transport failures keep their independent cooldown policies.
6. The mode changes scheduling and account state only. It does not rewrite
   `prompt_cache_key`, `conversation_id`, or Codex request identity.

## Architecture

### Settings

`openAIOverLimitStrategy.Settings` reads the two existing settings through a
per-`SettingService` cache:

- `openai_over_limit_mode_enabled`
- `openai_over_limit_cooldown_seconds`

The interval is normalized to 10-300 seconds while enabled. A settings update
refreshes the in-process value immediately; normal reads use a five-second
cache to keep the scheduling hot path off the database.

### Candidate snapshot

`SchedulerModeOpenAIOverLimit` owns a dedicated scheduler bucket. The bucket
contains active OpenAI accounts even when their global 429 reset window is
active. Current account data is still hydrated from scheduler account metadata
on each read.

The bucket participates in:

- startup/default rebuilds
- full rebuilds
- account and group outbox rebuilds
- database fallback
- simple and standard run modes

This avoids a database scan on every request while preserving current account
state at selection time.

### Request-time eligibility

`openAIOverLimitStrategy.IsAccountSelectable` starts from the upstream v150
hard gates and relaxes exactly one condition: an active account-level 429
window. Such an account is selectable only after
`RateLimitedAt + probe interval`.

The stale quota auto-pause snapshot associated with that same global 429 is
ignored for the probe. Independent model limits and all other hard gates remain
active.

### Runtime circuit state

The in-process block stores independent fields for:

- hard block deadline and reason
- 429 observation time and reset deadline

Hard blocks always win. In normal mode, a 429 blocks until its full reset. In
over-limit mode, it blocks until the next probe interval. Keeping these fields
separate prevents a later 429 from shortening or overwriting auth, transport,
529, or other hard blocks.

### 429 signal ingestion

HTTP, SSE, and WebSocket paths share one OpenAI rate-limit classifier. It
recognizes rate/usage limit types and codes, insufficient quota, quota exceeded,
and equivalent messages.

SSE `response.failed` handling persists the account state before passthrough or
client-output decisions. This includes native Responses, passthrough,
Chat Completions, Anthropic Messages, and non-streaming SSE-to-JSON bridges.
Reset metadata is preserved when the nested `response.error` is normalized.

`/v1/messages/count_tokens` uses the same exclusion-and-reschedule loop as the
other gateway handlers, so a 429 can switch accounts before any response body
is written.

## Verification

```bash
cd backend
go test ./internal/service \
  -run 'OpenAIOverLimit|OverLimitV150|ForwardCountTokensAsAnthropic_429ReturnsFailoverBeforeWriting|StreamFailedRateLimitPersistsNextProbeTime' \
  -count=1
go test -tags=unit ./internal/handler \
  -run 'OpenAIGatewayHandlerCountTokens_OverLimit429FallsBackToHealthyAccount' \
  -count=1
```
