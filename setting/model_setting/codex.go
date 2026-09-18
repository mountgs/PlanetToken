package model_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

const DefaultCodexImagesMainModel = "gpt-5.6-luna"

// CodexSettings defines Codex model configuration.
type CodexSettings struct {
	ImagesMainModel string `json:"images_main_model"`
}

var defaultCodexSettings = CodexSettings{
	ImagesMainModel: DefaultCodexImagesMainModel,
}

var codexSettings = defaultCodexSettings

func init() {
	config.GlobalConfig.Register("codex", &codexSettings)
}

func GetCodexSettings() *CodexSettings {
	return &codexSettings
}

// GetCodexImagesMainModel returns the Responses main model used when Codex
// OAuth accounts generate or edit images. Empty values fall back to the
// current ChatGPT-signed default.
func GetCodexImagesMainModel() string {
	model := strings.TrimSpace(codexSettings.ImagesMainModel)
	if model == "" {
		return DefaultCodexImagesMainModel
	}
	return model
}
