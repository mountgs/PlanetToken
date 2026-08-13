package codex

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/tidwall/gjson"
)

const maxCodexImageErrorFieldLength = 2048

var codexImageBearerPattern = regexp.MustCompile(`(?i)\bbearer\s+[^\s,;]+`)

type codexImageUpstreamError struct {
	StatusCode int
	Type       string
	Code       string
	Message    string
	Param      string
}

func (e *codexImageUpstreamError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Code != "" {
		return e.Code
	}
	return "upstream image generation failed"
}

func parseCodexImageUpstreamError(payload []byte) *codexImageUpstreamError {
	if !gjson.ValidBytes(payload) {
		return nil
	}
	eventType := strings.TrimSpace(gjson.GetBytes(payload, "type").String())
	switch eventType {
	case "response.error", "error":
		return codexImageUpstreamErrorFromJSON(gjson.GetBytes(payload, "error"))
	case "response.failed":
		errorObject := gjson.GetBytes(payload, "response.error")
		if !errorObject.Exists() {
			errorObject = gjson.GetBytes(payload, "error")
		}
		return codexImageUpstreamErrorFromJSON(errorObject)
	case "response.incomplete":
		reason := sanitizeCodexImageErrorField(gjson.GetBytes(payload, "response.incomplete_details.reason").String())
		if reason == "" {
			reason = "unknown"
		}
		statusCode := http.StatusBadGateway
		errorType := "incomplete_error"
		if isCodexImagePolicyError(reason, reason) {
			statusCode = http.StatusBadRequest
			errorType = "image_generation_user_error"
		}
		return &codexImageUpstreamError{
			StatusCode: statusCode,
			Type:       errorType,
			Code:       "response_incomplete",
			Message:    "Upstream image generation incomplete: " + reason,
		}
	default:
		return nil
	}
}

func codexImageUpstreamErrorFromJSON(errorObject gjson.Result) *codexImageUpstreamError {
	if !errorObject.Exists() {
		return nil
	}
	errorType := sanitizeCodexImageErrorField(errorObject.Get("type").String())
	code := sanitizeCodexImageErrorField(errorObject.Get("code").String())
	message := sanitizeCodexImageErrorField(errorObject.Get("message").String())
	param := sanitizeCodexImageErrorField(errorObject.Get("param").String())
	if message == "" {
		message = "upstream image generation failed"
	}
	return &codexImageUpstreamError{
		StatusCode: codexImageErrorStatus(errorType, code),
		Type:       firstNonEmpty(errorType, "upstream_error"),
		Code:       firstNonEmpty(code, "image_stream_error"),
		Message:    message,
		Param:      param,
	}
}

func codexImageErrorStatus(errorType, code string) int {
	value := strings.ToLower(strings.TrimSpace(errorType + " " + code))
	switch {
	case strings.Contains(value, "rate_limit"):
		return http.StatusTooManyRequests
	case strings.Contains(value, "authentication"), strings.Contains(value, "invalid_api_key"), strings.Contains(value, "unauthorized"):
		return http.StatusUnauthorized
	case strings.Contains(value, "permission"), strings.Contains(value, "forbidden"):
		return http.StatusForbidden
	case strings.Contains(value, "not_found"):
		return http.StatusNotFound
	case strings.Contains(value, "invalid_request"), isCodexImagePolicyError(errorType, code):
		return http.StatusBadRequest
	default:
		return http.StatusBadGateway
	}
}

func isCodexImagePolicyError(errorType, code string) bool {
	value := strings.ToLower(strings.TrimSpace(errorType + " " + code))
	for _, marker := range []string{"content_filter", "moderation", "policy_violation", "safety_violation", "moderation_blocked", "image_generation_user_error"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

func sanitizeCodexImageErrorField(value string) string {
	value = strings.TrimSpace(codexImageBearerPattern.ReplaceAllString(value, "Bearer ***"))
	value = common.MaskSensitiveInfo(value)
	if len(value) <= maxCodexImageErrorFieldLength {
		return value
	}
	return value[:maxCodexImageErrorFieldLength] + "...(truncated)"
}

func (e *codexImageUpstreamError) toNewAPIError() *types.NewAPIError {
	if e == nil {
		return types.NewOpenAIError(fmt.Errorf("upstream image generation failed"), types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}
	openAIError := types.OpenAIError{
		Message: e.Error(),
		Type:    firstNonEmpty(e.Type, "upstream_error"),
		Code:    firstNonEmpty(e.Code, "image_stream_error"),
		Param:   e.Param,
	}
	options := []types.NewAPIErrorOptions{}
	if isCodexImagePolicyError(e.Type, e.Code) {
		options = append(options, types.ErrOptionWithSkipRetry())
	}
	return types.WithOpenAIError(openAIError, e.StatusCode, options...)
}

func codexImageRefusalMessage(payload []byte) string {
	root := gjson.ParseBytes(payload)
	var messages []string
	collect := func(value string) {
		if value = strings.TrimSpace(value); value != "" {
			messages = append(messages, value)
		}
	}
	collect(root.Get("delta").String())
	for _, path := range []string{"response.output", "item.content"} {
		root.Get(path).ForEach(func(_, item gjson.Result) bool {
			if item.Get("type").String() == "message" {
				item.Get("content").ForEach(func(_, part gjson.Result) bool {
					if part.Get("type").String() == "output_text" || part.Get("type").String() == "refusal" {
						collect(part.Get("text").String())
					}
					return true
				})
				return true
			}
			if item.Get("type").String() == "output_text" || item.Get("type").String() == "refusal" {
				collect(item.Get("text").String())
			}
			return true
		})
	}
	return sanitizeCodexImageErrorField(strings.Join(messages, " "))
}

func codexImageRefusalError(message string) *codexImageUpstreamError {
	message = sanitizeCodexImageErrorField(message)
	if message == "" {
		return nil
	}
	return &codexImageUpstreamError{
		StatusCode: http.StatusBadRequest,
		Type:       "image_generation_user_error",
		Code:       "content_policy_violation",
		Message:    message,
	}
}
