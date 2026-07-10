package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CountTokens handles Anthropic-compatible POST /v1/messages/count_tokens for OpenAI groups.
// It validates billing and routes to an OpenAI token-count bridge without taking concurrency slots
// or recording usage.
func (h *OpenAIGatewayHandler) CountTokens(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.anthropicErrorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.anthropicErrorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.count_tokens",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)

	if apiKey.Group != nil && !apiKey.Group.AllowMessagesDispatch {
		h.anthropicErrorResponse(c, http.StatusForbidden, "permission_error",
			"This group does not allow /v1/messages dispatch")
		return
	}

	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := readLenientJSONRequestBodyWithPrealloc(c.Request, h.cfg)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.anthropicErrorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	bodyRef := service.NewRequestBodyRef(body)
	parsedReq, err := service.ParseGatewayRequest(bodyRef, domain.PlatformAnthropic)
	if err != nil {
		logRequestBodyParseFailure(reqLog, body, err)
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	if parsedReq.Model == "" {
		h.anthropicErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return
	}

	reqModel := parsedReq.Model
	routingModel := service.NormalizeOpenAICompatRequestedModel(reqModel)
	preferredMappedModel := resolveOpenAIMessagesDispatchMappedModel(apiKey, reqModel)
	reqLog = reqLog.With(zap.String("model", reqModel), zap.Bool("stream", parsedReq.Stream))

	setOpsRequestContext(c, reqModel, false)
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(false, false)))

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, reqModel)
	mappedBodyForMessages := newOpenAIModelMappedBodyCache(body, h.gatewayService.ReplaceModelInBody)

	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("openai_count_tokens.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.anthropicErrorResponse(c, status, code, message)
		return
	}

	sessionHash := h.gatewayService.GenerateSessionHash(c, body)
	currentRoutingModel := routingModel
	if preferredMappedModel != "" {
		currentRoutingModel = preferredMappedModel
	}
	forwardBody := mappedBodyForMessages(channelMapping.Mapped, channelMapping.MappedModel)
	defaultMappedModel := preferredMappedModel
	failedAccountIDs := make(map[int64]struct{})
	var lastFailoverErr *service.UpstreamFailoverError
	switchCount := 0
	maxAccountSwitches := h.maxAccountSwitches
	if maxAccountSwitches <= 0 {
		maxAccountSwitches = 3
	}
	routingStart := time.Now()

	for {
		selection, _, selectErr := h.gatewayService.SelectAccountWithSchedulerForCapability(
			c.Request.Context(),
			apiKey.GroupID,
			"",
			sessionHash,
			currentRoutingModel,
			failedAccountIDs,
			service.OpenAIUpstreamTransportAny,
			service.OpenAIEndpointCapabilityChatCompletions,
			false,
			false,
			openAICompatibleRequestPlatform(apiKey),
		)
		service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(routingStart).Milliseconds())
		if selectErr != nil || selection == nil || selection.Account == nil {
			if selectErr != nil {
				reqLog.Warn("openai_count_tokens.account_select_failed", zap.Error(selectErr))
			}
			if lastFailoverErr != nil {
				h.writeCountTokensFailoverError(c, lastFailoverErr)
				return
			}
			cls := classifyNoAccountErrorFromGin(c, h.gatewayService, apiKey, currentRoutingModel, reqModel, service.PlatformOpenAI)
			if !cls.ModelNotFound {
				if selectErr != nil {
					markOpsRoutingCapacityLimitedIfNoAvailable(c, selectErr)
				} else {
					markOpsRoutingCapacityLimited(c)
				}
			}
			h.anthropicErrorResponse(c, cls.Status, cls.ErrType, cls.Message)
			return
		}

		account := selection.Account
		setOpsSelectedAccount(c, account.ID, account.Platform)
		writerSizeBeforeForward := c.Writer.Size()
		forwardErr := h.gatewayService.ForwardCountTokensAsAnthropic(
			c.Request.Context(), c, account, forwardBody, defaultMappedModel,
		)
		if selection.Acquired && selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
		if forwardErr == nil {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
			return
		}

		var failoverErr *service.UpstreamFailoverError
		if errors.As(forwardErr, &failoverErr) && c.Writer.Size() == writerSizeBeforeForward {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
			h.gatewayService.RecordOpenAIAccountSwitch()
			failedAccountIDs[account.ID] = struct{}{}
			lastFailoverErr = failoverErr
			if switchCount >= maxAccountSwitches {
				h.writeCountTokensFailoverError(c, failoverErr)
				return
			}
			switchCount++
			reqLog.Warn("openai_count_tokens.upstream_failover_switching",
				zap.Int64("account_id", account.ID),
				zap.Int("upstream_status", failoverErr.StatusCode),
				zap.Int("switch_count", switchCount),
				zap.Int("max_switches", maxAccountSwitches),
			)
			continue
		}

		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		reqLog.Error("openai_count_tokens.forward_failed", zap.Int64("account_id", account.ID), zap.Error(forwardErr))
		return
	}
}

func (h *OpenAIGatewayHandler) writeCountTokensFailoverError(c *gin.Context, failoverErr *service.UpstreamFailoverError) {
	status := http.StatusBadGateway
	if failoverErr != nil && failoverErr.StatusCode >= 400 && failoverErr.StatusCode <= 599 {
		status = failoverErr.StatusCode
	}
	message := "Upstream request failed"
	switch status {
	case http.StatusTooManyRequests:
		message = "Rate limit exceeded"
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, 529:
		message = "Upstream service temporarily unavailable"
	}
	h.anthropicErrorResponse(c, status, "upstream_error", message)
}
