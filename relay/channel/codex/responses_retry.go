package codex

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/tidwall/gjson"
)

const maxResponsesPendingBytes = 1 << 20

type codexResponsesUpstreamError struct {
	statusCode       int
	errorType        string
	code             string
	message          string
	param            string
	retrySameChannel bool
	retryNextChannel bool
	skipRetry        bool
}

func (e *codexResponsesUpstreamError) Error() string {
	if e == nil || e.message == "" {
		return "codex upstream response failed"
	}
	return e.message
}

func parseCodexResponsesUpstreamError(payload []byte) *codexResponsesUpstreamError {
	if !gjson.ValidBytes(payload) {
		return nil
	}
	root := gjson.ParseBytes(payload)
	eventType := strings.TrimSpace(root.Get("type").String())
	if eventType != "response.failed" && eventType != "response.error" && eventType != "error" {
		return nil
	}
	errorObject := root.Get("response.error")
	if !errorObject.Exists() {
		errorObject = root.Get("error")
	}
	message := sanitizeResponsesErrorField(errorObject.Get("message").String())
	if message == "" {
		message = sanitizeResponsesErrorField(extractCodexErrorMessage(payload))
	}
	errorType := sanitizeResponsesErrorField(errorObject.Get("type").String())
	code := sanitizeResponsesErrorField(errorObject.Get("code").String())
	param := sanitizeResponsesErrorField(errorObject.Get("param").String())
	classification := strings.ToLower(strings.Join([]string{errorType, code, message}, " "))

	err := &codexResponsesUpstreamError{
		statusCode: http.StatusBadGateway,
		errorType:  firstNonEmpty(errorType, "upstream_error"),
		code:       firstNonEmpty(code, "upstream_error"),
		message:    firstNonEmpty(message, "codex upstream response failed"),
		param:      param,
	}

	if containsAnyFold(classification,
		"selected model is at capacity",
		"server_is_overloaded",
		"service_unavailable",
		"temporarily unavailable",
		"capacity_exhausted",
		"overloaded",
	) {
		err.statusCode = http.StatusServiceUnavailable
		err.errorType = "server_error"
		err.code = "server_error"
		err.retrySameChannel = true
		err.retryNextChannel = true
		return err
	}
	if containsAnyFold(classification, "rate_limit", "slow_down", "too many requests") {
		err.statusCode = http.StatusTooManyRequests
		err.errorType = "rate_limit_error"
		err.retrySameChannel = true
		err.retryNextChannel = true
		return err
	}
	if containsAnyFold(classification,
		"authentication", "invalid_api_key", "unauthorized", "deactivated",
		"disabled", "suspended", "insufficient_quota", "billing", "credit balance",
	) {
		err.statusCode = http.StatusBadGateway
		err.retryNextChannel = true
		return err
	}
	if containsAnyFold(classification,
		"invalid_request", "context_length", "context window", "content_filter",
		"moderation", "policy_violation", "safety_violation", "unsupported",
	) {
		err.statusCode = http.StatusBadRequest
		err.skipRetry = true
	}
	return err
}

func (e *codexResponsesUpstreamError) toNewAPIError(retryAfter ...time.Duration) *types.NewAPIError {
	if e == nil {
		return types.NewOpenAIError(fmt.Errorf("codex upstream response failed"), types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}
	options := make([]types.NewAPIErrorOptions, 0, 1)
	switch {
	case e.retrySameChannel:
		options = append(options, types.ErrOptionWithSameChannelRetry())
	case e.retryNextChannel:
		options = append(options, types.ErrOptionWithNextChannelRetry())
	case e.skipRetry:
		options = append(options, types.ErrOptionWithSkipRetry())
	}
	if len(retryAfter) > 0 && retryAfter[0] > 0 {
		options = append(options, types.ErrOptionWithRetryAfter(retryAfter[0]))
	}
	return types.WithOpenAIError(types.OpenAIError{
		Message: e.Error(),
		Type:    e.errorType,
		Code:    e.code,
		Param:   e.param,
	}, e.statusCode, options...)
}

func responsesEventHasSemanticOutput(payload gjson.Result) bool {
	eventType := strings.TrimSpace(payload.Get("type").String())
	switch eventType {
	case "response.created", "response.queued", "response.in_progress",
		"response.error", "response.failed", "error":
		return false
	case "response.completed":
		return true
	case "response.output_item.added":
		itemType := strings.TrimSpace(payload.Get("item.type").String())
		return itemType != "" && itemType != "message" && itemType != "reasoning"
	}

	for _, path := range []string{
		"delta", "text", "arguments", "partial_image_b64", "audio", "result",
		"part.text", "part.refusal", "item.arguments", "item.result",
	} {
		if strings.TrimSpace(payload.Get(path).String()) != "" {
			return true
		}
	}
	if content := payload.Get("item.content"); content.IsArray() {
		for _, part := range content.Array() {
			if strings.TrimSpace(part.Get("text").String()) != "" || strings.TrimSpace(part.Get("refusal").String()) != "" {
				return true
			}
		}
	}
	if strings.HasSuffix(eventType, ".added") || strings.HasSuffix(eventType, ".in_progress") {
		return false
	}
	// Unknown non-empty events conservatively commit output. Replaying after a
	// newly introduced semantic event is more dangerous than losing failover.
	return eventType != ""
}

func normalizeResponsesTransientFailure(data string, upstreamErr *codexResponsesUpstreamError) string {
	if upstreamErr == nil || (!upstreamErr.retrySameChannel && !upstreamErr.retryNextChannel) {
		return data
	}
	var root map[string]any
	if err := common.Unmarshal([]byte(data), &root); err != nil {
		return data
	}
	errorBody := map[string]any{
		"message": upstreamErr.Error(),
		"type":    "server_error",
		"code":    "server_error",
	}
	if response, ok := root["response"].(map[string]any); ok {
		response["status"] = "failed"
		response["error"] = errorBody
	} else {
		root["error"] = errorBody
	}
	normalized, err := common.Marshal(root)
	if err != nil {
		return data
	}
	return string(normalized)
}

func sanitizeResponsesErrorField(value string) string {
	return truncateErrorMessage(common.MaskSensitiveInfo(strings.TrimSpace(value)))
}

func containsAnyFold(value string, markers ...string) bool {
	value = strings.ToLower(value)
	for _, marker := range markers {
		if strings.Contains(value, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}
