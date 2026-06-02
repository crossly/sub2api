package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type openAIAndroidMobileConversationRequest struct {
	Action                     string                       `json:"action"`
	Messages                   []openAIAndroidMobileMessage `json:"messages"`
	Model                      string                       `json:"model"`
	HistoryAndTrainingDisabled bool                         `json:"history_and_training_disabled"`
	ForkFromSharedPost         bool                         `json:"fork_from_shared_post"`
	EnableMessageFollowups     bool                         `json:"enable_message_followups"`
	ForceUseSSE                bool                         `json:"force_use_sse"`
	ForceUseSearch             any                          `json:"force_use_search"`
	ForceParagen               bool                         `json:"force_paragen"`
	SupportedEncodings         []string                     `json:"supported_encodings"`
	SupportsBuffering          bool                         `json:"supports_buffering"`
	Timezone                   string                       `json:"timezone"`
	TimezoneOffsetMin          int                          `json:"timezone_offset_min"`
	SystemHints                []any                        `json:"system_hints"`
	IsOnboardingConversation   bool                         `json:"is_onboarding_conversation"`
	SkillsSettingsOverrides    map[string]bool              `json:"skills_settings_overrides"`
	ClientPrepareState         string                       `json:"client_prepare_state,omitempty"`
	Stream                     bool                         `json:"stream"`
	ConversationID             string                       `json:"conversation_id,omitempty"`
	ParentMessageID            string                       `json:"parent_message_id,omitempty"`
}

type openAIAndroidMobileMessage struct {
	ID        string                            `json:"id"`
	Author    openAIAndroidMobileAuthor         `json:"author"`
	Content   openAIAndroidMobileMessageContent `json:"content"`
	Status    string                            `json:"status"`
	Recipient string                            `json:"recipient"`
	Metadata  map[string]any                    `json:"metadata"`
}

type openAIAndroidMobileAuthor struct {
	Role string `json:"role"`
}

type openAIAndroidMobileMessageContent struct {
	ContentType string   `json:"content_type"`
	Parts       []string `json:"parts"`
}

type openAIAndroidMobileStreamResult struct {
	Text string
}

type openAIAndroidMobileToolCall struct {
	Name      string
	Arguments string
}

func buildOpenAIAndroidMobileConversationRequest(account *Account, body []byte) (*openAIAndroidMobileConversationRequest, error) {
	if account == nil {
		return nil, errors.New("account is required")
	}

	var raw map[string]any
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, err
		}
	}

	requestedModel := getStringFromMap(raw, "model")
	model := account.GetOpenAIAndroidMobileModelSlug(requestedModel)
	stream := getBoolFromMap(raw, "stream", false)
	text := extractOpenAIAndroidMobileUserText(raw)
	if strings.TrimSpace(text) == "" {
		text = "hi"
	}

	timezone := strings.TrimSpace(account.getExtraString("timezone"))
	if timezone == "" {
		timezone = "Asia/Shanghai"
	}
	req := &openAIAndroidMobileConversationRequest{
		Action:                     "next",
		Messages:                   []openAIAndroidMobileMessage{buildOpenAIAndroidMobileUserMessage(model, text)},
		Model:                      model,
		HistoryAndTrainingDisabled: account.getExtraBool("history_and_training_disabled"),
		ForkFromSharedPost:         false,
		EnableMessageFollowups:     true,
		ForceUseSSE:                true,
		ForceUseSearch:             nil,
		ForceParagen:               false,
		SupportedEncodings:         []string{"v1"},
		SupportsBuffering:          true,
		Timezone:                   timezone,
		TimezoneOffsetMin:          account.getExtraInt("timezone_offset_min"),
		SystemHints:                []any{},
		IsOnboardingConversation:   false,
		SkillsSettingsOverrides:    map[string]bool{"model_written_dil": false},
		ClientPrepareState:         "success",
		Stream:                     stream,
	}
	return req, nil
}

func buildOpenAIAndroidMobilePrepareRequest(account *Account, conversationReq *openAIAndroidMobileConversationRequest) *openAIAndroidMobileConversationRequest {
	prepare := *conversationReq
	prepare.Messages = []openAIAndroidMobileMessage{}
	prepare.EnableMessageFollowups = false
	prepare.ForceUseSSE = false
	prepare.ClientPrepareState = ""
	prepare.Stream = false
	return &prepare
}

func buildOpenAIAndroidMobileUserMessage(model, text string) openAIAndroidMobileMessage {
	return openAIAndroidMobileMessage{
		ID:     uuid.NewString(),
		Author: openAIAndroidMobileAuthor{Role: "user"},
		Content: openAIAndroidMobileMessageContent{
			ContentType: "text",
			Parts:       []string{text},
		},
		Status:    "finished_successfully",
		Recipient: "all",
		Metadata: map[string]any{
			"model_slug":                           model,
			"default_model_slug":                   model,
			"is_visually_hidden_from_conversation": false,
			"exclude_after_next_user_message":      false,
			"content_references":                   []any{},
			"search_result_groups":                 []any{},
			"search_queries":                       []any{},
			"image_results":                        []any{},
			"real_time_audio_has_video":            false,
			"dictation":                            false,
			"voice_mode_message":                   false,
			"image_gen_async":                      false,
			"trigger_async_ux":                     false,
			"writing_blocks":                       map[string]any{},
		},
	}
}

func extractOpenAIAndroidMobileUserText(raw map[string]any) string {
	if raw == nil {
		return ""
	}
	var parts []string
	if instructions := getStringFromMap(raw, "instructions"); strings.TrimSpace(instructions) != "" {
		parts = append(parts, instructions)
	}
	if messages, ok := raw["messages"].([]any); ok {
		for _, item := range messages {
			msg, ok := item.(map[string]any)
			if !ok {
				continue
			}
			role := strings.TrimSpace(getStringFromMap(msg, "role"))
			text := extractOpenAIAndroidMobileContentText(msg["content"])
			if text == "" {
				continue
			}
			if role == "system" || role == "developer" {
				parts = append(parts, role+": "+text)
			} else {
				parts = append(parts, text)
			}
		}
	}
	if input, ok := raw["input"]; ok {
		if text := extractOpenAIAndroidMobileContentText(input); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n\n")
}

func extractOpenAIAndroidMobileContentText(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		var parts []string
		for _, item := range v {
			switch entry := item.(type) {
			case string:
				if s := strings.TrimSpace(entry); s != "" {
					parts = append(parts, s)
				}
			case map[string]any:
				if text := getStringFromMap(entry, "text"); text != "" {
					parts = append(parts, text)
					continue
				}
				if content, ok := entry["content"]; ok {
					if text := extractOpenAIAndroidMobileContentText(content); text != "" {
						parts = append(parts, text)
					}
				}
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		if text := getStringFromMap(v, "text"); text != "" {
			return text
		}
		if content, ok := v["content"]; ok {
			return extractOpenAIAndroidMobileContentText(content)
		}
	}
	return ""
}

func writeOpenAIAndroidMobileChatCompletionStream(ctx context.Context, writer io.Writer, reader io.Reader, model string) (*openAIAndroidMobileStreamResult, error) {
	return streamOpenAIAndroidMobileText(ctx, reader, func(delta string) error {
		return writeOpenAIAndroidMobileChatChunk(writer, model, delta, false)
	}, func() error {
		if err := writeOpenAIAndroidMobileChatChunk(writer, model, "", true); err != nil {
			return err
		}
		_, err := fmt.Fprint(writer, "data: [DONE]\n\n")
		return err
	})
}

func streamOpenAIAndroidMobileText(ctx context.Context, reader io.Reader, onDelta func(string) error, onDone func() error) (*openAIAndroidMobileStreamResult, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var text strings.Builder
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		delta := extractOpenAIAndroidMobileSSEDelta(data)
		if delta == "" {
			continue
		}
		_, _ = text.WriteString(delta)
		if onDelta != nil {
			if err := onDelta(delta); err != nil {
				return nil, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if onDone != nil {
		if err := onDone(); err != nil {
			return nil, err
		}
	}
	return &openAIAndroidMobileStreamResult{Text: text.String()}, nil
}

func writeOpenAIAndroidMobileChatChunk(writer io.Writer, model, content string, finished bool) error {
	choice := map[string]any{
		"index": 0,
		"delta": map[string]any{},
	}
	if content != "" {
		choice["delta"] = map[string]any{"content": content}
	}
	if finished {
		choice["finish_reason"] = "stop"
	}
	chunk := map[string]any{
		"id":      "chatcmpl-" + uuid.NewString(),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{choice},
	}
	encoded, err := json.Marshal(chunk)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "data: %s\n\n", encoded)
	return err
}

func extractOpenAIAndroidMobileSSEDelta(data string) string {
	var decoded any
	if err := json.Unmarshal([]byte(data), &decoded); err != nil {
		return ""
	}
	switch v := decoded.(type) {
	case string:
		return ""
	case map[string]any:
		if isAndroidMobileAppendPatch(v) {
			return getStringFromMap(v, "v")
		}
		if patches, ok := v["patches"].([]any); ok {
			var out strings.Builder
			for _, item := range patches {
				patch, ok := item.(map[string]any)
				if !ok || !isAndroidMobileAppendPatch(patch) {
					continue
				}
				_, _ = out.WriteString(getStringFromMap(patch, "value"))
			}
			return out.String()
		}
	}
	return ""
}

func isAndroidMobileAppendPatch(patch map[string]any) bool {
	path := getStringFromMap(patch, "p")
	if path == "" {
		path = getStringFromMap(patch, "path")
	}
	op := getStringFromMap(patch, "o")
	if op == "" {
		op = getStringFromMap(patch, "op")
	}
	return path == "/message/content/parts/0" && op == "append"
}

var openAIAndroidMobileToolCallRe = regexp.MustCompile(`(?s)<sub2api_tool_call>(.*?)</sub2api_tool_call>`)

func extractOpenAIAndroidMobileToolCalls(text string) (string, []openAIAndroidMobileToolCall) {
	var calls []openAIAndroidMobileToolCall
	visible := openAIAndroidMobileToolCallRe.ReplaceAllStringFunc(text, func(match string) string {
		matches := openAIAndroidMobileToolCallRe.FindStringSubmatch(match)
		if len(matches) != 2 {
			return ""
		}
		var raw struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(matches[1]), &raw); err != nil || strings.TrimSpace(raw.Name) == "" {
			return ""
		}
		calls = append(calls, openAIAndroidMobileToolCall{
			Name:      raw.Name,
			Arguments: string(raw.Arguments),
		})
		return ""
	})
	return visible, calls
}

func (s *OpenAIGatewayService) forwardOpenAIAndroidMobile(ctx context.Context, c *gin.Context, account *Account, body []byte, startTime time.Time) (*OpenAIForwardResult, error) {
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}
	conversationReq, err := buildOpenAIAndroidMobileConversationRequest(account, body)
	if err != nil {
		return nil, err
	}
	baseURL := account.GetOpenAIAndroidMobileBaseURL()
	sessionID := uuid.NewString()

	prepareBody, err := json.Marshal(buildOpenAIAndroidMobilePrepareRequest(account, conversationReq))
	if err != nil {
		return nil, err
	}
	prepareReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/backend-api/f/conversation/prepare", bytes.NewReader(prepareBody))
	if err != nil {
		return nil, err
	}
	setOpenAIAndroidMobileHeaders(prepareReq, account, token, sessionID, "")
	prepareResp, err := s.doOpenAIAndroidMobileUpstream(ctx, prepareReq, account)
	if err != nil {
		return nil, err
	}
	defer func() { _ = prepareResp.Body.Close() }()
	if prepareResp.StatusCode < 200 || prepareResp.StatusCode >= 300 {
		return nil, writeOpenAIAndroidMobileUpstreamError(c, prepareResp)
	}
	prepareBytes, err := io.ReadAll(io.LimitReader(prepareResp.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	var prepareParsed struct {
		ConduitToken string `json:"conduit_token"`
	}
	if err := json.Unmarshal(prepareBytes, &prepareParsed); err != nil {
		return nil, err
	}
	if strings.TrimSpace(prepareParsed.ConduitToken) == "" {
		return nil, errors.New("android mobile prepare response missing conduit_token")
	}

	conversationBody, err := json.Marshal(conversationReq)
	if err != nil {
		return nil, err
	}
	conversationHTTPReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/backend-api/f/conversation", bytes.NewReader(conversationBody))
	if err != nil {
		return nil, err
	}
	setOpenAIAndroidMobileHeaders(conversationHTTPReq, account, token, sessionID, prepareParsed.ConduitToken)
	resp, err := s.doOpenAIAndroidMobileUpstream(ctx, conversationHTTPReq, account)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, writeOpenAIAndroidMobileUpstreamError(c, resp)
	}

	isResponsesEndpoint := isOpenAIAndroidMobileResponsesEndpoint(c)
	_, err = s.writeOpenAIAndroidMobileClientResponse(ctx, c, resp.Body, conversationReq.Model, conversationReq.Stream, isResponsesEndpoint)
	if err != nil {
		return nil, err
	}

	return &OpenAIForwardResult{
		Model:              conversationReq.Model,
		Stream:             conversationReq.Stream,
		Duration:           time.Since(startTime),
		FirstTokenMs:       nil,
		Usage:              OpenAIUsage{},
		ResponseHeaders:    resp.Header,
		ResponseID:         "android-mobile-" + uuid.NewString(),
		RequestID:          resp.Header.Get("x-request-id"),
		UpstreamModel:      conversationReq.Model,
		BillingModel:       conversationReq.Model,
		ImageCount:         0,
		ImageSize:          "",
		ImageInputSize:     "",
		ImageOutputSize:    "",
		ImageSizeBreakdown: map[string]int{},
	}, nil
}

func (s *OpenAIGatewayService) writeOpenAIAndroidMobileClientResponse(ctx context.Context, c *gin.Context, reader io.Reader, model string, stream bool, responsesEndpoint bool) (*openAIAndroidMobileStreamResult, error) {
	if stream {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)
		var result *openAIAndroidMobileStreamResult
		var err error
		if responsesEndpoint {
			result, err = writeOpenAIAndroidMobileResponsesStream(ctx, c.Writer, reader, model)
		} else {
			result, err = writeOpenAIAndroidMobileChatCompletionStream(ctx, c.Writer, reader, model)
		}
		c.Writer.Flush()
		return result, err
	}

	result, err := streamOpenAIAndroidMobileText(ctx, reader, nil, nil)
	if err != nil {
		return nil, err
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	if responsesEndpoint {
		err = json.NewEncoder(c.Writer).Encode(buildOpenAIAndroidMobileResponsesJSON(model, result.Text))
	} else {
		err = json.NewEncoder(c.Writer).Encode(buildOpenAIAndroidMobileChatCompletionJSON(model, result.Text))
	}
	return result, err
}

func writeOpenAIAndroidMobileResponsesStream(ctx context.Context, writer io.Writer, reader io.Reader, model string) (*openAIAndroidMobileStreamResult, error) {
	responseID := "resp_" + uuid.NewString()
	seq := 0
	if err := writeOpenAIAndroidMobileResponsesSSE(writer, "response.created", map[string]any{
		"type":            "response.created",
		"sequence_number": seq,
		"response":        buildOpenAIAndroidMobileResponsesJSONWithID(responseID, model, ""),
	}); err != nil {
		return nil, err
	}
	seq++
	var full strings.Builder
	return streamOpenAIAndroidMobileText(ctx, reader, func(delta string) error {
		_, _ = full.WriteString(delta)
		event := map[string]any{
			"type":            "response.output_text.delta",
			"sequence_number": seq,
			"output_index":    0,
			"content_index":   0,
			"delta":           delta,
		}
		seq++
		return writeOpenAIAndroidMobileResponsesSSE(writer, "response.output_text.delta", event)
	}, func() error {
		completed := buildOpenAIAndroidMobileResponsesJSONWithID(responseID, model, full.String())
		event := map[string]any{
			"type":            "response.completed",
			"sequence_number": seq,
			"response":        completed,
		}
		if err := writeOpenAIAndroidMobileResponsesSSE(writer, "response.completed", event); err != nil {
			return err
		}
		_, err := fmt.Fprint(writer, "data: [DONE]\n\n")
		return err
	})
}

func writeOpenAIAndroidMobileResponsesSSE(writer io.Writer, event string, payload map[string]any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event, encoded)
	return err
}

func buildOpenAIAndroidMobileChatCompletionJSON(model, text string) map[string]any {
	return map[string]any{
		"id":      "chatcmpl-" + uuid.NewString(),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   model,
		"choices": []any{
			map[string]any{
				"index": 0,
				"message": map[string]any{
					"role":    "assistant",
					"content": text,
				},
				"finish_reason": "stop",
			},
		},
	}
}

func buildOpenAIAndroidMobileResponsesJSON(model, text string) map[string]any {
	return buildOpenAIAndroidMobileResponsesJSONWithID("resp_"+uuid.NewString(), model, text)
}

func buildOpenAIAndroidMobileResponsesJSONWithID(id, model, text string) map[string]any {
	return map[string]any{
		"id":     id,
		"object": "response",
		"model":  model,
		"status": "completed",
		"output": []any{
			map[string]any{
				"type":   "message",
				"id":     "msg_" + uuid.NewString(),
				"role":   "assistant",
				"status": "completed",
				"content": []any{
					map[string]any{
						"type": "output_text",
						"text": text,
					},
				},
			},
		},
	}
}

func isOpenAIAndroidMobileResponsesEndpoint(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	return strings.Contains(c.Request.URL.Path, "/responses")
}

func setOpenAIAndroidMobileHeaders(req *http.Request, account *Account, token, sessionID, conduitToken string) {
	*req = *req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	req.Host = req.URL.Host
	req.Header.Set("authorization", "Bearer "+token)
	req.Header.Set("content-type", "application/json")
	req.Header.Set("accept", "text/event-stream,application/json")
	req.Header.Set("accept-language", "zh-Hans-CN,zh;q=0.9,en-CN;q=0.8,en;q=0.7")
	req.Header.Set("user-agent", account.GetOpenAIAndroidMobileUserAgent())
	req.Header.Set("x-oai-convo-session-id", sessionID)
	if chatgptAccountID := account.GetChatGPTAccountID(); chatgptAccountID != "" {
		req.Header.Set("chatgpt-account-id", chatgptAccountID)
	}
	if deviceID := account.GetOpenAIAndroidMobileDeviceID(); deviceID != "" {
		req.Header.Set("oai-device-id", deviceID)
	}
	if conduitToken != "" {
		req.Header.Set("x-conduit-token", conduitToken)
	}
}

func (s *OpenAIGatewayService) doOpenAIAndroidMobileUpstream(ctx context.Context, req *http.Request, account *Account) (*http.Response, error) {
	if s.httpUpstream == nil {
		return nil, errors.New("http upstream is not configured")
	}
	proxyURL := ""
	if account != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return s.httpUpstream.Do(req, proxyURL, account.ID, account.Concurrency)
}

func writeOpenAIAndroidMobileUpstreamError(c *gin.Context, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if c != nil && !c.Writer.Written() {
		c.Data(resp.StatusCode, resp.Header.Get("content-type"), body)
	}
	return fmt.Errorf("android mobile upstream returned %d: %s", resp.StatusCode, string(body))
}

func getStringFromMap(raw map[string]any, key string) string {
	if raw == nil {
		return ""
	}
	if s, ok := raw[key].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func getBoolFromMap(raw map[string]any, key string, fallback bool) bool {
	if raw == nil {
		return fallback
	}
	if b, ok := raw[key].(bool); ok {
		return b
	}
	return fallback
}
