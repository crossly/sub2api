//go:build unit

package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type countTokensFailoverAccountRepo struct {
	service.AccountRepository
	accounts         []service.Account
	rateLimitedCalls int
}

func (r *countTokensFailoverAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			account := r.accounts[i]
			return &account, nil
		}
	}
	return nil, service.ErrAccountNotFound
}

func (r *countTokensFailoverAccountRepo) ListByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r *countTokensFailoverAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r *countTokensFailoverAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r *countTokensFailoverAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]service.Account, error) {
	return r.accountsForPlatform(platform), nil
}

func (r *countTokensFailoverAccountRepo) SetRateLimited(_ context.Context, _ int64, _ time.Time) error {
	r.rateLimitedCalls++
	return nil
}

func (r *countTokensFailoverAccountRepo) accountsForPlatform(platform string) []service.Account {
	out := make([]service.Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.Platform == platform {
			out = append(out, account)
		}
	}
	return out
}

type countTokensFailoverSettingRepo struct {
	service.SettingRepository
}

func (r countTokensFailoverSettingRepo) Get(ctx context.Context, key string) (*service.Setting, error) {
	value, err := r.GetValue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &service.Setting{Key: key, Value: value}, nil
}

func (countTokensFailoverSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	switch key {
	case service.SettingKeyOpenAIOverLimitModeEnabled:
		return "true", nil
	case service.SettingKeyOpenAIOverLimitCooldownSeconds:
		return "10", nil
	default:
		return "", service.ErrSettingNotFound
	}
}

func (r countTokensFailoverSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	values := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, err := r.GetValue(context.Background(), key); err == nil {
			values[key] = value
		}
	}
	return values, nil
}

type countTokensFailoverUpstream struct {
	service.HTTPUpstream
	mu         sync.Mutex
	accountIDs []int64
}

func (u *countTokensFailoverUpstream) Do(_ *http.Request, _ string, accountID int64, _ int) (*http.Response, error) {
	u.mu.Lock()
	u.accountIDs = append(u.accountIDs, accountID)
	u.mu.Unlock()
	if accountID == 1 {
		resetAt := time.Now().Add(time.Hour).Unix()
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(bytes.NewBufferString(
				`{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","resets_at":` + strconv.FormatInt(resetAt, 10) + `}}`,
			)),
		}, nil
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(`{"object":"response.input_tokens","input_tokens":42}`)),
	}, nil
}

func (u *countTokensFailoverUpstream) calls() []int64 {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]int64(nil), u.accountIDs...)
}

func TestOpenAIGatewayHandlerCountTokens_OverLimit429FallsBackToHealthyAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		RunMode: config.RunModeSimple,
		Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{
			Enabled:           false,
			AllowInsecureHTTP: true,
		}},
	}
	settingService := service.NewSettingService(countTokensFailoverSettingRepo{}, cfg)
	repo := &countTokensFailoverAccountRepo{accounts: []service.Account{
		{
			ID:          1,
			Name:        "over-limit-primary",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
			Concurrency: 0,
			Priority:    1,
			Credentials: map[string]any{"api_key": "sk-primary", "base_url": "http://upstream.example"},
		},
		{
			ID:          2,
			Name:        "healthy-backup",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeAPIKey,
			Status:      service.StatusActive,
			Schedulable: true,
			Concurrency: 0,
			Priority:    2,
			Credentials: map[string]any{"api_key": "sk-backup", "base_url": "http://upstream.example"},
		},
	}}
	rateLimitService := service.NewRateLimitService(repo, nil, cfg, nil, nil)
	rateLimitService.SetSettingService(settingService)
	upstream := &countTokensFailoverUpstream{}
	concurrencyService := service.NewConcurrencyService(nil)
	gatewayService := service.NewOpenAIGatewayService(
		repo,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		cfg,
		nil,
		concurrencyService,
		nil,
		rateLimitService,
		nil,
		upstream,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		settingService,
		nil,
	)
	billingService := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	t.Cleanup(billingService.Stop)
	handler := NewOpenAIGatewayHandler(
		gatewayService,
		concurrencyService,
		billingService,
		service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg),
		nil,
		nil,
		nil,
		nil,
		cfg,
	)

	groupID := int64(9)
	body := []byte(`{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hello"}]}`)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		ID:      99,
		GroupID: &groupID,
		Group: &service.Group{
			ID:                    groupID,
			Platform:              service.PlatformOpenAI,
			AllowMessagesDispatch: true,
		},
		User: &service.User{ID: 100},
	})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 100, Concurrency: 1})

	handler.CountTokens(c)

	t.Logf("status=%d body=%s", rec.Code, rec.Body.String())
	require.Equal(t, []int64{1, 2}, upstream.calls())
	require.Equal(t, 1, repo.rateLimitedCalls)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), gjson.GetBytes(rec.Body.Bytes(), "input_tokens").Int())
}
