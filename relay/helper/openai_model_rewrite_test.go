package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsOpenAICompatibleRelayFormat(t *testing.T) {
	t.Parallel()

	compatible := []types.RelayFormat{
		types.RelayFormatOpenAI,
		types.RelayFormatOpenAIResponses,
		types.RelayFormatOpenAIResponsesCompaction,
		types.RelayFormatOpenAIAlphaSearch,
		types.RelayFormatOpenAIAudio,
		types.RelayFormatOpenAIImage,
		types.RelayFormatOpenAIRealtime,
		types.RelayFormatRerank,
		types.RelayFormatEmbedding,
		types.RelayFormatTask,
	}
	for _, format := range compatible {
		assert.Truef(t, IsOpenAICompatibleRelayFormat(format), "%s should rewrite OpenAI model fields", format)
	}

	assert.False(t, IsOpenAICompatibleRelayFormat(types.RelayFormatClaude))
	assert.False(t, IsOpenAICompatibleRelayFormat(types.RelayFormatGemini))
	assert.False(t, IsOpenAICompatibleRelayFormat(types.RelayFormatMjProxy))
}

func TestStringDataRewritesOpenAIMappedModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "gpt-5.5")
	EnableOpenAICompatibleModelRewrite(c, types.RelayFormatOpenAI)
	require.NoError(t, StringData(c, `{"id":"chatcmpl-1","model":"deepseek-flash"}`))
	assert.Contains(t, recorder.Body.String(), `"model":"gpt-5.5"`)
	assert.NotContains(t, recorder.Body.String(), "deepseek-flash")
}

func TestStringDataLeavesClaudeNativeModelUnchanged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "gpt-5.5")
	EnableOpenAICompatibleModelRewrite(c, types.RelayFormatClaude)
	require.NoError(t, StringData(c, `{"id":"msg_1","model":"claude-sonnet-4","type":"message"}`))
	assert.Contains(t, recorder.Body.String(), `"model":"claude-sonnet-4"`)
	assert.NotContains(t, recorder.Body.String(), "gpt-5.5")
}
