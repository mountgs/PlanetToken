package common

import (
	"bytes"
	"encoding/json"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

var openaiCompatibleModelObjectKeys = []string{"error", "response", "session"}

// RewriteOpenAICompatibleModelJSON replaces structured OpenAI-shaped `model`
// string fields with originModel. Non-objects, malformed JSON, and payloads
// without a string `model` field are returned unchanged. Free-text fields such
// as error.message are not rewritten.
func RewriteOpenAICompatibleModelJSON(data []byte, originModel string) []byte {
	if originModel == "" || len(data) == 0 {
		return data
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return data
	}
	var obj map[string]json.RawMessage
	if err := Unmarshal(trimmed, &obj); err != nil {
		return data
	}
	changed := rewriteJSONModelString(obj, originModel)
	for _, key := range openaiCompatibleModelObjectKeys {
		if rewriteNestedJSONModelString(obj, key, originModel) {
			changed = true
		}
	}
	if !changed {
		return data
	}
	formatted, err := Marshal(obj)
	if err != nil {
		return data
	}
	return formatted
}

// RewriteOutgoingOpenAICompatibleModel rewrites user-visible OpenAI JSON when
// the request opted into OpenAI-compatible model hiding. Native Claude/Gemini
// relay formats leave the context flag unset and pass through unchanged.
func RewriteOutgoingOpenAICompatibleModel(c *gin.Context, data []byte) []byte {
	if c == nil || !GetContextKeyBool(c, constant.ContextKeyRewriteOpenAICompatibleModel) {
		return data
	}
	return RewriteOpenAICompatibleModelJSON(data, GetContextKeyString(c, constant.ContextKeyOriginalModel))
}

// RewriteOutgoingOpenAICompatibleModelString is the streaming/SSE counterpart
// of RewriteOutgoingOpenAICompatibleModel. Control frames such as [DONE] are
// left untouched.
func RewriteOutgoingOpenAICompatibleModelString(c *gin.Context, data string) string {
	if data == "" || data == "[DONE]" {
		return data
	}
	rewritten := RewriteOutgoingOpenAICompatibleModel(c, StringToByteSlice(data))
	if len(rewritten) == 0 {
		return data
	}
	if bytes.Equal(rewritten, StringToByteSlice(data)) {
		return data
	}
	return string(rewritten)
}

func rewriteJSONModelString(obj map[string]json.RawMessage, originModel string) bool {
	raw, exists := obj["model"]
	if !exists {
		return false
	}
	var current string
	if err := Unmarshal(raw, &current); err != nil {
		return false
	}
	if current == originModel {
		return false
	}
	encoded, err := Marshal(originModel)
	if err != nil {
		return false
	}
	obj["model"] = encoded
	return true
}

func rewriteNestedJSONModelString(obj map[string]json.RawMessage, key, originModel string) bool {
	raw, exists := obj[key]
	if !exists {
		return false
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return false
	}
	var nested map[string]json.RawMessage
	if err := Unmarshal(trimmed, &nested); err != nil {
		return false
	}
	if !rewriteJSONModelString(nested, originModel) {
		return false
	}
	encoded, err := Marshal(nested)
	if err != nil {
		return false
	}
	obj[key] = encoded
	return true
}
