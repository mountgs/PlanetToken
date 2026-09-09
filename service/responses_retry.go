package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/samber/hot"
	"github.com/tidwall/gjson"
)

const (
	responsesRetryTraceContextKey   = "responses_retry_trace"
	responsesRetrySuccessContextKey = "responses_retry_success"
	responsesCooldownNamespace      = "new-api:responses_channel_cooldown:v1"
)

type ResponsesRetryAttempt struct {
	ChannelID     int             `json:"channel_id"`
	MultiKeyIndex int             `json:"multi_key_index,omitempty"`
	Attempt       int             `json:"attempt"`
	Scope         string          `json:"scope"`
	StatusCode    int             `json:"status_code"`
	ErrorCode     types.ErrorCode `json:"error_code,omitempty"`
	DurationMS    int64           `json:"duration_ms"`
}

type ResponsesRetryState struct {
	deadline time.Time
}

func NewResponsesRetryState(now time.Time) *ResponsesRetryState {
	seconds := common.ResponsesRetryMaxDurationSeconds
	if seconds <= 0 {
		seconds = 60
	}
	return &ResponsesRetryState{deadline: now.Add(time.Duration(seconds) * time.Second)}
}

func (s *ResponsesRetryState) CanStartAnotherAttempt(now time.Time) bool {
	return s != nil && now.Before(s.deadline)
}

func responsesRetryDelay(retryNumber int, retryAfter time.Duration) time.Duration {
	delay := 500 * time.Millisecond
	for i := 1; i < retryNumber && delay < 8*time.Second; i++ {
		delay *= 2
	}
	if delay > 8*time.Second {
		delay = 8 * time.Second
	}
	if retryAfter > 0 {
		return retryAfter
	}
	return delay
}

func (s *ResponsesRetryState) Wait(ctx context.Context, retryNumber int, retryAfter time.Duration) bool {
	if s == nil || retryNumber <= 0 {
		return false
	}
	delay := responsesRetryDelay(retryNumber, retryAfter)
	remaining := time.Until(s.deadline)
	if remaining <= 0 || delay > remaining {
		return false
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func IsResponsesRetryRequestPath(path string) bool {
	path = strings.TrimSuffix(strings.TrimSpace(path), "/")
	return strings.HasSuffix(path, "/responses")
}

func ParseResponsesUpstreamError(payload []byte, retryAfter time.Duration) *types.NewAPIError {
	if !gjson.ValidBytes(payload) {
		return nil
	}
	root := gjson.ParseBytes(payload)
	eventType := strings.TrimSpace(root.Get("type").String())
	failedResponse := strings.EqualFold(strings.TrimSpace(root.Get("status").String()), "failed") ||
		strings.EqualFold(strings.TrimSpace(root.Get("response.status").String()), "failed")
	if eventType != "response.failed" && eventType != "response.error" && eventType != "error" && !failedResponse {
		return nil
	}

	errorObject := root.Get("response.error")
	if !errorObject.Exists() {
		errorObject = root.Get("error")
	}
	errorType := sanitizeResponsesErrorValue(errorObject.Get("type").String())
	code := sanitizeResponsesErrorValue(errorObject.Get("code").String())
	message := sanitizeResponsesErrorValue(errorObject.Get("message").String())
	if message == "" {
		message = "upstream Responses request failed"
	}
	classification := strings.ToLower(strings.Join([]string{errorType, code, message}, " "))
	statusCode := http.StatusBadGateway
	options := make([]types.NewAPIErrorOptions, 0, 3)

	switch {
	case containsResponsesErrorMarker(classification,
		"selected model is at capacity", "server_is_overloaded", "service_unavailable",
		"temporarily unavailable", "capacity_exhausted", "overloaded"):
		statusCode = http.StatusServiceUnavailable
		errorType = "server_error"
		code = "server_error"
		options = append(options, types.ErrOptionWithSameChannelRetry(), types.ErrOptionWithNextChannelRetry())
	case containsResponsesErrorMarker(classification, "rate_limit", "slow_down", "too many requests"):
		statusCode = http.StatusTooManyRequests
		errorType = "rate_limit_error"
		options = append(options, types.ErrOptionWithSameChannelRetry(), types.ErrOptionWithNextChannelRetry())
	case containsResponsesErrorMarker(classification,
		"authentication", "invalid_api_key", "unauthorized", "deactivated", "disabled",
		"suspended", "insufficient_quota", "billing", "credit balance"):
		options = append(options, types.ErrOptionWithNextChannelRetry())
	case containsResponsesErrorMarker(classification,
		"invalid_request", "context_length", "context window", "content_filter", "moderation",
		"policy_violation", "safety_violation", "unsupported"):
		statusCode = http.StatusBadRequest
		options = append(options, types.ErrOptionWithSkipRetry())
	default:
		options = append(options, types.ErrOptionWithSameChannelRetry(), types.ErrOptionWithNextChannelRetry())
	}
	if retryAfter > 0 {
		options = append(options, types.ErrOptionWithRetryAfter(retryAfter))
	}

	return types.WithOpenAIError(types.OpenAIError{
		Message: message,
		Type:    firstResponsesErrorValue(errorType, "upstream_error"),
		Code:    firstResponsesErrorValue(code, "upstream_error"),
		Param:   sanitizeResponsesErrorValue(errorObject.Get("param").String()),
	}, statusCode, options...)
}

func ParseResponsesRetryAfterHeader(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	delay := time.Until(retryAt)
	if delay <= 0 {
		return 0
	}
	return delay
}

func ResponsesEventHasSemanticOutput(payload []byte) bool {
	if !gjson.ValidBytes(payload) {
		return true
	}
	root := gjson.ParseBytes(payload)
	eventType := strings.TrimSpace(root.Get("type").String())
	switch eventType {
	case "response.created", "response.queued", "response.in_progress", "response.error", "response.failed", "error":
		return false
	case "response.completed", "response.done":
		return true
	case "response.output_item.added":
		itemType := strings.TrimSpace(root.Get("item.type").String())
		return itemType != "" && itemType != "message" && itemType != "reasoning"
	}
	for _, path := range []string{
		"delta", "text", "arguments", "partial_image_b64", "audio", "result",
		"part.text", "part.refusal", "item.arguments", "item.result",
	} {
		if strings.TrimSpace(root.Get(path).String()) != "" {
			return true
		}
	}
	if content := root.Get("item.content"); content.IsArray() {
		for _, part := range content.Array() {
			if strings.TrimSpace(part.Get("text").String()) != "" || strings.TrimSpace(part.Get("refusal").String()) != "" {
				return true
			}
		}
	}
	if strings.HasSuffix(eventType, ".added") || strings.HasSuffix(eventType, ".in_progress") {
		return false
	}
	return eventType != ""
}

func sanitizeResponsesErrorValue(value string) string {
	return common.MaskSensitiveInfo(strings.TrimSpace(value))
}

func containsResponsesErrorMarker(value string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(value, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func firstResponsesErrorValue(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func ShouldRetryResponsesOnSameChannel(err *types.NewAPIError) bool {
	if err == nil || types.IsSkipRetryError(err) {
		return false
	}
	if types.IsSameChannelRetryError(err) {
		return true
	}
	if types.IsNextChannelRetryError(err) {
		return false
	}
	if err.StatusCode == http.StatusTooManyRequests || err.StatusCode == http.StatusRequestTimeout {
		return true
	}
	return err.StatusCode >= 500 && err.StatusCode <= 599
}

func CanRetryResponsesOnSameChannel(err *types.NewAPIError, failedAttempt int) bool {
	return failedAttempt > 0 && failedAttempt <= common.SameChannelRetryTimes && ShouldRetryResponsesOnSameChannel(err)
}

func ShouldRetryResponsesOnNextChannel(err *types.NewAPIError) bool {
	if err == nil || types.IsSkipRetryError(err) {
		return false
	}
	if types.IsNextChannelRetryError(err) || ShouldRetryResponsesOnSameChannel(err) {
		return true
	}
	return err.StatusCode == http.StatusUnauthorized || err.StatusCode == http.StatusPaymentRequired || err.StatusCode == http.StatusForbidden
}

func AppendResponsesRetryAttempt(c *gin.Context, attempt ResponsesRetryAttempt) {
	if c == nil {
		return
	}
	trace, _ := c.Get(responsesRetryTraceContextKey)
	attempts, _ := trace.([]ResponsesRetryAttempt)
	attempts = append(attempts, attempt)
	c.Set(responsesRetryTraceContextKey, attempts)
	if _, exists := c.Get(responsesRetrySuccessContextKey); !exists {
		c.Set(responsesRetrySuccessContextKey, false)
	}
}

func HasResponsesRetryAttempts(c *gin.Context) bool {
	if c == nil {
		return false
	}
	trace, ok := c.Get(responsesRetryTraceContextKey)
	attempts, valid := trace.([]ResponsesRetryAttempt)
	return ok && valid && len(attempts) > 0
}

func MarkResponsesRetrySuccess(c *gin.Context) {
	if c != nil {
		c.Set(responsesRetrySuccessContextKey, true)
	}
}

func responsesRetryAffinityOutcome(c *gin.Context) (bool, bool) {
	if c == nil {
		return false, false
	}
	value, exists := c.Get(responsesRetrySuccessContextKey)
	success, ok := value.(bool)
	return success, exists && ok
}

func AppendResponsesRetryAdminInfo(c *gin.Context, other interface{ SetAdmin(string, any) bool }) {
	if c == nil || other == nil {
		return
	}
	trace, ok := c.Get(responsesRetryTraceContextKey)
	if !ok {
		return
	}
	attempts, ok := trace.([]ResponsesRetryAttempt)
	if !ok || len(attempts) == 0 {
		return
	}
	other.SetAdmin("responses_retry", map[string]any{
		"attempts":      attempts,
		"attempt_count": len(attempts),
	})
}

var (
	responsesCooldownOnce  sync.Once
	responsesCooldownCache *cachex.HybridCache[int]
	responsesCooldownLocal *hot.HotCache[string, int]
)

func getResponsesCooldownCaches() (*cachex.HybridCache[int], *hot.HotCache[string, int]) {
	responsesCooldownOnce.Do(func() {
		responsesCooldownLocal = hot.NewHotCache[string, int](hot.LRU, 100_000).
			WithTTL(30 * time.Second).
			WithJanitor().
			Build()
		responsesCooldownCache = cachex.NewHybridCache[int](cachex.HybridCacheConfig[int]{
			Namespace: responsesCooldownNamespace,
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.IntCodec{},
			Memory: func() *hot.HotCache[string, int] {
				return responsesCooldownLocal
			},
		})
	})
	return responsesCooldownCache, responsesCooldownLocal
}

func responsesCooldownKey(channelID int, modelName string) string {
	return strconv.Itoa(channelID) + ":" + strings.ToLower(strings.TrimSpace(modelName))
}

func MarkResponsesChannelCooldown(channelID int, modelNames ...string) {
	seconds := common.ResponsesChannelCooldownSeconds
	if channelID <= 0 || seconds <= 0 {
		return
	}
	ttl := time.Duration(seconds) * time.Second
	cache, local := getResponsesCooldownCaches()
	seen := make(map[string]struct{}, len(modelNames))
	for _, modelName := range modelNames {
		key := responsesCooldownKey(channelID, modelName)
		if key == strconv.Itoa(channelID)+":" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		local.SetWithTTL(cache.FullKey(key), 1, ttl)
		if err := cache.SetWithTTL(key, 1, ttl); err != nil {
			logger.LogWarn(nil, fmt.Sprintf("responses channel cooldown Redis write failed, using local fallback: %v", err))
		}
	}
}

func IsResponsesChannelCooling(channelID int, modelName string) bool {
	if channelID <= 0 || common.ResponsesChannelCooldownSeconds <= 0 {
		return false
	}
	key := responsesCooldownKey(channelID, modelName)
	cache, local := getResponsesCooldownCaches()
	_, found, err := cache.Get(key)
	if err == nil && found {
		return true
	}
	if err != nil {
		logger.LogWarn(nil, fmt.Sprintf("responses channel cooldown Redis read failed, using local fallback: %v", err))
	}
	_, found, _ = local.Get(cache.FullKey(key))
	return found
}
