# OpenAI Over-Limit Merge Playbook

## Preserve These 112 Semantics

1. Over-limit mode is a routing policy, not a generic filter tweak.
2. Candidate lookup must broaden beyond schedulable-only repo/snapshot filtering when over-limit mode is enabled.
3. Short cooldown is scoped to `account + model`, and `429/529` should write that scoped cooldown.
4. Sticky fallback should yield when a higher-priority over-limit candidate is eligible.
5. `previous_response_id` should not keep dominating routing while over-limit mode is enabled.

## The 3 Hook Points

1. Candidate source hook
   File: `backend/internal/service/openai_gateway_service.go`
   Entry: `listSchedulableAccounts`
   Strategy call: `openAIOverLimitStrategy().CandidateAccounts(...)`

2. Sticky / continuation hook
   Files:
   - `backend/internal/service/openai_gateway_service.go`
   - `backend/internal/service/openai_ws_forwarder.go`
   Entries:
   - `shouldBypassStickySessionForOpenAIOverLimit`
   - `shouldIgnorePreviousResponseForOpenAIOverLimit`

3. Upstream error side-effect hook
   File: `backend/internal/service/openai_gateway_service.go`
   Entry: `maybeMarkOpenAIOverLimitCooldown`
   Strategy call: `openAIOverLimitStrategy().HandleUpstreamError(...)`

## Strategy Layer Ownership

Primary file:
- `backend/internal/service/openai_over_limit_strategy.go`

This file owns:
- settings normalization and cache
- cooldown keying
- cooldown state reads/writes
- account selectability under over-limit mode
- broad candidate lookup
- sticky bypass policy
- previous-response bypass policy
- upstream cooldown side effects

## What To Recheck After Upstream Merges

1. Did upstream change how OpenAI candidate accounts are loaded?
   Reconcile in `CandidateAccounts(...)`.

2. Did upstream add or reorder routing layers?
   Recheck sticky and `previous_response_id` hook placement.

3. Did upstream touch 429/529 handling?
   Reconfirm `HandleUpstreamError(...)` still receives the request model.

4. Did upstream change setting keys or cache style?
   Recheck `openAIOverLimitModeEnabled` and cooldown normalization.

## Minimum Regression Commands

```bash
go test ./internal/service -run 'OpenAIOverLimit|SelectAccountByPreviousResponseID|SelectAccountWithScheduler' -count=1
go test ./internal/service ./internal/server -count=1
```
