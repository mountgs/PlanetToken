package common

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteOpenAICompatibleModelJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		origin        string
		wantModel     string
		nestedKey     string
		wantNested    string
		wantUnchanged bool
	}{
		{
			name:      "chat completion model",
			input:     `{"id":"chatcmpl-1","model":"deepseek-flash","choices":[]}`,
			origin:    "gpt-5.5",
			wantModel: "gpt-5.5",
		},
		{
			name:       "responses stream event",
			input:      `{"type":"response.created","response":{"id":"resp_1","model":"deepseek-flash"}}`,
			origin:     "gpt-5.5",
			nestedKey:  "response",
			wantNested: "gpt-5.5",
		},
		{
			name:       "error model field",
			input:      `{"error":{"message":"upstream failed","model":"deepseek-flash","type":"server_error"}}`,
			origin:     "gpt-5.5",
			nestedKey:  "error",
			wantNested: "gpt-5.5",
		},
		{
			name:       "realtime session model",
			input:      `{"type":"session.created","session":{"model":"gpt-4o-realtime-preview"}}`,
			origin:     "alias-realtime",
			nestedKey:  "session",
			wantNested: "alias-realtime",
		},
		{
			name:          "leaves already matching model unchanged",
			input:         `{"model":"gpt-5.5"}`,
			origin:        "gpt-5.5",
			wantUnchanged: true,
		},
		{
			name:          "leaves non-json control frames unchanged",
			input:         "[DONE]",
			origin:        "gpt-5.5",
			wantUnchanged: true,
		},
		{
			name:          "does not rewrite free-text error messages",
			input:         `{"error":{"message":"model deepseek-flash is unavailable"}}`,
			origin:        "gpt-5.5",
			wantUnchanged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RewriteOpenAICompatibleModelJSON([]byte(tt.input), tt.origin)
			if tt.wantUnchanged {
				assert.Equal(t, tt.input, string(got))
				return
			}
			var parsed map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(got, &parsed))
			if tt.wantModel != "" {
				var model string
				require.NoError(t, json.Unmarshal(parsed["model"], &model))
				assert.Equal(t, tt.wantModel, model)
			}
			if tt.nestedKey != "" {
				var nested map[string]string
				require.NoError(t, json.Unmarshal(parsed[tt.nestedKey], &nested))
				assert.Equal(t, tt.wantNested, nested["model"])
			}
			if tt.nestedKey == "error" {
				var nested map[string]string
				require.NoError(t, json.Unmarshal(parsed["error"], &nested))
				assert.Equal(t, "upstream failed", nested["message"])
			}
		})
	}
}

func TestRewriteOutgoingOpenAICompatibleModelHonorsContextFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)
	payload := []byte(`{"model":"deepseek-flash"}`)

	recorder := httptest.NewRecorder()
	disabled, _ := gin.CreateTestContext(recorder)
	SetContextKey(disabled, constant.ContextKeyOriginalModel, "gpt-5.5")
	assert.Equal(t, payload, RewriteOutgoingOpenAICompatibleModel(disabled, payload))

	enabled, _ := gin.CreateTestContext(httptest.NewRecorder())
	SetContextKey(enabled, constant.ContextKeyOriginalModel, "gpt-5.5")
	SetContextKey(enabled, constant.ContextKeyRewriteOpenAICompatibleModel, true)
	rewritten := RewriteOutgoingOpenAICompatibleModel(enabled, payload)
	var parsed map[string]string
	require.NoError(t, json.Unmarshal(rewritten, &parsed))
	assert.Equal(t, "gpt-5.5", parsed["model"])

	assert.Equal(t, "[DONE]", RewriteOutgoingOpenAICompatibleModelString(enabled, "[DONE]"))
}
