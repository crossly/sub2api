package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type compactOpenAIOAuthServiceStub struct {
	tokenInfo     *OpenAITokenInfo
	refreshErr    error
	refreshCalled int32
}

func (s *compactOpenAIOAuthServiceStub) RefreshAccountToken(ctx context.Context, account *Account) (*OpenAITokenInfo, error) {
	atomic.AddInt32(&s.refreshCalled, 1)
	if s.refreshErr != nil {
		return nil, s.refreshErr
	}
	return s.tokenInfo, nil
}

func (s *compactOpenAIOAuthServiceStub) BuildAccountCredentials(info *OpenAITokenInfo) map[string]any {
	return map[string]any{
		"access_token":  info.AccessToken,
		"refresh_token": info.RefreshToken,
	}
}

func TestAccountTestService_TestAccountConnection_OpenAICompactOAuthSuccessPersistsSupport(t *testing.T) {
	gin.SetMode(gin.TestMode)

	updateCalls := make(chan map[string]any, 1)
	account := Account{
		ID:          1,
		Name:        "openai-oauth",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":               "oauth-token",
			"refresh_token":              "refresh-token",
			"chatgpt_account_id":         "chatgpt-acc",
			"chatgpt_account_is_fedramp": true,
		},
	}
	repo := &snapshotUpdateAccountRepo{
		stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}},
		updateExtraCalls:      updateCalls,
	}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"rid-probe"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"cmp_probe","status":"completed"}`)),
	}}
	oauthSvc := &compactOpenAIOAuthServiceStub{
		tokenInfo: &OpenAITokenInfo{
			AccessToken:  "fresh-access-token",
			RefreshToken: "fresh-refresh-token",
			ExpiresIn:    3600,
		},
	}
	svc := &AccountTestService{
		accountRepo:        repo,
		openaiOAuthService: oauthSvc,
		httpUpstream:       upstream,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/1/test", bytes.NewReader(nil))

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "", AccountTestModeCompact)
	require.NoError(t, err)
	require.Equal(t, int32(1), atomic.LoadInt32(&oauthSvc.refreshCalled))

	require.Equal(t, chatgptCodexAPIURL+"/compact", upstream.lastReq.URL.String())
	require.Equal(t, "chatgpt.com", upstream.lastReq.Host)
	require.Equal(t, "application/json", upstream.lastReq.Header.Get("Accept"))
	require.Equal(t, "Bearer fresh-access-token", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, codexCLIVersion, upstream.lastReq.Header.Get("Version"))
	require.NotEmpty(t, upstream.lastReq.Header.Get("Session_Id"))
	require.Equal(t, HTTPUpstreamProfileOpenAI, HTTPUpstreamProfileFromContext(upstream.lastReq.Context()))
	require.Equal(t, codexCLIUserAgent, upstream.lastReq.Header.Get("User-Agent"))
	require.Equal(t, "chatgpt-acc", upstream.lastReq.Header.Get("chatgpt-account-id"))
	require.Equal(t, "true", upstream.lastReq.Header.Get("x-openai-fedramp"))
	require.Equal(t, "gpt-5.4", gjson.GetBytes(upstream.lastBody, "model").String())

	updates := <-updateCalls
	require.Equal(t, true, updates["openai_compact_supported"])
	require.Equal(t, http.StatusOK, updates["openai_compact_last_status"])
	require.Contains(t, rec.Body.String(), `"type":"test_complete"`)
}

func TestAccountTestService_TestAccountConnection_OpenAICompactOAuth404MarksUnsupported(t *testing.T) {
	gin.SetMode(gin.TestMode)

	updateCalls := make(chan map[string]any, 1)
	account := Account{
		ID:          2,
		Name:        "openai-oauth",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "oauth-token",
			"chatgpt_account_id": "chatgpt-acc",
		},
	}
	repo := &snapshotUpdateAccountRepo{
		stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}},
		updateExtraCalls:      updateCalls,
	}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusNotFound,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`404 page not found`)),
	}}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/2/test", bytes.NewReader(nil))

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "", AccountTestModeCompact)
	require.Error(t, err)

	updates := <-updateCalls
	require.Equal(t, false, updates["openai_compact_supported"])
	require.Equal(t, http.StatusNotFound, updates["openai_compact_last_status"])
	require.Contains(t, rec.Body.String(), `"type":"error"`)
}

func TestAccountTestService_TestAccountConnection_OpenAICompactAPIKeyUsesCompactPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	updateCalls := make(chan map[string]any, 1)
	account := Account{
		ID:          3,
		Name:        "openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":               "sk-test",
			"base_url":              "https://example.com/v1",
			"compact_model_mapping": map[string]any{"gpt-5.4": "gpt-5.4-openai-compact"},
		},
	}
	repo := &snapshotUpdateAccountRepo{
		stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}},
		updateExtraCalls:      updateCalls,
	}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"cmp_probe_apikey","status":"completed"}`)),
	}}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/3/test", bytes.NewReader(nil))

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "", AccountTestModeCompact)
	require.NoError(t, err)

	require.Equal(t, "https://example.com/v1/responses/compact", upstream.lastReq.URL.String())
	require.Equal(t, "gpt-5.4-openai-compact", gjson.GetBytes(upstream.lastBody, "model").String())
	updates := <-updateCalls
	require.Equal(t, true, updates["openai_compact_supported"])
}

func TestAccountTestService_TestAccountConnection_OpenAICompactAPIKeyDefaultBaseURLUsesV1Path(t *testing.T) {
	gin.SetMode(gin.TestMode)

	updateCalls := make(chan map[string]any, 1)
	account := Account{
		ID:          4,
		Name:        "openai-apikey-default",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key": "sk-test",
		},
	}
	repo := &snapshotUpdateAccountRepo{
		stubOpenAIAccountRepo: stubOpenAIAccountRepo{accounts: []Account{account}},
		updateExtraCalls:      updateCalls,
	}
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"id":"cmp_probe_apikey_default","status":"completed"}`)),
	}}
	svc := &AccountTestService{
		accountRepo:  repo,
		httpUpstream: upstream,
		cfg:          &config.Config{Security: config.SecurityConfig{URLAllowlist: config.URLAllowlistConfig{Enabled: false}}},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/4/test", bytes.NewReader(nil))

	err := svc.TestAccountConnection(c, account.ID, "gpt-5.4", "", AccountTestModeCompact)
	require.NoError(t, err)
	require.Equal(t, "https://api.openai.com/v1/responses/compact", upstream.lastReq.URL.String())
	<-updateCalls
}
