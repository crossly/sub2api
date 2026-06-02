package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAccount_IsOpenAIAndroidMobile(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "android-token",
		},
		Extra: map[string]any{
			"openai_backend": " android_mobile ",
		},
	}

	require.True(t, account.IsOpenAIAndroidMobile())
	require.Equal(t, "https://android.chat.openai.com", account.GetOpenAIAndroidMobileBaseURL())
	require.NotEmpty(t, account.GetOpenAIAndroidMobileDeviceID())
}

func TestBuildOpenAIAndroidMobileConversationRequestFromResponses(t *testing.T) {
	account := &Account{
		ID:       42,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"openai_backend":                "android_mobile",
			"android_model_slug":            "gpt-5-5",
			"timezone":                      "Asia/Shanghai",
			"timezone_offset_min":           -480,
			"history_and_training_disabled": true,
		},
	}
	body := []byte(`{"model":"gpt-5.5","stream":true,"instructions":"Be terse.","input":[{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}]}`)

	req, err := buildOpenAIAndroidMobileConversationRequest(account, body)

	require.NoError(t, err)
	require.Equal(t, "gpt-5-5", req.Model)
	require.True(t, req.Stream)
	require.True(t, req.ForceUseSSE)
	require.Equal(t, "Asia/Shanghai", req.Timezone)
	require.Equal(t, -480, req.TimezoneOffsetMin)
	require.True(t, req.HistoryAndTrainingDisabled)
	require.Len(t, req.Messages, 1)
	require.Contains(t, req.Messages[0].Content.Parts[0], "Be terse.")
	require.Contains(t, req.Messages[0].Content.Parts[0], "hi")
	require.Equal(t, "gpt-5-5", req.Messages[0].Metadata["model_slug"])
}

func TestOpenAIAndroidMobileSSEToChatCompletionChunks(t *testing.T) {
	androidSSE := strings.NewReader("" +
		"event: delta_encoding\n" +
		"data: \"v1\"\n\n" +
		"event: delta\n" +
		"data: {\"p\":\"/message/content/parts/0\",\"o\":\"append\",\"v\":\"hel\"}\n\n" +
		"event: delta\n" +
		"data: {\"patches\":[{\"path\":\"/message/content/parts/0\",\"op\":\"append\",\"value\":\"lo\"}]}\n\n",
	)
	rec := httptest.NewRecorder()

	result, err := writeOpenAIAndroidMobileChatCompletionStream(context.Background(), rec, androidSSE, "gpt-5.5")

	require.NoError(t, err)
	require.Equal(t, "hello", result.Text)
	out := rec.Body.String()
	require.Contains(t, out, `"object":"chat.completion.chunk"`)
	require.Contains(t, out, `"content":"hel"`)
	require.Contains(t, out, `"content":"lo"`)
	require.Contains(t, out, "data: [DONE]")
}

func TestOpenAIAndroidMobileExtractsSyntheticToolCall(t *testing.T) {
	text := `before <sub2api_tool_call>{"name":"apply_patch","arguments":{"patch":"*** Begin Patch\n*** End Patch"}}</sub2api_tool_call> after`

	visible, calls := extractOpenAIAndroidMobileToolCalls(text)

	require.Equal(t, "before  after", visible)
	require.Len(t, calls, 1)
	require.Equal(t, "apply_patch", calls[0].Name)
	require.JSONEq(t, `{"patch":"*** Begin Patch\n*** End Patch"}`, calls[0].Arguments)
}

func TestOpenAIGatewayService_AndroidMobileForwardUsesAndroidConversationBackend(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":"hi"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"status":"ok","conduit_token":"conduit-test"}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				"event: delta\n" +
					"data: {\"p\":\"/message/content/parts/0\",\"o\":\"append\",\"v\":\"hello android\"}\n\n",
			)),
		},
	}}
	svc := &OpenAIGatewayService{
		cfg:          &config.Config{},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          1001,
		Name:        "android-mobile",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "android-token",
			"chatgpt_account_id": "chatgpt-account",
			"client_id":          "app_xwBKzt04752TTSfXnki17hmB",
		},
		Extra: map[string]any{
			"openai_backend":     "android_mobile",
			"android_model_slug": "gpt-5-5",
			"android_device_id":  "device-test",
		},
		Status:      StatusActive,
		Schedulable: true,
	}

	result, err := svc.Forward(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Len(t, upstream.requests, 2)
	require.Equal(t, "https://android.chat.openai.com/backend-api/f/conversation/prepare", upstream.requests[0].URL.String())
	require.Equal(t, "https://android.chat.openai.com/backend-api/f/conversation", upstream.requests[1].URL.String())
	require.Equal(t, "android.chat.openai.com", upstream.requests[1].Host)
	require.Equal(t, "Bearer android-token", upstream.requests[1].Header.Get("authorization"))
	require.Equal(t, "chatgpt-account", upstream.requests[1].Header.Get("chatgpt-account-id"))
	require.Equal(t, "device-test", upstream.requests[1].Header.Get("oai-device-id"))
	require.Equal(t, "conduit-test", upstream.requests[1].Header.Get("x-conduit-token"))
	require.Equal(t, "gpt-5-5", gjson.GetBytes(upstream.bodies[1], "model").String())
	require.Contains(t, rec.Body.String(), "hello android")
	require.NotContains(t, upstream.requests[1].URL.String(), "codex/responses")
}

func TestOpenAIGatewayService_AndroidMobileForwardNonStreamingChatCompletion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-5.5","stream":false,"messages":[{"role":"user","content":"hi"}]}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := newOpenAIAndroidMobileUpstream("hello json")
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	account := newOpenAIAndroidMobileTestAccount()

	result, err := svc.Forward(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Stream)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.Equal(t, "chat.completion", gjson.Get(rec.Body.String(), "object").String())
	require.Equal(t, "hello json", gjson.Get(rec.Body.String(), "choices.0.message.content").String())
	require.True(t, gjson.GetBytes(upstream.bodies[1], "force_use_sse").Bool())
}

func TestOpenAIGatewayService_AndroidMobileForwardResponsesStreamingShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	body := []byte(`{"model":"gpt-5.5","stream":true,"input":"hi"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	upstream := newOpenAIAndroidMobileUpstream("hello responses")
	svc := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
	account := newOpenAIAndroidMobileTestAccount()

	result, err := svc.Forward(context.Background(), c, account, body)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Stream)
	out := rec.Body.String()
	require.Contains(t, out, `"type":"response.created"`)
	require.Contains(t, out, `"type":"response.output_text.delta"`)
	require.Contains(t, out, `"delta":"hello responses"`)
	require.Contains(t, out, `"type":"response.completed"`)
	require.NotContains(t, out, `"object":"chat.completion.chunk"`)
}

func newOpenAIAndroidMobileUpstream(text string) *httpUpstreamRecorder {
	return &httpUpstreamRecorder{responses: []*http.Response{
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"status":"ok","conduit_token":"conduit-test"}`)),
		},
		{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body: io.NopCloser(strings.NewReader(
				"event: delta\n" +
					"data: {\"p\":\"/message/content/parts/0\",\"o\":\"append\",\"v\":\"" + text + "\"}\n\n",
			)),
		},
	}}
}

func newOpenAIAndroidMobileTestAccount() *Account {
	return &Account{
		ID:          1001,
		Name:        "android-mobile",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{
			"access_token":       "android-token",
			"chatgpt_account_id": "chatgpt-account",
			"client_id":          "app_xwBKzt04752TTSfXnki17hmB",
		},
		Extra: map[string]any{
			"openai_backend":     "android_mobile",
			"android_model_slug": "gpt-5-5",
			"android_device_id":  "device-test",
		},
		Status:      StatusActive,
		Schedulable: true,
	}
}
