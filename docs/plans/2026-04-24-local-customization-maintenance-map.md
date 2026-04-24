# Local Customization Maintenance Map

**Baseline:** upstream `v0.1.116` (`a22a5b9e`)

**Purpose:** inventory local customizations that are not part of the OpenAI over-limit strategy layer, so future upstream merges can keep `overlimit-only` isolated and reapply branding/deploy changes in smaller batches.

## Branch Notes

- `codex/overlimit-only` exists as the logical branch for the isolated over-limit strategy work.
- As of 2026-04-24, `main` and `codex/overlimit-only` both point to `22f0ea25`.
- This is still usable for future merges, but treat `codex/overlimit-only` as a long-lived maintenance branch and avoid deleting or force-moving it.
- If a future merge needs a stricter upstream-only patch branch, cut a fresh branch from upstream tag `a22a5b9e` (or the new upstream release tag) and cherry-pick only the over-limit commits.

## Review Findings

### 1. Locale files still mix branding copy and over-limit copy

**Files:**
- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`

**Why it matters:**
- The same locale files currently contain both homepage branding copy and OpenAI over-limit admin copy.
- Future upstream changes to either homepage copy or admin settings text will increase merge conflict surface for the other concern.

**Maintenance rule:**
- When editing homepage branding next time, keep changes limited to the `home.*` keys.
- When editing over-limit behavior next time, keep changes limited to the `admin.settings.openaiOverLimitMode.*` keys.
- Do not mix the two concerns again in the same follow-up commit unless strictly necessary.

### 2. SVG logo is live, but legacy PNG references still exist

**Files already migrated:**
- `frontend/index.html`
- `frontend/public/logo.svg`
- `frontend/src/views/HomeView.vue`
- `frontend/src/components/layout/AppSidebar.vue`
- `frontend/src/components/layout/AuthLayout.vue`
- `frontend/src/views/KeyUsageView.vue`

**Legacy references still present:**
- `deploy/Caddyfile`
- `frontend/src/stores/__tests__/app.spec.ts`
- `backend/internal/web/embed_test.go`

**Why it matters:**
- The runtime fallback logo has moved to `/logo.svg`, and `frontend/public/logo.png` no longer exists.
- Remaining `/logo.png` references are now historical compatibility debt and should be cleaned up or intentionally preserved by restoring a PNG alias.

**Maintenance rule:**
- Pick one of these and stay consistent:
- Option A: keep SVG-only and update remaining `/logo.png` references.
- Option B: restore a generated PNG alias and intentionally support both `/logo.svg` and `/logo.png`.

### 3. `HomeView.vue` is now both branding layer and responsive-nav layer

**File:**
- `frontend/src/views/HomeView.vue`

**Why it matters:**
- The file now carries the purple landing hero, provider presentation, CTA layout, and the mobile dropdown navigation behavior.
- Upstream homepage adjustments and local branding tweaks will now collide in one large file.

**Maintenance rule:**
- Treat `HomeView.vue` as a local customization hotspot.
- If it changes again, consider the next split as:
- `HomeHero`
- `HomeNav`
- `HomeProviderCard`

## Non-Overlimit Customization Inventory

### 1. Deploy / Build / Runtime

**Files:**
- `Dockerfile`
- `frontend/package.json`
- `deploy/docker-entrypoint.sh`
- `docker-compose.coolify.yml`
- `backend/cmd/server/VERSION`

**What changed:**
- Docker build now pins `pnpm` to `10.33.0`.
- Frontend install/build uses BuildKit cache mounts.
- Docker build uses `pnpm run build:docker` to skip `vue-tsc` inside container builds.
- Go module download and Go build use cache mounts for better rebuild performance.
- Runtime entrypoint prints image ref and binary version at startup.
- Coolify compose now pulls `ghcr.io/crossly/sub2api:latest` instead of building on the host.
- Coolify compose includes the high-value gateway, Redis, and DB timeout/pool knobs that were kept after cleanup.
- Local app version string is `1.116`, not upstream `0.1.116`.

**How to maintain it:**
- Reapply this group together after upstream Docker or deployment changes.
- Keep it separate from UI or over-limit commits.
- If upstream changes Docker build stages again, reconcile `build:docker`, `PNPM_VERSION`, and cache mounts together in one patch.

### 2. Branding / Theme System

**Files:**
- `frontend/tailwind.config.js`
- `frontend/src/components/layout/AuthLayout.vue`
- `frontend/src/views/admin/DashboardView.vue`
- `frontend/src/components/charts/ModelDistributionChart.vue`
- `frontend/src/components/charts/TokenUsageTrend.vue`
- `frontend/src/components/charts/UserBreakdownSubTable.vue`

**What changed:**
- Primary palette moved from teal/green to business purple.
- Glow, gradient, mesh background, and dark-mode accents were updated to match the new palette.
- Auth pages now use the purple light/dark background system instead of the old green-tinted background.
- Dashboard cards and chart colors were tuned to remove lingering green bias in dark mode.

**How to maintain it:**
- Treat this as one coherent theme layer.
- If upstream updates component classes, reapply color-token changes first and only re-tune per-component accents when needed.

### 3. Logo / Favicon / Brand Mark

**Files:**
- `frontend/index.html`
- `frontend/public/logo.svg`
- `frontend/src/views/HomeView.vue`
- `frontend/src/components/layout/AppSidebar.vue`
- `frontend/src/components/layout/AuthLayout.vue`
- `frontend/src/views/KeyUsageView.vue`

**What changed:**
- Default fallback brand asset changed from PNG to SVG favicon.
- New fallback logo is the purple `OI` mark for `OINANCE`.
- Layout fallbacks now prefer `/logo.svg`.

**How to maintain it:**
- Keep logo asset swaps separate from theme token changes when possible.
- If a future brand refresh happens, only touch the asset file and the fallback references first.

### 4. Landing Page / Hero / Messaging

**Files:**
- `frontend/src/views/HomeView.vue`
- `frontend/src/i18n/locales/zh.ts`
- `frontend/src/i18n/locales/en.ts`

**What changed:**
- Default homepage was reduced to a hero-driven landing layout.
- Brand tone changed to:
- `AI API Gateway`
- `Native Direct`
- `Long-Term Stable`
- Chinese copy changed to:
- `AI API 网关`
- `原生直连`
- `长效稳定`
- Provider row now explicitly includes `ChatGPT` first.
- Header actions were adapted for mobile dropdown behavior.

**How to maintain it:**
- Keep homepage structure and homepage copy changes together.
- Keep provider-order changes with homepage branding, not with generic i18n cleanup.

### 5. App Shell / Responsive UX

**Files:**
- `frontend/src/components/layout/AppHeader.vue`
- `frontend/src/components/layout/__tests__/app-header-responsive.spec.ts`
- `frontend/src/views/__tests__/home-header-responsive.spec.ts`

**What changed:**
- App header uses tighter spacing and min-width guards to reduce squeezing on mobile.
- Docs link and subscription mini-card now hide earlier on narrow screens.
- Homepage mobile actions use a dropdown menu instead of a large expanded action block.
- Snapshot-style guard tests were added to protect these responsive choices.

**How to maintain it:**
- Keep responsive layout tweaks paired with their source-based guard tests.
- If upstream rewrites these templates, re-evaluate the tests instead of blindly preserving class strings.

### 6. Email Template Branding

**Files:**
- `backend/internal/service/email_service.go`
- `backend/internal/service/email_templates.go`

**What changed:**
- Verify-code email template moved out of inline HTML.
- Password-reset email template moved out of inline HTML.
- SMTP test email now shares the same branded rendering system.
- Email subject lines and content were localized and styled with the purple business visual language.

**How to maintain it:**
- Keep template rendering changes in `email_templates.go`.
- Keep send-flow logic changes in `email_service.go`.
- Avoid moving branded HTML back into service methods.

### 7. Small Compatibility / Test Tweaks

**Files:**
- `frontend/src/components/account/__tests__/AccountStatusIndicator.spec.ts`

**What changed:**
- Added `localStorage` hoist stub for the test environment.
- Updated assertions to the current translation key path.

**How to maintain it:**
- Treat this as a compatibility patch, not as part of theming.
- Re-check it only when account status rendering or i18n key paths change.

## Recommended Maintenance Buckets

For future upstream merges, keep local changes mentally grouped as these buckets:

1. `overlimit-only`
2. `deploy-coolify-runtime`
3. `branding-theme`
4. `logo-favicon`
5. `landing-hero-copy`
6. `responsive-shell`
7. `email-branding`

## Reapply Order After Future Upstream Merge

1. Merge upstream release into your integration branch.
2. Reapply `deploy-coolify-runtime`.
3. Reapply `branding-theme`.
4. Reapply `logo-favicon`.
5. Reapply `landing-hero-copy`.
6. Reapply `responsive-shell`.
7. Reapply `email-branding`.
8. Reapply `overlimit-only` last and validate with the dedicated merge playbook.

## Companion Docs

- `docs/plans/2026-04-24-openai-over-limit-strategy-layer.md`
- `docs/plans/2026-04-24-openai-over-limit-merge-playbook.md`
