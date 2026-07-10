package service

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const (
	openAIAccountStateUpdateTimeout       = 5 * time.Second
	openAIOAuth429FallbackCooldown        = 5 * time.Second
	openAIStopSchedulingBridgeCooldown    = 2 * time.Minute
	openAIOAuth429StormWindow             = 10 * time.Second
	openAIOAuth429StormThreshold          = 20
	openAIOAuth429StormMaxAccountSwitches = 1
)

type openAIAccountRuntimeBlock struct {
	HardUntil      time.Time
	HardReason     string
	RateLimitedAt  time.Time
	RateLimitUntil time.Time
}

func openAIAccountStateContext(ctx context.Context) (context.Context, context.CancelFunc) {
	base := context.Background()
	if ctx != nil {
		base = context.WithoutCancel(ctx)
	}
	return context.WithTimeout(base, openAIAccountStateUpdateTimeout)
}

func isOpenAIOAuthAccount(account *Account) bool {
	return account != nil && account.Platform == PlatformOpenAI && account.Type == AccountTypeOAuth
}

func isGrokOAuthAccount(account *Account) bool {
	return account != nil && account.Platform == PlatformGrok && account.Type == AccountTypeOAuth
}

func isOpenAIAccount(account *Account) bool {
	return account != nil && (account.Platform == PlatformOpenAI || account.Platform == PlatformGrok)
}

func (s *OpenAIGatewayService) handleOpenAIAccountUpstreamError(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte, requestedModel ...string) bool {
	stateCtx, cancel := openAIAccountStateContext(ctx)
	defer cancel()

	if account != nil && account.Platform == PlatformOpenAI && isOpenAIContextWindowError("", responseBody) {
		return false
	}

	if isOpenAIImageRateLimitError(statusCode, responseBody) {
		if s != nil && s.rateLimitService != nil {
			_ = s.rateLimitService.HandleOpenAIImageRateLimit(stateCtx, account, statusCode, headers, responseBody)
		}
		return false
	}

	if statusCode == http.StatusTooManyRequests {
		s.markOpenAIOAuth429RateLimited(stateCtx, account, headers, responseBody)
	}
	if s == nil || account == nil || s.rateLimitService == nil {
		return false
	}
	if len(requestedModel) > 0 && s.rateLimitService.HandleUpstreamModelNotFound(stateCtx, account, requestedModel[0], statusCode, responseBody) {
		return true
	}
	shouldDisable := s.rateLimitService.HandleUpstreamError(stateCtx, account, statusCode, headers, responseBody)
	if shouldDisable {
		s.BlockAccountScheduling(account, time.Time{}, "upstream_disable")
	}
	return shouldDisable
}

func (s *OpenAIGatewayService) markOpenAIOAuth429RateLimited(ctx context.Context, account *Account, headers http.Header, responseBody []byte) {
	if s == nil || !isOpenAIOAuthAccount(account) {
		return
	}
	// Spark 影子：不按 /responses 429 的 global x-codex-* 信号做内存运行时熔断(同 handle429,外审第8轮 P1)。
	// 同时避免把 spark 的 429 计入全局 429 storm 计数(recordOpenAIOAuth429),否则会误伤母账号 failover 决策。
	if account.IsShadow() {
		return
	}
	s.recordOpenAIOAuth429()

	cooldownUntil := time.Now().Add(openAIOAuth429FallbackCooldown)
	if s.rateLimitService != nil {
		if resetAt := s.rateLimitService.calculateOpenAI429ResetTime(headers); resetAt != nil && resetAt.After(time.Now()) {
			cooldownUntil = *resetAt
		} else if resetUnix := parseOpenAIRateLimitResetTime(responseBody); resetUnix != nil {
			if resetAt := time.Unix(*resetUnix, 0); resetAt.After(time.Now()) {
				cooldownUntil = resetAt
			}
		} else if cooldown, ok := s.rateLimitService.get429FallbackCooldown(ctx, account); ok && cooldown > 0 {
			cooldownUntil = time.Now().Add(cooldown)
		}
	}
	s.BlockAccountScheduling(account, cooldownUntil, "429")
}

func (s *OpenAIGatewayService) BlockAccountScheduling(account *Account, until time.Time, reason string) {
	if s == nil || !isOpenAIAccount(account) {
		return
	}
	now := time.Now()
	blockUntil := until
	if blockUntil.IsZero() || !blockUntil.After(now) {
		blockUntil = now.Add(openAIStopSchedulingBridgeCooldown)
	}
	reason = strings.TrimSpace(reason)
	isRateLimit := isOpenAIRateLimitRuntimeBlockReason(reason)

	for {
		currentValue, loaded := s.openaiAccountRuntimeBlockUntil.Load(account.ID)
		if !loaded {
			block := openAIAccountRuntimeBlock{}
			if isRateLimit {
				block.RateLimitedAt = now
				block.RateLimitUntil = blockUntil
			} else {
				block.HardUntil = blockUntil
				block.HardReason = reason
			}
			actual, stored := s.openaiAccountRuntimeBlockUntil.LoadOrStore(account.ID, block)
			if !stored {
				return
			}
			currentValue = actual
		}

		currentBlock, ok := currentValue.(openAIAccountRuntimeBlock)
		if !ok {
			currentBlock = openAIAccountRuntimeBlock{}
		}
		next := currentBlock
		if isRateLimit {
			next.RateLimitedAt = now
			if next.RateLimitUntil.Before(blockUntil) {
				next.RateLimitUntil = blockUntil
			}
		} else if next.HardUntil.Before(blockUntil) {
			next.HardUntil = blockUntil
			next.HardReason = reason
		}
		if next == currentBlock {
			return
		}
		if s.openaiAccountRuntimeBlockUntil.CompareAndSwap(account.ID, currentValue, next) {
			return
		}
	}
}

func isOpenAIRateLimitRuntimeBlockReason(reason string) bool {
	switch reason {
	case "429", "429_fallback":
		return true
	default:
		return false
	}
}

func (s *OpenAIGatewayService) ClearAccountSchedulingBlock(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.openaiAccountRuntimeBlockUntil.Delete(accountID)
}

func (s *OpenAIGatewayService) isOpenAIAccountRuntimeBlocked(account *Account) bool {
	if s == nil || !isOpenAIAccount(account) {
		return false
	}
	value, ok := s.openaiAccountRuntimeBlockUntil.Load(account.ID)
	if !ok {
		return false
	}
	block, ok := value.(openAIAccountRuntimeBlock)
	if !ok {
		s.openaiAccountRuntimeBlockUntil.Delete(account.ID)
		return false
	}
	now := time.Now()
	if !block.HardUntil.IsZero() && now.Before(block.HardUntil) {
		return true
	}

	settings := openAIOverLimitModeSettings{}
	if account.IsOpenAI() {
		settings = s.getOpenAIOverLimitModeSettings(context.Background())
	}
	if settings.Enabled && !block.RateLimitedAt.IsZero() {
		probeAt := block.RateLimitedAt.Add(time.Duration(settings.CooldownSeconds) * time.Second)
		if now.Before(probeAt) {
			return true
		}
	} else if !block.RateLimitUntil.IsZero() && now.Before(block.RateLimitUntil) {
		return true
	}

	rateLimitRetentionUntil := block.RateLimitedAt.Add(time.Duration(maxOpenAIOverLimitCooldownSeconds) * time.Second)
	if (block.HardUntil.IsZero() || !now.Before(block.HardUntil)) &&
		(block.RateLimitUntil.IsZero() || !now.Before(block.RateLimitUntil)) &&
		(block.RateLimitedAt.IsZero() || !now.Before(rateLimitRetentionUntil)) {
		s.openaiAccountRuntimeBlockUntil.Delete(account.ID)
	}
	return false
}

func (s *OpenAIGatewayService) recordOpenAIOAuth429() {
	if s == nil {
		return
	}
	now := time.Now()
	windowStart := s.openaiOAuth429WindowStartUnixNano.Load()
	if windowStart == 0 || now.Sub(time.Unix(0, windowStart)) >= openAIOAuth429StormWindow {
		if s.openaiOAuth429WindowStartUnixNano.CompareAndSwap(windowStart, now.UnixNano()) {
			s.openaiOAuth429WindowCount.Store(1)
			return
		}
	}
	s.openaiOAuth429WindowCount.Add(1)
}

func (s *OpenAIGatewayService) isOpenAIOAuth429Storm() bool {
	if s == nil {
		return false
	}
	windowStart := s.openaiOAuth429WindowStartUnixNano.Load()
	if windowStart == 0 || time.Since(time.Unix(0, windowStart)) >= openAIOAuth429StormWindow {
		return false
	}
	return s.openaiOAuth429WindowCount.Load() >= openAIOAuth429StormThreshold
}

func (s *OpenAIGatewayService) ShouldStopOpenAIOAuth429Failover(account *Account, statusCode int, failedSwitches int) bool {
	if statusCode != http.StatusTooManyRequests || failedSwitches < openAIOAuth429StormMaxAccountSwitches {
		return false
	}
	if s.getOpenAIOverLimitModeSettings(context.Background()).Enabled {
		return false
	}
	if isGrokOAuthAccount(account) {
		return true
	}
	if !isOpenAIOAuthAccount(account) {
		return false
	}
	return s.isOpenAIOAuth429Storm()
}
