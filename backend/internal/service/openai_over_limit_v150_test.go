package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type openAIOverLimitSnapshotCache struct {
	SchedulerCache
	accounts   []*Account
	lastBucket SchedulerBucket
}

func (c *openAIOverLimitSnapshotCache) GetSnapshot(_ context.Context, bucket SchedulerBucket) ([]*Account, bool, error) {
	c.lastBucket = bucket
	return c.accounts, true, nil
}

type openAIOverLimitSnapshotRepo struct {
	productionLikeOpenAIOverLimitRepoStub
	listByPlatformCalls int
}

type openAIOverLimitRebuildCache struct {
	SchedulerCache
	rebuilt []SchedulerBucket
}

func (c *openAIOverLimitRebuildCache) TryLockBucket(_ context.Context, _ SchedulerBucket, _ time.Duration) (bool, error) {
	return true, nil
}

func (c *openAIOverLimitRebuildCache) UnlockBucket(_ context.Context, _ SchedulerBucket) error {
	return nil
}

func (c *openAIOverLimitRebuildCache) SetSnapshot(_ context.Context, bucket SchedulerBucket, _ []Account) error {
	c.rebuilt = append(c.rebuilt, bucket)
	return nil
}

func (r *openAIOverLimitSnapshotRepo) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	r.listByPlatformCalls++
	return r.productionLikeOpenAIOverLimitRepoStub.ListByPlatform(ctx, platform)
}

func newOpenAIOverLimitV150Service(t *testing.T, accounts []Account, cooldownSeconds int) *OpenAIGatewayService {
	t.Helper()
	svc := &OpenAIGatewayService{
		accountRepo: productionLikeOpenAIOverLimitRepoStub{
			stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: accounts},
		},
		cfg: &config.Config{},
	}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, true, cooldownSeconds))
	return svc
}

func selectOpenAIOverLimitV150Account(t *testing.T, svc *OpenAIGatewayService, model string) *Account {
	t.Helper()
	selection, _, err := svc.SelectAccountWithScheduler(
		context.Background(),
		nil,
		"",
		"",
		model,
		nil,
		OpenAIUpstreamTransportAny,
		false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	return selection.Account
}

func TestOpenAIOverLimitV150_Recent429UsesHealthyFallbackUntilProbeInterval(t *testing.T) {
	now := time.Now()
	resetAt := now.Add(time.Hour)
	rateLimitedAt := now.Add(-2 * time.Second)
	primary := Account{
		ID:               95001,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         1,
		RateLimitedAt:    &rateLimitedAt,
		RateLimitResetAt: &resetAt,
	}
	backup := Account{
		ID:          95002,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    10,
	}

	selected := selectOpenAIOverLimitV150Account(t, newOpenAIOverLimitV150Service(t, []Account{primary, backup}, 10), "gpt-5.1")
	require.Equal(t, backup.ID, selected.ID)
}

func TestOpenAIOverLimitV150_ProbeReadyAccountStillHonorsModelRateLimit(t *testing.T) {
	now := time.Now()
	globalResetAt := now.Add(time.Hour)
	rateLimitedAt := now.Add(-time.Minute)
	modelResetAt := now.Add(30 * time.Minute)
	primary := Account{
		ID:               95101,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         1,
		RateLimitedAt:    &rateLimitedAt,
		RateLimitResetAt: &globalResetAt,
		Extra: map[string]any{
			"model_rate_limits": map[string]any{
				"gpt-5.1": map[string]any{"rate_limit_reset_at": modelResetAt.UTC().Format(time.RFC3339)},
			},
		},
	}
	backup := Account{
		ID:          95102,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    10,
	}

	selected := selectOpenAIOverLimitV150Account(t, newOpenAIOverLimitV150Service(t, []Account{primary, backup}, 10), "gpt-5.1")
	require.Equal(t, backup.ID, selected.ID)
}

func TestOpenAIOverLimitV150_ProbeReadyRateLimitedAccountOverridesQuotaAutoPause(t *testing.T) {
	now := time.Now()
	resetAt := now.Add(time.Hour)
	rateLimitedAt := now.Add(-time.Minute)
	primary := Account{
		ID:               95201,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         1,
		RateLimitedAt:    &rateLimitedAt,
		RateLimitResetAt: &resetAt,
		Extra: map[string]any{
			"codex_5h_used_percent":   100.0,
			"auto_pause_5h_threshold": 0.95,
		},
	}
	backup := Account{
		ID:          95202,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    10,
	}

	selected := selectOpenAIOverLimitV150Account(t, newOpenAIOverLimitV150Service(t, []Account{primary, backup}, 10), "gpt-5.1")
	require.Equal(t, primary.ID, selected.ID)
}

func TestOpenAIOverLimitV150_AdvancedStickyCannotOverrideBestPriorityTier(t *testing.T) {
	now := time.Now()
	resetAt := now.Add(time.Hour)
	rateLimitedAt := now.Add(-time.Minute)
	primary := Account{
		ID:               95301,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         1,
		RateLimitedAt:    &rateLimitedAt,
		RateLimitResetAt: &resetAt,
	}
	backup := Account{
		ID:          95302,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    10,
	}

	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	t.Cleanup(resetOpenAIAdvancedSchedulerSettingCacheForTest)
	settingService := newOpenAIOverLimitSettingServiceWithValuesForTest(t, map[string]string{
		SettingKeyOpenAIOverLimitModeEnabled:                   "true",
		SettingKeyOpenAIOverLimitCooldownSeconds:               "10",
		openAIAdvancedSchedulerSettingKey:                      "true",
		SettingKeyOpenAIAdvancedSchedulerStickyWeightedEnabled: "true",
		SettingKeyOpenAIAdvancedSchedulerLBTopK:                "2",
		SettingKeyOpenAIAdvancedSchedulerWeightSessionSticky:   "1000",
	})
	svc := &OpenAIGatewayService{
		accountRepo: productionLikeOpenAIOverLimitRepoStub{
			stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{primary, backup}},
		},
		cache:            &schedulerTestGatewayCache{sessionBindings: map[string]int64{"sticky-backup": backup.ID}},
		cfg:              &config.Config{},
		rateLimitService: &RateLimitService{settingService: settingService},
	}
	svc.SetSettingService(settingService)

	selection, decision, err := svc.SelectAccountWithScheduler(
		context.Background(), nil, "", "sticky-backup", "gpt-5.1", nil, OpenAIUpstreamTransportAny, false,
	)
	require.NoError(t, err)
	require.NotNil(t, selection)
	require.NotNil(t, selection.Account)
	require.Equal(t, primary.ID, selection.Account.ID)
	require.Equal(t, 1, decision.TopK)
}

func TestOpenAIOverLimitV150_EnablingModeStillHonorsRecent429ProbeInterval(t *testing.T) {
	now := time.Now()
	resetAt := now.Add(time.Hour)
	rateLimitedAt := now.Add(-time.Minute)
	primary := Account{
		ID:               95401,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Concurrency:      1,
		Priority:         1,
		RateLimitedAt:    &rateLimitedAt,
		RateLimitResetAt: &resetAt,
	}
	backup := Account{
		ID:          95402,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    10,
	}
	svc := &OpenAIGatewayService{
		accountRepo: productionLikeOpenAIOverLimitRepoStub{
			stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{primary, backup}},
		},
		cfg: &config.Config{},
	}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, false, 10))
	svc.BlockAccountScheduling(&primary, time.Now().Add(time.Hour), "429")
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(&primary))

	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, true, 10))
	selected := selectOpenAIOverLimitV150Account(t, svc, "gpt-5.1")
	require.Equal(t, backup.ID, selected.ID)
}

func TestOpenAIOverLimitV150_StreamFailedRateLimitPersistsNextProbeTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &openAIOverLimitRateLimitRepoStub{}
	settingService := newOpenAIOverLimitSettingServiceForTest(t, true, 10)
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetSettingService(settingService)
	svc := &OpenAIGatewayService{
		accountRepo:      repo,
		cfg:              &config.Config{},
		rateLimitService: rateLimitService,
		settingService:   settingService,
	}
	account := &Account{
		ID:          95501,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
	}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	resetAt := time.Now().Add(time.Hour).Truncate(time.Second)
	payload := []byte(`{"type":"response.failed","response":{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","resets_at":` +
		strconv.FormatInt(resetAt.Unix(), 10) + `,"resets_in_seconds":3600,"plan_type":"pro"}}}`)

	svc.handleOpenAIStreamFailedAccountState(c, account, payload, "The usage limit has been reached")
	failoverErr := svc.newOpenAIStreamFailoverError(c, account, false, "req_95501", payload, "The usage limit has been reached")

	require.NotNil(t, failoverErr)
	require.Equal(t, http.StatusTooManyRequests, failoverErr.StatusCode)
	require.Equal(t, 1, repo.rateLimitCalls)
	require.Equal(t, account.ID, repo.lastRateLimitID)
	require.WithinDuration(t, resetAt, repo.lastRateLimitReset, time.Second)
}

func TestOpenAIOverLimitV150_StreamFailedSemanticStatusRecognizesRealRateLimitSignals(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		message string
	}{
		{
			name:    "usage limit type",
			payload: `{"type":"response.failed","response":{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached"}}}`,
		},
		{
			name:    "insufficient quota code",
			payload: `{"type":"response.failed","response":{"error":{"code":"insufficient_quota","message":"quota exhausted"}}}`,
		},
		{
			name:    "rate limit message",
			payload: `{"type":"response.failed","response":{"error":{"message":"Rate limit exceeded for this account"}}}`,
		},
		{
			name:    "rate code overrides generic error type",
			payload: `{"type":"response.failed","response":{"error":{"type":"invalid_request_error","code":"rate_limit_exceeded","message":"limit reached"}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, http.StatusTooManyRequests, openAIStreamFailedEventSemanticStatus([]byte(tt.payload), tt.message))
		})
	}
}

func TestOpenAIOverLimitV150_StreamFailedNormalizationPreservesResetMetadata(t *testing.T) {
	payload := []byte(`{"type":"response.failed","response":{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","resets_at":1999999999,"resets_in_seconds":600,"plan_type":"pro"}}}`)

	body := openAIStreamFailedEventPassthroughBody(payload, "The usage limit has been reached")

	require.Equal(t, int64(1999999999), gjson.GetBytes(body, "error.resets_at").Int())
	require.Equal(t, int64(600), gjson.GetBytes(body, "error.resets_in_seconds").Int())
	require.Equal(t, "pro", gjson.GetBytes(body, "error.plan_type").String())
}

func TestOpenAIOverLimitV150_StreamFailedPassthroughRuleStillPersistsRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetAt := time.Now().Add(time.Hour).Truncate(time.Second)
	payload := `{"type":"response.failed","response":{"error":{"type":"rate_limit_error","code":"rate_limit_exceeded","message":"rate limited","resets_at":` +
		strconv.FormatInt(resetAt.Unix(), 10) + `}}}`

	repo := &openAIOverLimitRateLimitRepoStub{}
	settingService := newOpenAIOverLimitSettingServiceForTest(t, true, 10)
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetSettingService(settingService)
	svc := &OpenAIGatewayService{
		accountRepo:      repo,
		cfg:              &config.Config{},
		rateLimitService: rateLimitService,
		settingService:   settingService,
	}
	rateLimitService.SetAccountRuntimeBlocker(svc)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	ruleStatus := http.StatusTooManyRequests
	ruleSvc := &ErrorPassthroughService{}
	ruleSvc.setLocalCache([]*model.ErrorPassthroughRule{{
		ID:              1,
		Enabled:         true,
		Platforms:       []string{PlatformOpenAI},
		ErrorCodes:      []int{http.StatusTooManyRequests},
		Keywords:        []string{"rate_limit_exceeded"},
		MatchMode:       model.MatchModeAll,
		ResponseCode:    &ruleStatus,
		PassthroughBody: true,
	}})
	BindErrorPassthroughService(c, ruleSvc)

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			"event: response.created",
			`data: {"type":"response.created","response":{"id":"resp_rate_limit"}}`,
			"",
			"event: response.failed",
			"data: " + payload,
			"",
		}, "\n"))),
	}
	account := &Account{ID: 95502, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true}

	_, err := svc.handleStreamingResponse(c.Request.Context(), resp, c, account, time.Now(), "gpt-5.1", "gpt-5.1")

	require.Error(t, err)
	require.Equal(t, 1, repo.rateLimitCalls)
	require.Equal(t, account.ID, repo.lastRateLimitID)
	require.WithinDuration(t, resetAt, repo.lastRateLimitReset, time.Second)
}

func TestOpenAIOverLimitV150_RuntimeHardBlockSurvivesEarlierLong429Block(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, true, 10))
	account := &Account{ID: 95503, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	svc.BlockAccountScheduling(account, time.Now().Add(time.Hour), "429")
	svc.BlockAccountScheduling(account, time.Now().Add(time.Minute), "transport_error")

	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAIOverLimitV150_RuntimeHardBlockSurvivesLaterLong429Block(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, true, 10))
	account := &Account{ID: 95504, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	svc.BlockAccountScheduling(account, time.Now().Add(time.Minute), "transport_error")
	svc.BlockAccountScheduling(account, time.Now().Add(time.Hour), "429")

	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAIOverLimitV150_RecentRuntime429StillBlocksUntilProbeInterval(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, true, 10))
	account := &Account{ID: 95505, Platform: PlatformOpenAI, Type: AccountTypeOAuth}

	svc.BlockAccountScheduling(account, time.Now().Add(time.Hour), "429")

	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	value, ok := svc.openaiAccountRuntimeBlockUntil.Load(account.ID)
	require.True(t, ok)
	block, ok := value.(openAIAccountRuntimeBlock)
	require.True(t, ok)
	block.RateLimitedAt = time.Now().Add(-11 * time.Second)
	svc.openaiAccountRuntimeBlockUntil.Store(account.ID, block)
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))
}

func TestOpenAIOverLimitV150_CooldownNormalizationBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		enabled  bool
		cooldown int
		want     int
	}{
		{name: "disabled", enabled: false, cooldown: 120, want: 0},
		{name: "minimum", enabled: true, cooldown: 9, want: 10},
		{name: "unchanged", enabled: true, cooldown: 45, want: 45},
		{name: "maximum", enabled: true, cooldown: 301, want: 300},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, normalizeOpenAIOverLimitCooldownSeconds(tt.enabled, tt.cooldown))
		})
	}
}

func TestOpenAIOverLimitV150_NonStreamingSSE429PersistsStateAndFailsOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetAt := time.Now().Add(time.Hour).Truncate(time.Second)
	payload := `{"type":"response.failed","response":{"id":"resp_nonstream_rate_limit","error":{"type":"usage_limit_reached","message":"The usage limit has been reached","resets_at":` +
		strconv.FormatInt(resetAt.Unix(), 10) + `}}}`

	repo := &openAIOverLimitRateLimitRepoStub{}
	settingService := newOpenAIOverLimitSettingServiceForTest(t, true, 10)
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetSettingService(settingService)
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: &config.Config{}, rateLimitService: rateLimitService, settingService: settingService}
	rateLimitService.SetAccountRuntimeBlocker(svc)
	account := &Account{ID: 95509, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "X-Request-Id": []string{"req_nonstream_rate_limit"}},
		Body:       io.NopCloser(strings.NewReader("event: response.failed\ndata: " + payload + "\n\n")),
	}

	_, err := svc.handleNonStreamingResponse(c.Request.Context(), resp, c, account, "gpt-5.1", "gpt-5.1")

	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr), "err=%T %v", err, err)
	require.Empty(t, rec.Body.String())
	require.Equal(t, 1, repo.rateLimitCalls)
	require.WithinDuration(t, resetAt, repo.lastRateLimitReset, time.Second)
}

func TestOpenAIOverLimitV150_PassthroughNonStreamingSSE429PersistsStateAndFailsOver(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetAt := time.Now().Add(time.Hour).Truncate(time.Second)
	payload := `{"type":"response.failed","response":{"id":"resp_passthrough_nonstream_rate_limit","error":{"type":"usage_limit_reached","message":"The usage limit has been reached","resets_at":` +
		strconv.FormatInt(resetAt.Unix(), 10) + `}}}`

	repo := &openAIOverLimitRateLimitRepoStub{}
	settingService := newOpenAIOverLimitSettingServiceForTest(t, true, 10)
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetSettingService(settingService)
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: &config.Config{}, rateLimitService: rateLimitService, settingService: settingService}
	rateLimitService.SetAccountRuntimeBlocker(svc)
	account := &Account{ID: 95511, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "X-Request-Id": []string{"req_passthrough_nonstream_rate_limit"}},
		Body:       io.NopCloser(strings.NewReader("event: response.failed\ndata: " + payload + "\n\n")),
	}

	_, err := svc.handleNonStreamingResponsePassthrough(c.Request.Context(), resp, c, account, "gpt-5.1", "gpt-5.1")

	var failoverErr *UpstreamFailoverError
	require.True(t, errors.As(err, &failoverErr), "err=%T %v", err, err)
	require.Equal(t, http.StatusTooManyRequests, failoverErr.StatusCode)
	require.Empty(t, rec.Body.String())
	require.Equal(t, 1, repo.rateLimitCalls)
	require.WithinDuration(t, resetAt, repo.lastRateLimitReset, time.Second)
}

func TestOpenAIOverLimitV150_PassthroughNonRateSSEFailureKeepsV150ProtocolError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.failed","error":{"message":"upstream rejected request"}}`,
			`data: [DONE]`,
		}, "\n"))),
	}

	_, err := svc.handleNonStreamingResponsePassthrough(c.Request.Context(), resp, c, nil, "gpt-5.1", "gpt-5.1")

	var failoverErr *UpstreamFailoverError
	require.Error(t, err)
	require.False(t, errors.As(err, &failoverErr))
	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Contains(t, rec.Body.String(), "upstream rejected request")
}

func TestOpenAIOverLimitV150_Stream429AfterClientOutputStillPersistsState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetAt := time.Now().Add(time.Hour).Truncate(time.Second)
	payload := `{"type":"response.failed","response":{"id":"resp_late_rate_limit","error":{"type":"usage_limit_reached","message":"The usage limit has been reached","resets_at":` +
		strconv.FormatInt(resetAt.Unix(), 10) + `}}}`

	repo := &openAIOverLimitRateLimitRepoStub{}
	settingService := newOpenAIOverLimitSettingServiceForTest(t, true, 10)
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetSettingService(settingService)
	svc := &OpenAIGatewayService{accountRepo: repo, cfg: &config.Config{}, rateLimitService: rateLimitService, settingService: settingService}
	rateLimitService.SetAccountRuntimeBlocker(svc)
	account := &Account{ID: 95510, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true}
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: io.NopCloser(strings.NewReader(strings.Join([]string{
			`data: {"type":"response.output_text.delta","delta":"partial"}`,
			"",
			"event: response.failed",
			"data: " + payload,
			"",
		}, "\n"))),
	}

	_, err := svc.handleStreamingResponse(c.Request.Context(), resp, c, account, time.Now(), "gpt-5.1", "gpt-5.1")

	require.Error(t, err)
	require.NotEmpty(t, rec.Body.String())
	require.Equal(t, 1, repo.rateLimitCalls)
	require.WithinDuration(t, resetAt, repo.lastRateLimitReset, time.Second)
}

func TestOpenAIOverLimitV150_CandidateAccountsUseDedicatedSchedulerSnapshot(t *testing.T) {
	resetAt := time.Now().Add(time.Hour)
	rateLimitedAt := time.Now().Add(-time.Minute)
	account := Account{
		ID:               95506,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Priority:         1,
		RateLimitedAt:    &rateLimitedAt,
		RateLimitResetAt: &resetAt,
	}
	cache := &openAIOverLimitSnapshotCache{accounts: []*Account{&account}}
	repo := &openAIOverLimitSnapshotRepo{productionLikeOpenAIOverLimitRepoStub: productionLikeOpenAIOverLimitRepoStub{
		stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}},
	}}
	snapshot := NewSchedulerSnapshotService(cache, nil, repo, nil, &config.Config{})
	svc := &OpenAIGatewayService{accountRepo: repo, schedulerSnapshot: snapshot, cfg: &config.Config{}}
	svc.SetSettingService(newOpenAIOverLimitSettingServiceForTest(t, true, 10))

	accounts, err := svc.openAIOverLimitStrategy().CandidateAccounts(context.Background(), nil)

	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, account.ID, accounts[0].ID)
	require.Equal(t, "openai_over_limit", cache.lastBucket.Mode)
	require.Zero(t, repo.listByPlatformCalls, "snapshot hit must not query the candidate pool from DB")
}

func TestOpenAIOverLimitV150_SchedulerBucketDBLoadIncludesRateLimitedAccounts(t *testing.T) {
	now := time.Now()
	resetAt := now.Add(time.Hour)
	rateLimitedAt := now.Add(-time.Minute)
	rateLimited := Account{
		ID:               95507,
		Platform:         PlatformOpenAI,
		Type:             AccountTypeOAuth,
		Status:           StatusActive,
		Schedulable:      true,
		Priority:         1,
		RateLimitedAt:    &rateLimitedAt,
		RateLimitResetAt: &resetAt,
	}
	healthy := Account{ID: 95508, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Priority: 2}
	repo := productionLikeOpenAIOverLimitRepoStub{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{rateLimited, healthy}}}
	snapshot := NewSchedulerSnapshotService(nil, nil, repo, nil, &config.Config{})

	accounts, err := snapshot.loadAccountsFromDB(context.Background(), SchedulerBucket{
		GroupID:  0,
		Platform: PlatformOpenAI,
		Mode:     "openai_over_limit",
	}, false)

	require.NoError(t, err)
	require.Len(t, accounts, 2)
	require.Equal(t, rateLimited.ID, accounts[0].ID)
}

func TestOpenAIOverLimitV150_DefaultSchedulerBucketsIncludeDedicatedOpenAIMode(t *testing.T) {
	snapshot := NewSchedulerSnapshotService(nil, nil, nil, nil, &config.Config{})

	buckets, err := snapshot.defaultBuckets(context.Background())

	require.NoError(t, err)
	found := false
	for _, bucket := range buckets {
		if bucket.GroupID == 0 && bucket.Platform == PlatformOpenAI && bucket.Mode == "openai_over_limit" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestOpenAIOverLimitV150_AccountOutboxRebuildIncludesDedicatedOpenAIBucket(t *testing.T) {
	account := Account{
		ID:          95512,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Priority:    1,
	}
	cache := &openAIOverLimitRebuildCache{}
	repo := productionLikeOpenAIOverLimitRepoStub{stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}}}
	snapshot := NewSchedulerSnapshotService(cache, nil, repo, nil, &config.Config{})

	err := snapshot.rebuildByAccount(context.Background(), &account, nil, "account_change", make(map[batchSeenKey]struct{}))

	require.NoError(t, err)
	require.Contains(t, cache.rebuilt, SchedulerBucket{
		GroupID:  0,
		Platform: PlatformOpenAI,
		Mode:     SchedulerModeOpenAIOverLimit,
	})
}

func TestOpenAIOverLimitV150_DoesNotOverride529OverloadCooldown(t *testing.T) {
	overloadUntil := time.Now().Add(10 * time.Minute)
	primary := Account{
		ID:            95601,
		Platform:      PlatformOpenAI,
		Type:          AccountTypeOAuth,
		Status:        StatusActive,
		Schedulable:   true,
		Concurrency:   1,
		Priority:      1,
		OverloadUntil: &overloadUntil,
	}
	backup := Account{
		ID:          95602,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    10,
	}

	selected := selectOpenAIOverLimitV150Account(t, newOpenAIOverLimitV150Service(t, []Account{primary, backup}, 10), "gpt-5.1")
	require.Equal(t, backup.ID, selected.ID)
}
