package helper

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestGetAndValidOpenAIImageRequestMultipartStream verifies multipart image
// edit parsing: the stream field is parsed and validated, and the request body
// stays replayable for the upstream request.
func TestGetAndValidOpenAIImageRequestMultipartStream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func(t *testing.T, streamValue string, withImage bool) (*gin.Context, string) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-image-1"))
		require.NoError(t, writer.WriteField("prompt", "edit this image"))
		require.NoError(t, writer.WriteField("stream", streamValue))
		if withImage {
			part, err := writer.CreateFormFile("image", "input.png")
			require.NoError(t, err)
			_, err = part.Write([]byte("fake image"))
			require.NoError(t, err)
		}
		require.NoError(t, writer.Close())
		originalBody := body.String()

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return c, originalBody
	}

	t.Run("valid stream value keeps body replayable", func(t *testing.T) {
		c, originalBody := newContext(t, "true", true)

		req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
		require.NoError(t, err)
		require.NotNil(t, req.Stream)
		require.True(t, *req.Stream)
		require.True(t, req.IsStream(c.Request))

		bodyAfterValidation, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Equal(t, originalBody, string(bodyAfterValidation))

		form, err := common.ParseMultipartFormReusable(c)
		require.NoError(t, err)
		require.Equal(t, "true", url.Values(form.Value).Get("stream"))
		require.Len(t, form.File["image"], 1)
	})

	t.Run("invalid stream value is rejected", func(t *testing.T) {
		c, _ := newContext(t, "notabool", false)

		_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid stream value")
	})
}

func TestGetAndValidOpenAIImageRequestStreamContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		body       string
		wantStream *bool
	}{
		{name: "absent", body: `{"model":"gpt-image-2","prompt":"draw"}`},
		{name: "false", body: `{"model":"gpt-image-2","prompt":"draw","stream":false}`, wantStream: common.GetPointer(false)},
		{name: "true", body: `{"model":"gpt-image-2","prompt":"draw","stream":true}`, wantStream: common.GetPointer(true)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")

			req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
			require.NoError(t, err)
			if tt.wantStream == nil {
				require.Nil(t, req.Stream)
				require.False(t, req.IsStream(c.Request))
				return
			}
			require.NotNil(t, req.Stream)
			require.Equal(t, *tt.wantStream, *req.Stream)
			require.Equal(t, *tt.wantStream, req.IsStream(c.Request))
		})
	}
}

func TestGetAndValidOpenAIImageRequestPartialImagesBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, tt := range []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "zero", value: "0"},
		{name: "maximum", value: "3"},
		{name: "negative", value: "-1", wantErr: true},
		{name: "above maximum", value: "4", wantErr: true},
		{name: "fraction", value: "1.5", wantErr: true},
		{name: "string", value: `"2"`, wantErr: true},
	} {
		t.Run("json/"+tt.name, func(t *testing.T) {
			body := fmt.Sprintf(`{"model":"gpt-image-2","prompt":"draw","partial_images":%s}`, tt.value)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(body))
			c.Request.Header.Set("Content-Type", "application/json")

			req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "partial_images must be an integer between 0 and 3")
				return
			}
			require.NoError(t, err)
			require.JSONEq(t, tt.value, string(req.PartialImages))
		})
	}

	t.Run("absent remains omitted", func(t *testing.T) {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(`{"model":"gpt-image-2","prompt":"draw","stream":true}`))
		c.Request.Header.Set("Content-Type", "application/json")

		req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
		require.NoError(t, err)
		require.Empty(t, req.PartialImages)
	})

	for _, tt := range []struct {
		value   string
		wantErr bool
	}{
		{value: "0"},
		{value: "3"},
		{value: "-1", wantErr: true},
		{value: "4", wantErr: true},
		{value: "1.5", wantErr: true},
	} {
		t.Run("multipart/"+tt.value, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			require.NoError(t, writer.WriteField("model", "gpt-image-2"))
			require.NoError(t, writer.WriteField("prompt", "edit"))
			require.NoError(t, writer.WriteField("partial_images", tt.value))
			require.NoError(t, writer.Close())
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
			c.Request.Header.Set("Content-Type", writer.FormDataContentType())

			req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "partial_images must be an integer between 0 and 3")
				return
			}
			require.NoError(t, err)
			require.JSONEq(t, tt.value, string(req.PartialImages))
		})
	}
}

// TestGetAndValidOpenAIImageRequestNBounds guards the billing invariant that
// the image generation count can never reach quota calculation with a value
// large enough to overflow int64 into a negative charge.
func TestGetAndValidOpenAIImageRequestNBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newJSONContext := func(t *testing.T, body string) *gin.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
		return c
	}

	boundErr := fmt.Sprintf("n must be an integer between 1 and %d", dto.MaxImageN)

	tests := []struct {
		name    string
		body    string
		wantErr string
		wantN   uint
	}{
		{
			name:    "overflowed uint64 n is rejected",
			body:    `{"model":"gpt-image-1","prompt":"a cat","n":18446744073686646784}`,
			wantErr: boundErr,
		},
		{
			name:    "n above max is rejected",
			body:    fmt.Sprintf(`{"model":"gpt-image-1","prompt":"a cat","n":%d}`, dto.MaxImageN+1),
			wantErr: boundErr,
		},
		{
			name:  "n at max is accepted",
			body:  fmt.Sprintf(`{"model":"gpt-image-1","prompt":"a cat","n":%d}`, dto.MaxImageN),
			wantN: dto.MaxImageN,
		},
		{
			name:  "explicit n is accepted",
			body:  `{"model":"gpt-image-1","prompt":"a cat","n":3}`,
			wantN: 3,
		},
		{
			name:  "zero n defaults to 1",
			body:  `{"model":"gpt-image-1","prompt":"a cat","n":0}`,
			wantN: 1,
		},
		{
			name:  "absent n defaults to 1",
			body:  `{"model":"gpt-image-1","prompt":"a cat"}`,
			wantN: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newJSONContext(t, tt.body)
			req, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesGenerations)
			if tt.wantErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, req.N)
			require.Equal(t, tt.wantN, *req.N)
			require.Equal(t, float64(tt.wantN), req.GetTokenCountMeta().BillingRatios["n"])
		})
	}

	t.Run("negative multipart n is rejected", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		require.NoError(t, writer.WriteField("model", "gpt-image-1"))
		require.NoError(t, writer.WriteField("prompt", "edit this image"))
		require.NoError(t, writer.WriteField("n", "-22904832"))
		require.NoError(t, writer.Close())

		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		_, err := GetAndValidOpenAIImageRequest(c, relayconstant.RelayModeImagesEdits)
		require.Error(t, err)
		require.Contains(t, err.Error(), boundErr)
	})
}
