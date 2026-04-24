# OpenAI Over-Limit Merge Playbook

## Preserve These 112 Semantics

1. Over-limit mode is a routing policy, not a generic filter tweak.
2. Candidate lookup must broaden beyond schedulable-only repo/snapshot filtering when over-limit mode is enabled.
3. Short cooldown is scoped to `account + model`, and `429/529` should write that scoped cooldown.
4. Sticky fallback should yield when a higher-priority over-limit candidate is eligible.
5. `previous_response_id` should not keep dominating routing while over-limit mode is enabled.

## Scope Boundary

This playbook is only for OpenAI over-limit behavior.

Keep it scoped to:
- `openai_over_limit_mode_enabled`
- `openai_over_limit_cooldown_seconds`
- the short-cooldown normalization rule: empty / `< 10` => `10`
- candidate broadening, sticky bypass, and `previous_response_id` bypass while over-limit mode is active

Do not mix in:
- `openai_advanced_scheduler_enabled`
- gateway forwarding toggles
- web search emulation
- generic 529 overload cooldown settings

If upstream changes one of those unrelated areas, merge them normally instead of pulling them into the over-limit layer.

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

## Frontend Ownership

The frontend is intentionally narrowed to an over-limit-only slice.

Primary files:
- `frontend/src/views/admin/settings/openaiOverLimitFields.ts`
- `frontend/src/views/admin/settings/useOpenAIOverLimitSettings.ts`
- `frontend/src/views/admin/settings/OpenAIOverLimitSection.vue`

Integration points:
- `frontend/src/views/admin/SettingsView.vue`
- `frontend/src/views/admin/__tests__/SettingsView.spec.ts`
- `frontend/src/views/admin/settings/__tests__/openaiOverLimitFields.spec.ts`

Frontend rules:
1. Keep `openai_advanced_scheduler_enabled` inline in `SettingsView.vue`.
2. Keep the dedicated helper layer limited to over-limit mode and cooldown.
3. Serialize only:
   - `openai_over_limit_mode_enabled`
   - `openai_over_limit_cooldown_seconds`
4. On load and save, normalize cooldown to `10` when over-limit mode is enabled and the value is empty or `< 10`.
5. When over-limit mode is disabled, save cooldown as `0`.

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

5. Did upstream change the admin settings page around OpenAI routing?
   Recheck that the over-limit-only frontend files still own only:
   - `openai_over_limit_mode_enabled`
   - `openai_over_limit_cooldown_seconds`

6. Did upstream add more OpenAI toggles nearby?
   Keep them inline unless they are directly part of over-limit semantics.

## Merge Procedure

1. Merge upstream into `main`.
2. Reapply backend hook points only if upstream touched them.
3. Reapply frontend over-limit-only files only if upstream touched the OpenAI gateway settings block.
4. Do not re-abstract unrelated settings while resolving conflicts.
5. Re-run the regression commands below before committing.

## Minimum Regression Commands

```bash
go test ./internal/service -run 'OpenAIOverLimit|SelectAccountByPreviousResponseID|SelectAccountWithScheduler' -count=1
go test ./internal/service ./internal/server -count=1
cd frontend && pnpm test:run src/views/admin/settings/__tests__/openaiOverLimitFields.spec.ts src/views/admin/__tests__/SettingsView.spec.ts
cd frontend && pnpm exec vue-tsc --noEmit
```
