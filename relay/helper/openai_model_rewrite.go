package helper

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

// EnableOpenAICompatibleModelRewrite marks this request so OpenAI-shaped JSON
// `model` fields are rewritten to the client's requested model before they are
// written. Claude and Gemini native formats are intentionally excluded.
func EnableOpenAICompatibleModelRewrite(c *gin.Context, format types.RelayFormat) {
	if c == nil || !IsOpenAICompatibleRelayFormat(format) {
		return
	}
	common.SetContextKey(c, constant.ContextKeyRewriteOpenAICompatibleModel, true)
}

func IsOpenAICompatibleRelayFormat(format types.RelayFormat) bool {
	switch format {
	case types.RelayFormatOpenAI,
		types.RelayFormatOpenAIResponses,
		types.RelayFormatOpenAIResponsesCompaction,
		types.RelayFormatOpenAIAlphaSearch,
		types.RelayFormatOpenAIAudio,
		types.RelayFormatOpenAIImage,
		types.RelayFormatOpenAIRealtime,
		types.RelayFormatRerank,
		types.RelayFormatEmbedding,
		types.RelayFormatTask:
		return true
	default:
		return false
	}
}
