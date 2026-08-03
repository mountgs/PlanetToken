package codex

import (
	"slices"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

var baseModelList = []string{
	"gpt-5.6-sol",
	"gpt-5.6-terra",
	"gpt-5.6-luna",
	"gpt-5.5",
	"gpt-5.4",
	"gpt-5.4-mini",
	"gpt-5.3-codex-spark",
	"codex-auto-review",
}

const (
	CodexImageModel        = "gpt-image-2"
	defaultImagesMainModel = "gpt-5.4-mini"
	imageGenerationTool    = "image_generation"
)

var builtinModelList = []string{
	CodexImageModel,
}

var ModelList = slices.DeleteFunc(
	append(ratio_setting.WithCompactModelVariants(baseModelList), builtinModelList...),
	func(modelName string) bool {
		return modelName == ratio_setting.WithCompactModelSuffix("codex-auto-review")
	},
)

const ChannelName = "codex"
