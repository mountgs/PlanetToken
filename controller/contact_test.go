package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetContactRevalidatesAdministratorContent(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	original := common.OptionMap
	common.OptionMap = map[string]string{"Contact": "", "Secret": "not public"}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = original
		common.OptionMapRWMutex.Unlock()
	})
	router := gin.New()
	router.GET("/api/contact", GetContact)
	request := func(etag string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/contact", nil)
		req.Header.Set("If-None-Match", etag)
		router.ServeHTTP(response, req)
		return response
	}
	empty := request("")
	require.Equal(t, http.StatusOK, empty.Code)
	var payload publicContentResponse
	require.NoError(t, common.Unmarshal(empty.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Empty(t, payload.Data)
	etag := empty.Header().Get("ETag")
	require.NotEmpty(t, etag)
	require.Equal(t, http.StatusNotModified, request(etag).Code)

	content := "Email: help@example.com\n\n![WeChat](https://example.com/qr.png)"
	common.OptionMapRWMutex.Lock()
	common.OptionMap["Contact"] = content
	common.OptionMapRWMutex.Unlock()
	updated := request(etag)
	require.Equal(t, http.StatusOK, updated.Code)
	require.Equal(t, "no-cache", updated.Header().Get("Cache-Control"))
	require.NotEqual(t, etag, updated.Header().Get("ETag"))
	require.NoError(t, common.Unmarshal(updated.Body.Bytes(), &payload))
	require.Equal(t, content, payload.Data)
	require.NotContains(t, updated.Body.String(), "not public")
}
