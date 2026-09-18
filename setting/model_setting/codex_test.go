package model_setting

import "testing"

func TestGetCodexImagesMainModel(t *testing.T) {
	original := codexSettings.ImagesMainModel
	t.Cleanup(func() {
		codexSettings.ImagesMainModel = original
	})

	t.Run("uses configured model", func(t *testing.T) {
		codexSettings.ImagesMainModel = "gpt-5.6-terra"
		if got := GetCodexImagesMainModel(); got != "gpt-5.6-terra" {
			t.Fatalf("GetCodexImagesMainModel() = %q, want gpt-5.6-terra", got)
		}
	})

	t.Run("empty falls back to default", func(t *testing.T) {
		codexSettings.ImagesMainModel = "  "
		if got := GetCodexImagesMainModel(); got != DefaultCodexImagesMainModel {
			t.Fatalf("GetCodexImagesMainModel() = %q, want %q", got, DefaultCodexImagesMainModel)
		}
	})
}
