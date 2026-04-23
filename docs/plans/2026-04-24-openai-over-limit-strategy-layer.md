# OpenAI Over-Limit Strategy Layer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor the current `116` OpenAI over-limit logic into an independent strategy layer that preserves the `112` routing semantics while minimizing future merge conflicts with upstream.

**Architecture:** Introduce a small strategy object that owns over-limit settings, short-cooldown state, candidate-pool broadening, sticky/session bypass rules, and upstream `429/529` cooldown side effects. The gateway and advanced scheduler should stop open-coding over-limit behavior and instead call a few focused hooks. The policy remains OpenAI-specific, but the implementation becomes isolated from the large routing functions.

**Tech Stack:** Go, `testing`, existing `OpenAIGatewayService`, existing scheduler/load-awareness code paths, repo-local docs in `docs/plans`.

### Task 1: Lock the 112 contract into red tests

**Files:**
- Modify: `backend/internal/service/openai_over_limit_mode_test.go`
- Create: `backend/internal/service/openai_over_limit_strategy_test.go`

**Step 1: Write the failing test**

Add contract tests for:
- over-limit mode widens the candidate pool beyond repo/snapshot schedulable filtering
- active short cooldown suppresses immediate reuse of a rate-limited account
- sticky session fallback is bypassed when a higher-priority rate-limited account is eligible
- `previous_response_id` routing is neutralized when over-limit mode needs to pick from the broader pool

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run 'OpenAIOverLimit' -count=1`
Expected: FAIL on the new strategy-contract test(s) before implementation exists.

**Step 3: Commit**

```bash
git add backend/internal/service/openai_over_limit_mode_test.go backend/internal/service/openai_over_limit_strategy_test.go
git commit -m "test: lock openai over-limit strategy contract"
```

### Task 2: Extract settings/state/policy into a strategy layer

**Files:**
- Create: `backend/internal/service/openai_over_limit_strategy.go`
- Modify: `backend/internal/service/openai_gateway_service.go`
- Modify: `backend/internal/service/setting_service.go`

**Step 1: Write the failing test**

Add or extend tests so they construct the strategy through `OpenAIGatewayService` and assert:
- cooldown normalization still falls back to `10`
- strategy state uses account+model cooldown keys
- selection policy treats long upstream rate limit and short local cooldown as separate concerns

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run 'OpenAIOverLimit|ParseSettings_OpenAIOverLimitMode' -count=1`
Expected: FAIL because the new strategy helper and hook wiring do not exist yet.

**Step 3: Write minimal implementation**

Implement:
- strategy settings loader wrapper around existing setting service
- short-cooldown state helpers
- candidate-pool broadening helper
- account selectability helper
- sticky/previous-response bypass helper
- upstream `429/529` cooldown side-effect helper

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run 'OpenAIOverLimit|ParseSettings_OpenAIOverLimitMode' -count=1`
Expected: PASS

### Task 3: Rewire gateway and scheduler through 3 hook points

**Files:**
- Modify: `backend/internal/service/openai_gateway_service.go`
- Modify: `backend/internal/service/openai_account_scheduler.go`

**Step 1: Write the failing test**

Add or update focused tests for these hooks:
- candidate-source hook: scheduler/load-awareness can see broad over-limit candidates
- sticky hook: sticky or `previous_response_id` path is bypassed when strategy says so
- upstream error hook: failover/compat/passthrough responses record short cooldown through strategy

**Step 2: Run test to verify it fails**

Run: `go test ./internal/service -run 'SelectAccountWithScheduler|HandleFailoverSideEffects|HandleCompatErrorResponse' -count=1`
Expected: FAIL until the hooks call the extracted strategy.

**Step 3: Write minimal implementation**

Keep the hook surface small:
- `strategy.CandidateAccounts(...)`
- `strategy.ShouldBypassSticky(...)` / `strategy.ShouldIgnorePreviousResponse(...)`
- `strategy.HandleUpstreamError(...)`

**Step 4: Run test to verify it passes**

Run: `go test ./internal/service -run 'SelectAccountWithScheduler|HandleFailoverSideEffects|HandleCompatErrorResponse' -count=1`
Expected: PASS

### Task 4: Add merge documentation for future upstream syncs

**Files:**
- Create: `docs/plans/2026-04-24-openai-over-limit-merge-playbook.md`

**Step 1: Write the file**

Document:
- the 112 business contract we must preserve
- the exact 3 hook points
- where upstream changes are most likely to land
- what tests must be rerun after future merges

**Step 2: Verify file exists**

Run: `sed -n '1,220p' docs/plans/2026-04-24-openai-over-limit-merge-playbook.md`
Expected: shows the preserved contract and merge checklist.

### Task 5: Full verification

**Files:**
- Verify only

**Step 1: Run targeted service tests**

Run: `go test ./internal/service -run 'OpenAIOverLimit|SelectAccountWithScheduler|HandleFailoverSideEffects|HandleCompatErrorResponse' -count=1`
Expected: PASS

**Step 2: Run related server/settings regressions**

Run: `go test ./internal/service ./internal/server -count=1`
Expected: PASS

**Step 3: Commit**

```bash
git add backend/internal/service/openai_gateway_service.go backend/internal/service/openai_account_scheduler.go backend/internal/service/openai_over_limit_mode_test.go backend/internal/service/openai_over_limit_strategy.go backend/internal/service/openai_over_limit_strategy_test.go backend/internal/service/setting_service.go docs/plans/2026-04-24-openai-over-limit-strategy-layer.md docs/plans/2026-04-24-openai-over-limit-merge-playbook.md
git commit -m "refactor: isolate openai over-limit strategy"
```
