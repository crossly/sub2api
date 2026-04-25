package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIOverLimitSettingsRepoStub struct {
	values map[string]string
}

type openAIOverLimitAccountRepoStub struct {
	stubOpenAIAccountRepo
}

type productionLikeOpenAIOverLimitRepoStub struct {
	stubOpenAIAccountRepo
}

func newOpenAIOverLimitSettingsRepoStub(enabled bool, cooldownSeconds int) *openAIOverLimitSettingsRepoStub {
	return &openAIOverLimitSettingsRepoStub{
		values: map[string]string{
			SettingKeyOpenAIOverLimitModeEnabled:     strconv.FormatBool(enabled),
			SettingKeyOpenAIOverLimitCooldownSeconds: strconv.Itoa(cooldownSeconds),
		},
	}
}

func (s *openAIOverLimitSettingsRepoStub) Get(_ context.Context, key string) (*Setting, error) {
	if value, ok := s.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (s *openAIOverLimitSettingsRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *openAIOverLimitSettingsRepoStub) Set(_ context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *openAIOverLimitSettingsRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (s *openAIOverLimitSettingsRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = make(map[string]string, len(settings))
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *openAIOverLimitSettingsRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	result := make(map[string]string, len(s.values))
	for key, value := range s.values {
		result[key] = value
	}
	return result, nil
}

func (s *openAIOverLimitSettingsRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func (r openAIOverLimitAccountRepoStub) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.ListSchedulableByPlatform(ctx, platform)
}

func (r openAIOverLimitAccountRepoStub) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	var result []Account
	result = append(result, r.accounts...)
	return result, nil
}

func (r productionLikeOpenAIOverLimitRepoStub) ListSchedulableByGroupIDAndPlatform(ctx context.Context, groupID int64, platform string) ([]Account, error) {
	now := time.Now()
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform != platform {
			continue
		}
		if acc.RateLimitResetAt != nil && now.Before(*acc.RateLimitResetAt) {
			continue
		}
		result = append(result, acc)
	}
	return result, nil
}

func (r productionLikeOpenAIOverLimitRepoStub) ListSchedulableByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.ListSchedulableByGroupIDAndPlatform(ctx, 0, platform)
}

func (r productionLikeOpenAIOverLimitRepoStub) ListSchedulableUngroupedByPlatform(ctx context.Context, platform string) ([]Account, error) {
	return r.ListSchedulableByPlatform(ctx, platform)
}

func (r productionLikeOpenAIOverLimitRepoStub) ListByGroup(ctx context.Context, groupID int64) ([]Account, error) {
	var result []Account
	result = append(result, r.accounts...)
	return result, nil
}

func (r productionLikeOpenAIOverLimitRepoStub) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	var result []Account
	for _, acc := range r.accounts {
		if acc.Platform == platform {
			result = append(result, acc)
		}
	}
	return result, nil
}

func newOpenAIOverLimitSettingServiceForTest(t *testing.T, enabled bool, cooldownSeconds int) *SettingService {
	t.Helper()
	openAIOverLimitSettingsCache = atomic.Value{}
	t.Cleanup(func() {
		openAIOverLimitSettingsCache = atomic.Value{}
	})
	return NewSettingService(newOpenAIOverLimitSettingsRepoStub(enabled, cooldownSeconds), &config.Config{})
}

func newOpenAIOverLimitSettingServiceWithValuesForTest(t *testing.T, values map[string]string) *SettingService {
	t.Helper()
	openAIOverLimitSettingsCache = atomic.Value{}
	t.Cleanup(func() {
		openAIOverLimitSettingsCache = atomic.Value{}
	})
	repo := &openAIOverLimitSettingsRepoStub{values: map[string]string{}}
	for key, value := range values {
		repo.values[key] = value
	}
	return NewSettingService(repo, &config.Config{})
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_OpenAIOverLimitModeAllowsRateLimitedAccountAfterCooldown(t *testing.T) {
	ctx := context.Background()
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	account := Account{
		ID:               41001,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         0,
		RateLimitResetAt: &rateLimitedUntil,
	}

	svc := &OpenAIGatewayService{
		accountRepo: openAIOverLimitAccountRepoStub{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}}},
		cfg:         &config.Config{},
		schedulerSnapshot: &SchedulerSnapshotService{cache: &openAISnapshotCacheStub{
			snapshotAccounts: []*Account{&account},
			accountsByID: map[int64]*Account{
				account.ID: &account,
			},
		}},
	}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, true, 10))

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		nil,
		"resp_prev_41001",
		"session_hash_41001",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, account.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_RecheckSelectedOpenAIAccountFromDB_OpenAIOverLimitModeAllowsRateLimitedAccountAfterCooldown(t *testing.T) {
	ctx := context.Background()
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	dbAccount := Account{
		ID:               42001,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         0,
		RateLimitResetAt: &rateLimitedUntil,
	}

	svc := &OpenAIGatewayService{
		accountRepo:       stubOpenAIAccountRepo{accounts: []Account{dbAccount}},
		cfg:               &config.Config{},
		schedulerSnapshot: &SchedulerSnapshotService{},
	}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, true, 10))

	got := svc.recheckSelectedOpenAIAccountFromDB(ctx, &Account{ID: dbAccount.ID, Platform: PlatformOpenAI}, "gpt-5.1", false)
	require.NotNil(t, got)
	require.Equal(t, dbAccount.ID, got.ID)
}

func TestOpenAIGatewayService_GetSchedulableAccount_OpenAIOverLimitModeSkipsActiveShortCooldown(t *testing.T) {
	ctx := context.Background()
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	account := Account{
		ID:               43001,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         0,
		RateLimitResetAt: &rateLimitedUntil,
	}

	svc := &OpenAIGatewayService{
		accountRepo: stubOpenAIAccountRepo{accounts: []Account{account}},
		cfg:         &config.Config{},
	}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, true, 10))
	svc.markOpenAIOverLimitCooldown(account.ID, "", time.Minute)

	got, err := svc.getSchedulableAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_AdvancedSchedulerAllowsRateLimitedAccountInOpenAIOverLimitMode(t *testing.T) {
	ctx := context.Background()
	groupID := int64(51001)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	primary := Account{
		ID:               51001,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         0,
		RateLimitResetAt: &rateLimitedUntil,
	}
	backup := Account{
		ID:          51002,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    5,
	}

	settingService := newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
		openAIAdvancedSchedulerSettingKey:        "true",
		SettingKeyOpenAIOverLimitModeEnabled:     "true",
		SettingKeyOpenAIOverLimitCooldownSeconds: "15",
	})
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.LBTopK = 1
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{primary, backup}},
		cfg:                cfg,
		rateLimitService:   &RateLimitService{settingService: settingService},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	svc.SetSettingService(settingService)

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, primary.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_AdvancedSchedulerUsesBroadAccountListWhenOverLimitEnabled(t *testing.T) {
	ctx := context.Background()
	groupID := int64(51101)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	primary := Account{
		ID:               51101,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         1,
		RateLimitResetAt: &rateLimitedUntil,
	}
	backup := Account{
		ID:          51102,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    11,
	}

	settingService := newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
		openAIAdvancedSchedulerSettingKey:        "true",
		SettingKeyOpenAIOverLimitModeEnabled:     "true",
		SettingKeyOpenAIOverLimitCooldownSeconds: "15",
	})
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.LBTopK = 1
	svc := &OpenAIGatewayService{
		accountRepo:        productionLikeOpenAIOverLimitRepoStub{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{primary, backup}}},
		cfg:                cfg,
		rateLimitService:   &RateLimitService{settingService: settingService},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	svc.SetSettingService(settingService)

	selection, _, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, primary.ID, selection.Account.ID)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_LoadAwarenessUsesBroadAccountListWhenOverLimitEnabled(t *testing.T) {
	ctx := context.Background()
	groupID := int64(51111)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	primary := Account{
		ID:               51111,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         1,
		RateLimitResetAt: &rateLimitedUntil,
	}
	backup := Account{
		ID:          51112,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    11,
	}

	settingService := newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
		openAIAdvancedSchedulerSettingKey:        "false",
		SettingKeyOpenAIOverLimitModeEnabled:     "true",
		SettingKeyOpenAIOverLimitCooldownSeconds: "15",
	})
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	svc := &OpenAIGatewayService{
		accountRepo:        productionLikeOpenAIOverLimitRepoStub{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{primary, backup}}},
		cfg:                &config.Config{},
		rateLimitService:   &RateLimitService{settingService: settingService},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	svc.SetSettingService(settingService)

	selection, _, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, primary.ID, selection.Account.ID)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_OpenAIOverLimitModeBypassesStickyFallbackAccount(t *testing.T) {
	testCases := []struct {
		name              string
		advancedScheduler bool
	}{
		{
			name:              "load awareness",
			advancedScheduler: false,
		},
		{
			name:              "advanced scheduler",
			advancedScheduler: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			groupID := int64(51121)
			rateLimitedUntil := time.Now().Add(30 * time.Minute)
			primary := Account{
				ID:               51121,
				Platform:         PlatformOpenAI,
				Type:             AccountTypeOAuth,
				Status:           StatusActive,
				Schedulable:      true,
				Concurrency:      1,
				Priority:         1,
				RateLimitResetAt: &rateLimitedUntil,
			}
			backup := Account{
				ID:          51122,
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 1,
				Priority:    11,
			}

			settingService := newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
				openAIAdvancedSchedulerSettingKey:        strconv.FormatBool(tt.advancedScheduler),
				SettingKeyOpenAIOverLimitModeEnabled:     "true",
				SettingKeyOpenAIOverLimitCooldownSeconds: "15",
			})
			resetOpenAIAdvancedSchedulerSettingCacheForTest()

			cache := &schedulerTestGatewayCache{
				sessionBindings: map[string]int64{
					"openai:session_hash_over_limit_backup": backup.ID,
				},
			}
			svc := &OpenAIGatewayService{
				accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{primary, backup}},
				cache:              cache,
				cfg:                &config.Config{},
				rateLimitService:   &RateLimitService{settingService: settingService},
				concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
			}
			svc.SetSettingService(settingService)

			selection, _, err := svc.SelectAccountWithScheduler(
				ctx,
				&groupID,
				"",
				"session_hash_over_limit_backup",
				"gpt-5.1",
				nil,
				OpenAIUpstreamTransportAny,
				false,
			)
			require.NoError(t, err)
			require.NotNil(t, selection)
			require.NotNil(t, selection.Account)
			require.Equal(t, primary.ID, selection.Account.ID)
		})
	}
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_OpenAIOverLimitModeDoesNotBypassStickyForUnsupportedCompactCandidate(t *testing.T) {
	ctx := context.Background()
	groupID := int64(51125)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	primary := Account{
		ID:               51125,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeAPIKey,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         1,
		RateLimitResetAt: &rateLimitedUntil,
		Extra:            map[string]any{"openai_compact_supported": false},
	}
	backup := Account{
		ID:          51126,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    11,
		Extra:       map[string]any{"openai_compact_supported": true},
	}

	settingService := newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
		openAIAdvancedSchedulerSettingKey:        "true",
		SettingKeyOpenAIOverLimitModeEnabled:     "true",
		SettingKeyOpenAIOverLimitCooldownSeconds: "15",
	})
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	cache := &schedulerTestGatewayCache{
		sessionBindings: map[string]int64{
			"openai:session_hash_over_limit_compact_backup": backup.ID,
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{primary, backup}},
		cache:              cache,
		cfg:                &config.Config{},
		rateLimitService:   &RateLimitService{settingService: settingService},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	svc.SetSettingService(settingService)

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"session_hash_over_limit_compact_backup",
		"gpt-5.4",
		nil,
		OpenAIUpstreamTransportAny,
		true,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, backup.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerSessionSticky, decision.Layer)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_OpenAIOverLimitModeBypassesPreviousResponseFallbackAccount(t *testing.T) {
	ctx := context.Background()
	groupID := int64(51131)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	primary := Account{
		ID:               51131,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeAPIKey,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         1,
		RateLimitResetAt: &rateLimitedUntil,
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_enabled": true,
		},
	}
	backup := Account{
		ID:          51132,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    11,
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_enabled": true,
		},
	}

	settingService := newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
		openAIAdvancedSchedulerSettingKey:        "true",
		SettingKeyOpenAIOverLimitModeEnabled:     "true",
		SettingKeyOpenAIOverLimitCooldownSeconds: "15",
	})
	resetOpenAIAdvancedSchedulerSettingCacheForTest()

	cfg := newSchedulerTestOpenAIWSV2Config()
	cfg.Gateway.OpenAIWS.LBTopK = 1
	store := NewOpenAIWSStateStore(&schedulerTestGatewayCache{})
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{primary, backup}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
		openaiWSStateStore: store,
	}
	svc.SetSettingService(settingService)

	require.NoError(t, store.BindResponseAccount(ctx, groupID, "resp_prev_over_limit_backup", backup.ID, time.Hour))

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"resp_prev_over_limit_backup",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, primary.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickyPreviousHit)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_AdvancedSchedulerSkipsActiveOpenAIOverLimitShortCooldown(t *testing.T) {
	ctx := context.Background()
	groupID := int64(52001)
	rateLimitedUntil := time.Now().Add(30 * time.Minute)
	primary := Account{
		ID:               52001,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         0,
		RateLimitResetAt: &rateLimitedUntil,
	}
	backup := Account{
		ID:          52002,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    5,
	}

	settingService := newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
		openAIAdvancedSchedulerSettingKey:        "true",
		SettingKeyOpenAIOverLimitModeEnabled:     "true",
		SettingKeyOpenAIOverLimitCooldownSeconds: "15",
	})
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	cfg := &config.Config{}
	cfg.Gateway.OpenAIWS.LBTopK = 1
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{primary, backup}},
		cfg:                cfg,
		rateLimitService:   &RateLimitService{settingService: settingService},
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	svc.SetSettingService(settingService)
	svc.markOpenAIOverLimitCooldown(primary.ID, "gpt-5.1", time.Minute)

	selection, _, err := svc.SelectAccountWithScheduler(
		ctx,
		&groupID,
		"",
		"",
		"gpt-5.1",
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, backup.ID, selection.Account.ID)
}

func TestOpenAIGatewayService_HandleFailoverSideEffects_MarksOpenAIOverLimitCooldown(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.Model, "gpt-5.1")
	account := &Account{
		ID:          53001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}

	settingService := newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
		SettingKeyOpenAIOverLimitModeEnabled:     "true",
		SettingKeyOpenAIOverLimitCooldownSeconds: "12",
	})
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	svc.SetSettingService(settingService)

	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"error":"rate limited"}`)),
	}

	svc.handleFailoverSideEffects(ctx, resp, account)

	require.True(t, svc.isOpenAIOverLimitCooldownActive(account.ID, "gpt-5.1", time.Now()))
	require.False(t, svc.isOpenAIOverLimitCooldownActive(account.ID, "gpt-4.1", time.Now()))
}

func TestOpenAIGatewayService_GetOpenAIOverLimitModeSettings_NormalizesCooldownToTenWhenEnabled(t *testing.T) {
	testCases := []struct {
		name   string
		values map[string]string
	}{
		{
			name: "missing cooldown",
			values: map[string]string{
				SettingKeyOpenAIOverLimitModeEnabled: "true",
			},
		},
		{
			name: "small cooldown",
			values: map[string]string{
				SettingKeyOpenAIOverLimitModeEnabled:     "true",
				SettingKeyOpenAIOverLimitCooldownSeconds: "5",
			},
		},
		{
			name: "empty cooldown",
			values: map[string]string{
				SettingKeyOpenAIOverLimitModeEnabled:     "true",
				SettingKeyOpenAIOverLimitCooldownSeconds: "",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &OpenAIGatewayService{cfg: &config.Config{}}
			svc.SetSettingService(newOpenAIOverLimitSettingServiceWithValuesForTest(t, tc.values))

			settings := svc.getOpenAIOverLimitModeSettings(context.Background())

			require.True(t, settings.Enabled)
			require.Equal(t, 10, settings.CooldownSeconds)
		})
	}
}

func TestOpenAIGatewayService_HandleCompatErrorResponse_MarksOpenAIOverLimitCooldown(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	account := &Account{
		ID:          54001,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
		SettingKeyOpenAIOverLimitModeEnabled:     "true",
		SettingKeyOpenAIOverLimitCooldownSeconds: "12",
	}))
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.Model, "gpt-5.1"))

	resp := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"rate limited"}}`)),
	}

	_, err := svc.handleCompatErrorResponse(resp, c, account, writeChatCompletionsError)

	require.Error(t, err)
	require.True(t, svc.isOpenAIOverLimitCooldownActive(account.ID, "gpt-5.1", time.Now()))
	require.False(t, svc.isOpenAIOverLimitCooldownActive(account.ID, "gpt-4.1", time.Now()))
}
