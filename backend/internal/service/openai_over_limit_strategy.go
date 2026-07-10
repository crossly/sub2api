package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type openAIOverLimitModeSettings struct {
	Enabled         bool
	CooldownSeconds int
}

type cachedOpenAIOverLimitModeSettings struct {
	settings  openAIOverLimitModeSettings
	expiresAt int64
}

const (
	openAIOverLimitSettingsCacheTTL   = 5 * time.Second
	openAIOverLimitSettingsDBTimeout  = 2 * time.Second
	minOpenAIOverLimitCooldownSeconds = 10
	maxOpenAIOverLimitCooldownSeconds = 300
)

type openAIOverLimitStrategy struct {
	service *OpenAIGatewayService
}

func (s *OpenAIGatewayService) openAIOverLimitStrategy() openAIOverLimitStrategy {
	return openAIOverLimitStrategy{service: s}
}

func normalizeOpenAIOverLimitCooldownSeconds(enabled bool, cooldown int) int {
	if !enabled {
		return 0
	}
	if cooldown < minOpenAIOverLimitCooldownSeconds {
		return minOpenAIOverLimitCooldownSeconds
	}
	if cooldown > maxOpenAIOverLimitCooldownSeconds {
		return maxOpenAIOverLimitCooldownSeconds
	}
	return cooldown
}

func normalizeOpenAIOverLimitModeSettings(settings openAIOverLimitModeSettings) openAIOverLimitModeSettings {
	settings.CooldownSeconds = normalizeOpenAIOverLimitCooldownSeconds(settings.Enabled, settings.CooldownSeconds)
	return settings
}

func (s *SettingService) getOpenAIOverLimitModeSettings(ctx context.Context) openAIOverLimitModeSettings {
	if s == nil || s.settingRepo == nil {
		return openAIOverLimitModeSettings{}
	}
	if cached, ok := s.openAIOverLimitSettingsCache.Load().(*cachedOpenAIOverLimitModeSettings); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.settings
		}
	}

	result, _, _ := s.openAIOverLimitSettingsSF.Do("openai_over_limit_settings", func() (any, error) {
		if cached, ok := s.openAIOverLimitSettingsCache.Load().(*cachedOpenAIOverLimitModeSettings); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.settings, nil
			}
		}
		version := s.openAIOverLimitSettingsVersion.Load()

		baseCtx := ctx
		if baseCtx == nil {
			baseCtx = context.Background()
		}
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(baseCtx), openAIOverLimitSettingsDBTimeout)
		defer cancel()
		values, err := s.settingRepo.GetMultiple(dbCtx, []string{
			SettingKeyOpenAIOverLimitModeEnabled,
			SettingKeyOpenAIOverLimitCooldownSeconds,
		})
		settings := openAIOverLimitModeSettings{}
		if err == nil {
			settings.Enabled = strings.EqualFold(strings.TrimSpace(values[SettingKeyOpenAIOverLimitModeEnabled]), "true")
			if seconds, convErr := strconv.Atoi(strings.TrimSpace(values[SettingKeyOpenAIOverLimitCooldownSeconds])); convErr == nil {
				settings.CooldownSeconds = seconds
			}
		}
		settings = normalizeOpenAIOverLimitModeSettings(settings)
		if s.openAIOverLimitSettingsVersion.Load() != version {
			if current, ok := s.openAIOverLimitSettingsCache.Load().(*cachedOpenAIOverLimitModeSettings); ok && current != nil {
				return current.settings, nil
			}
		}
		s.storeOpenAIOverLimitModeSettings(settings)
		return settings, nil
	})

	settings, _ := result.(openAIOverLimitModeSettings)
	return settings
}

func (s *SettingService) setOpenAIOverLimitModeSettings(settings openAIOverLimitModeSettings) {
	if s == nil {
		return
	}
	s.openAIOverLimitSettingsVersion.Add(1)
	s.storeOpenAIOverLimitModeSettings(settings)
	s.openAIOverLimitSettingsSF.Forget("openai_over_limit_settings")
}

func (s *SettingService) storeOpenAIOverLimitModeSettings(settings openAIOverLimitModeSettings) {
	settings = normalizeOpenAIOverLimitModeSettings(settings)
	s.openAIOverLimitSettingsCache.Store(&cachedOpenAIOverLimitModeSettings{
		settings:  settings,
		expiresAt: time.Now().Add(openAIOverLimitSettingsCacheTTL).UnixNano(),
	})
}

func (st openAIOverLimitStrategy) Settings(ctx context.Context) openAIOverLimitModeSettings {
	if st.service == nil || st.service.settingService == nil {
		return openAIOverLimitModeSettings{}
	}
	return st.service.settingService.getOpenAIOverLimitModeSettings(ctx)
}

func openAIAccountGlobalRateLimitActive(account *Account, now time.Time) bool {
	return account != nil && account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt)
}

func openAIOverLimitProbeReady(account *Account, settings openAIOverLimitModeSettings, now time.Time) bool {
	if !openAIAccountGlobalRateLimitActive(account, now) {
		return true
	}
	if !settings.Enabled {
		return false
	}
	if account.RateLimitedAt == nil {
		// A legacy row may have a reset timestamp without the observation time.
		// Allow one probe; a failed probe writes both timestamps atomically.
		return true
	}
	probeAt := account.RateLimitedAt.Add(time.Duration(settings.CooldownSeconds) * time.Second)
	return !now.Before(probeAt)
}

func (st openAIOverLimitStrategy) IsAccountSelectable(
	ctx context.Context,
	account *Account,
	requestedModel string,
	settings openAIOverLimitModeSettings,
) bool {
	if account == nil || !account.IsOpenAI() || !account.IsActive() || !account.Schedulable {
		return false
	}

	now := time.Now()
	if account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt) {
		return false
	}
	if account.OverloadUntil != nil && now.Before(*account.OverloadUntil) {
		return false
	}
	if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
		return false
	}
	if account.IsAPIKeyOrBedrock() && account.IsQuotaExceeded() {
		return false
	}
	if requestedModel != "" && !account.IsModelSupported(requestedModel) {
		return false
	}
	if account.isModelRateLimitedWithContext(ctx, requestedModel) {
		return false
	}

	globalRateLimited := openAIAccountGlobalRateLimitActive(account, now)
	if globalRateLimited && !openAIOverLimitProbeReady(account, settings, now) {
		return false
	}
	// A globally rate-limited account is intentionally probed in this mode. Quota
	// auto-pause is derived from the same stale usage snapshot and must not veto
	// that probe. Healthy accounts still respect the independent auto-pause policy.
	if !globalRateLimited {
		if paused, _ := shouldAutoPauseOpenAIAccountByQuota(ctx, account); paused {
			return false
		}
	}

	return true
}

func (st openAIOverLimitStrategy) CandidateAccounts(ctx context.Context, groupID *int64) ([]Account, error) {
	service := st.service
	if service == nil {
		return nil, nil
	}
	if service.schedulerSnapshot != nil {
		return service.schedulerSnapshot.ListOpenAIOverLimitAccounts(ctx, groupID)
	}
	if service.accountRepo == nil {
		return nil, nil
	}

	if service.cfg != nil && service.cfg.RunMode == config.RunModeSimple {
		accounts, err := service.accountRepo.ListByPlatform(ctx, PlatformOpenAI)
		if err != nil {
			return nil, fmt.Errorf("query openai accounts failed: %w", err)
		}
		return filterActiveOpenAIAccounts(accounts, false), nil
	}

	if groupID != nil {
		accounts, err := service.accountRepo.ListByGroup(ctx, *groupID)
		if err != nil {
			return nil, fmt.Errorf("query group accounts failed: %w", err)
		}
		return filterActiveOpenAIAccounts(accounts, false), nil
	}

	accounts, err := service.accountRepo.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return nil, fmt.Errorf("query openai accounts failed: %w", err)
	}
	return filterActiveOpenAIAccounts(accounts, true), nil
}

func (st openAIOverLimitStrategy) ShouldIgnorePreviousResponse(
	ctx context.Context,
	previousResponseID string,
	previousResponseCanMove bool,
) bool {
	if strings.TrimSpace(previousResponseID) == "" || !previousResponseCanMove {
		return false
	}
	return st.Settings(ctx).Enabled
}

func (st openAIOverLimitStrategy) ShouldBypassStickySession(
	ctx context.Context,
	groupID *int64,
	requestedModel string,
	excludedIDs map[int64]struct{},
	stickyAccount *Account,
	requireCompact bool,
) bool {
	if st.service == nil || stickyAccount == nil {
		return false
	}

	settings := st.Settings(ctx)
	if !settings.Enabled {
		return false
	}
	accounts, err := st.CandidateAccounts(ctx, groupID)
	if err != nil || len(accounts) == 0 {
		return false
	}

	now := time.Now()
	needsUpstreamCheck := st.service.needsUpstreamChannelRestrictionCheck(ctx, groupID)
	for i := range accounts {
		candidate := &accounts[i]
		if candidate.ID == stickyAccount.ID || candidate.Priority >= stickyAccount.Priority {
			continue
		}
		if excludedIDs != nil {
			if _, excluded := excludedIDs[candidate.ID]; excluded {
				continue
			}
		}
		if !openAIAccountGlobalRateLimitActive(candidate, now) {
			continue
		}
		if !st.IsAccountSelectable(ctx, candidate, requestedModel, settings) {
			continue
		}
		if requireCompact && openAICompactSupportTier(candidate) == 0 {
			continue
		}
		if needsUpstreamCheck && st.service.isUpstreamModelRestrictedByChannel(ctx, *groupID, candidate, requestedModel, requireCompact) {
			continue
		}
		return true
	}
	return false
}

func filterActiveOpenAIAccounts(accounts []Account, onlyUngrouped bool) []Account {
	if len(accounts) == 0 {
		return nil
	}
	result := make([]Account, 0, len(accounts))
	for _, account := range accounts {
		if account.Platform != PlatformOpenAI || !account.IsActive() {
			continue
		}
		if onlyUngrouped && len(account.GroupIDs) > 0 {
			continue
		}
		result = append(result, account)
	}
	return result
}
