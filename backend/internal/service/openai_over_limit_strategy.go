package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"golang.org/x/sync/singleflight"
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
)

var openAIOverLimitSettingsCache atomic.Value // *cachedOpenAIOverLimitModeSettings
var openAIOverLimitSettingsSF singleflight.Group

type openAIOverLimitStrategy struct {
	service *OpenAIGatewayService
}

func (s *OpenAIGatewayService) openAIOverLimitStrategy() openAIOverLimitStrategy {
	return openAIOverLimitStrategy{service: s}
}

func normalizeOpenAIOverLimitCooldownSeconds(enabled bool, cooldown int) int {
	if !enabled {
		return cooldown
	}
	if cooldown < minOpenAIOverLimitCooldownSeconds {
		return minOpenAIOverLimitCooldownSeconds
	}
	return cooldown
}

func normalizeOpenAIOverLimitModeSettings(settings openAIOverLimitModeSettings) openAIOverLimitModeSettings {
	settings.CooldownSeconds = normalizeOpenAIOverLimitCooldownSeconds(settings.Enabled, settings.CooldownSeconds)
	return settings
}

func normalizeOpenAIOverLimitCooldownModel(requestedModel string) string {
	trimmed := strings.TrimSpace(requestedModel)
	if trimmed == "" {
		return "*"
	}
	return NormalizeOpenAICompatRequestedModel(trimmed)
}

func openAIOverLimitCooldownKey(accountID int64, requestedModel string) string {
	return fmt.Sprintf("%d:%s", accountID, normalizeOpenAIOverLimitCooldownModel(requestedModel))
}

func openAIRequestedModelFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	model, _ := ctx.Value(ctxkey.Model).(string)
	return strings.TrimSpace(model)
}

func (st openAIOverLimitStrategy) effectiveRequestedModel(ctx context.Context, requestedModel string) string {
	if trimmed := strings.TrimSpace(requestedModel); trimmed != "" {
		return trimmed
	}
	return openAIRequestedModelFromContext(ctx)
}

func (st openAIOverLimitStrategy) Settings(ctx context.Context) openAIOverLimitModeSettings {
	if cached, ok := openAIOverLimitSettingsCache.Load().(*cachedOpenAIOverLimitModeSettings); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.settings
		}
	}

	result, _, _ := openAIOverLimitSettingsSF.Do("openai_over_limit_settings", func() (any, error) {
		if cached, ok := openAIOverLimitSettingsCache.Load().(*cachedOpenAIOverLimitModeSettings); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.settings, nil
			}
		}

		settings := openAIOverLimitModeSettings{}
		if service := st.service; service != nil {
			if repo := service.openAIOverLimitSettingRepo(); repo != nil {
				dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), openAIOverLimitSettingsDBTimeout)
				defer cancel()

				if value, err := repo.GetValue(dbCtx, SettingKeyOpenAIOverLimitModeEnabled); err == nil {
					settings.Enabled = strings.EqualFold(strings.TrimSpace(value), "true")
				}
				if value, err := repo.GetValue(dbCtx, SettingKeyOpenAIOverLimitCooldownSeconds); err == nil {
					if seconds, convErr := strconv.Atoi(strings.TrimSpace(value)); convErr == nil {
						settings.CooldownSeconds = seconds
					}
				}
			}
		}

		settings = normalizeOpenAIOverLimitModeSettings(settings)
		openAIOverLimitSettingsCache.Store(&cachedOpenAIOverLimitModeSettings{
			settings:  settings,
			expiresAt: time.Now().Add(openAIOverLimitSettingsCacheTTL).UnixNano(),
		})
		return settings, nil
	})

	settings, _ := result.(openAIOverLimitModeSettings)
	return settings
}

func (st openAIOverLimitStrategy) MarkCooldown(accountID int64, requestedModel string, cooldown time.Duration) {
	if st.service == nil || accountID <= 0 || cooldown <= 0 {
		return
	}
	st.service.openAIOverLimitUntil.Store(openAIOverLimitCooldownKey(accountID, requestedModel), time.Now().Add(cooldown))
}

func (st openAIOverLimitStrategy) IsCooldownActive(accountID int64, requestedModel string, now time.Time) bool {
	if st.service == nil || accountID <= 0 {
		return false
	}

	key := openAIOverLimitCooldownKey(accountID, requestedModel)
	untilValue, ok := st.service.openAIOverLimitUntil.Load(key)
	if !ok {
		return false
	}
	until, ok := untilValue.(time.Time)
	if !ok {
		st.service.openAIOverLimitUntil.Delete(key)
		return false
	}
	if now.Before(until) {
		return true
	}
	st.service.openAIOverLimitUntil.Delete(key)
	return false
}

func (st openAIOverLimitStrategy) IsAccountSelectable(account *Account, requestedModel string, settings openAIOverLimitModeSettings) bool {
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
	if account.RateLimitResetAt != nil && now.Before(*account.RateLimitResetAt) {
		if !settings.Enabled {
			return false
		}
		if st.IsCooldownActive(account.ID, requestedModel, now) {
			return false
		}
	}

	return true
}

func (st openAIOverLimitStrategy) CandidateAccounts(ctx context.Context, groupID *int64) ([]Account, error) {
	service := st.service
	if service == nil || service.accountRepo == nil {
		if service != nil && service.schedulerSnapshot != nil {
			accounts, _, err := service.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, PlatformOpenAI, false)
			return accounts, err
		}
		return nil, nil
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
	onlyUngrouped := service.cfg == nil || service.cfg.RunMode != config.RunModeSimple
	return filterActiveOpenAIAccounts(accounts, onlyUngrouped), nil
}

func (st openAIOverLimitStrategy) ShouldIgnorePreviousResponse(ctx context.Context, previousResponseID string) bool {
	if strings.TrimSpace(previousResponseID) == "" {
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
		if candidate.ID == stickyAccount.ID {
			continue
		}
		if excludedIDs != nil {
			if _, excluded := excludedIDs[candidate.ID]; excluded {
				continue
			}
		}
		if candidate.Priority >= stickyAccount.Priority {
			continue
		}
		if candidate.RateLimitResetAt == nil || !now.Before(*candidate.RateLimitResetAt) {
			continue
		}
		if !st.IsAccountSelectable(candidate, requestedModel, settings) {
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

func (st openAIOverLimitStrategy) HandleUpstreamError(
	ctx context.Context,
	account *Account,
	requestedModel string,
	statusCode int,
) {
	if st.service == nil || account == nil || !account.IsOpenAI() {
		return
	}

	switch statusCode {
	case http.StatusTooManyRequests, 529:
	default:
		return
	}

	settings := st.Settings(ctx)
	if !settings.Enabled || settings.CooldownSeconds <= 0 {
		return
	}

	st.MarkCooldown(
		account.ID,
		st.effectiveRequestedModel(ctx, requestedModel),
		time.Duration(settings.CooldownSeconds)*time.Second,
	)
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
