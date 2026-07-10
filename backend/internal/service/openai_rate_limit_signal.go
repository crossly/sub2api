package service

import "strings"

// isOpenAIRateLimitSignal classifies protocol-level OpenAI errors independently
// from the transport used to deliver them (HTTP, SSE, or WebSocket).
func isOpenAIRateLimitSignal(codeRaw, errTypeRaw, messageRaw string) bool {
	code := strings.ToLower(strings.TrimSpace(codeRaw))
	errType := strings.ToLower(strings.TrimSpace(errTypeRaw))
	message := strings.ToLower(strings.TrimSpace(messageRaw))

	for _, value := range []string{code, errType} {
		if strings.Contains(value, "rate_limit") ||
			strings.Contains(value, "usage_limit") ||
			strings.Contains(value, "insufficient_quota") ||
			strings.Contains(value, "quota_exceeded") {
			return true
		}
	}

	if strings.Contains(message, "too many requests") || strings.Contains(message, "insufficient quota") {
		return true
	}
	if strings.Contains(message, "usage limit") && (strings.Contains(message, "reached") || strings.Contains(message, "exceeded")) {
		return true
	}
	if strings.Contains(message, "rate limit") && (strings.Contains(message, "reached") || strings.Contains(message, "exceeded")) {
		return true
	}
	return strings.Contains(message, "quota") && (strings.Contains(message, "reached") || strings.Contains(message, "exceeded"))
}
