package codex

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCodexImageSSEDecoderPreservesFrameFields(t *testing.T) {
	decoder := newCodexImageSSEDecoder(strings.NewReader(": keepalive\r\nevent: response.completed\r\ndata: {\"type\":\r\ndata: \"response.completed\"}\r\n\r\n"))

	frame, err := decoder.Next()
	require.NoError(t, err)
	require.Equal(t, "response.completed", frame.Event)
	require.Equal(t, "{\"type\":\n\"response.completed\"}", string(frame.Data))
	require.Equal(t, []string{" keepalive"}, frame.Comments)
}

func TestCodexImageSSEDecoderReturnsFrameBeforeEOF(t *testing.T) {
	reader, writer := io.Pipe()
	decoder := newCodexImageSSEDecoder(reader)
	result := make(chan codexImageSSEFrame, 1)
	errResult := make(chan error, 1)
	go func() {
		frame, err := decoder.Next()
		result <- frame
		errResult <- err
	}()

	_, err := io.WriteString(writer, "data: first\n\n")
	require.NoError(t, err)
	frame := <-result
	require.Equal(t, "first", string(frame.Data))
	require.NoError(t, <-errResult)
	require.NoError(t, writer.Close())
}

func TestCodexImageSSEDecoderEnforcesFrameLimit(t *testing.T) {
	decoder := newCodexImageSSEDecoder(strings.NewReader("data: 123456789\n\n"))
	decoder.maxFrameSize = 8

	_, err := decoder.Next()
	require.ErrorIs(t, err, errCodexImageSSEFrameTooLarge)
}

func TestHandleCodexImageStreamMapsGenerationCompleted(t *testing.T) {
	body := `data: {"type":"response.completed","response":{"created_at":123,"output":[{"type":"image_generation_call","result":"aGVsbG8=","output_format":"png","size":"1024x1024"}],"usage":{"input_tokens":1,"output_tokens":2,"total_tokens":3}}}` + "\n\n"
	c, recorder, resp, info := newCodexImageStreamTest(t, body, relayconstant.RelayModeImagesGenerations, "b64_json")
	info.PriceData.UsePrice = true
	info.PriceData.AddOtherRatio("n", 3)

	usage, apiErr := handleImageStreamResponse(c, resp, info)

	require.Nil(t, apiErr)
	require.Equal(t, 3, usage.TotalTokens)
	require.Contains(t, recorder.Body.String(), "event: image_generation.completed")
	require.Contains(t, recorder.Body.String(), `"b64_json":"aGVsbG8="`)
	require.Contains(t, recorder.Body.String(), "data: [DONE]")
	require.Equal(t, 1.0, info.PriceData.OtherRatios()["n"])
}

func TestHandleCodexImageStreamMapsEditPartialAndURL(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"response.image_generation_call.partial_image","partial_image_b64":"cGFydGlhbA==","output_format":"webp","partial_image_index":0}`,
		``,
		`data: {"type":"response.completed","response":{"created_at":123,"output":[{"type":"image_generation_call","result":"ZmluYWw=","output_format":"webp"}]}}`,
		``,
	}, "\n")
	c, recorder, resp, info := newCodexImageStreamTest(t, body, relayconstant.RelayModeImagesEdits, "url")

	_, apiErr := handleImageStreamResponse(c, resp, info)

	require.Nil(t, apiErr)
	require.Contains(t, recorder.Body.String(), "event: image_edit.partial_image")
	require.Contains(t, recorder.Body.String(), "event: image_edit.completed")
	require.Contains(t, recorder.Body.String(), `data:image/webp;base64,cGFydGlhbA==`)
	require.Contains(t, recorder.Body.String(), `data:image/webp;base64,ZmluYWw=`)
}

func TestHandleCodexImageStreamPassesStructuredErrorAfterPartial(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"response.image_generation_call.partial_image","partial_image_b64":"cGFydGlhbA==","partial_image_index":0}`,
		``,
		`data: {"type":"response.failed","response":{"id":"resp_123","error":{"message":"Your request was rejected by the safety system.","type":"image_generation_user_error","code":"moderation_blocked","param":"prompt"}}}`,
		``,
	}, "\n")
	c, recorder, resp, info := newCodexImageStreamTest(t, body, relayconstant.RelayModeImagesGenerations, "b64_json")

	_, apiErr := handleImageStreamResponse(c, resp, info)

	require.Nil(t, apiErr)
	require.Contains(t, recorder.Body.String(), "event: error")
	require.Contains(t, recorder.Body.String(), `"message":"Your request was rejected by the safety system."`)
	require.Contains(t, recorder.Body.String(), `"type":"image_generation_user_error"`)
	require.Contains(t, recorder.Body.String(), `"code":"moderation_blocked"`)
	require.Contains(t, recorder.Body.String(), `"param":"prompt"`)
	require.Contains(t, recorder.Body.String(), "data: [DONE]")
	require.True(t, info.StreamStatus.HasErrors())
}

func TestHandleCodexImageStreamSanitizesStructuredErrorAfterPartial(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"response.image_generation_call.partial_image","partial_image_b64":"cGFydGlhbA==","partial_image_index":0}`,
		``,
		`data: {"type":"response.failed","response":{"error":{"message":"Bearer secret-token at https://internal.example","type":"image_generation_user_error","code":"moderation_blocked","param":"prompt"}}}`,
		``,
	}, "\n")
	c, recorder, resp, info := newCodexImageStreamTest(t, body, relayconstant.RelayModeImagesGenerations, "b64_json")

	_, apiErr := handleImageStreamResponse(c, resp, info)

	require.Nil(t, apiErr)
	require.Contains(t, recorder.Body.String(), "event: error")
	require.NotContains(t, recorder.Body.String(), "secret-token")
	require.NotContains(t, recorder.Body.String(), "internal.example")
	require.Contains(t, recorder.Body.String(), `"type":"image_generation_user_error"`)
	require.Contains(t, recorder.Body.String(), `"code":"moderation_blocked"`)
	require.Contains(t, recorder.Body.String(), `"param":"prompt"`)
	require.True(t, info.StreamStatus.HasErrors())
}

func TestCodexImageStreamIncompleteContentFilterIsNotRetryable(t *testing.T) {
	body := `data: {"type":"response.incomplete","response":{"id":"resp_123","status":"incomplete","incomplete_details":{"reason":"content_filter"}}}` + "\n\n"
	c, recorder, resp, info := newCodexImageStreamTest(t, body, relayconstant.RelayModeImagesGenerations, "b64_json")

	usage, apiErr := handleImageStreamResponse(c, resp, info)

	require.Nil(t, usage)
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.True(t, types.IsSkipRetryError(apiErr))
	require.Equal(t, "response_incomplete", string(apiErr.GetErrorCode()))
	require.Contains(t, apiErr.Error(), "content_filter")
	require.Empty(t, recorder.Body.String())
}

func TestCollectCodexImageSSEPassesCompletedRefusal(t *testing.T) {
	body := strings.Join([]string{
		`data: {"type":"response.completed","response":{"output":[{"type":"message","content":[{"type":"output_text","text":"The safety system rejected this image request."}]}]}}`,
		``,
	}, "\n")

	_, _, _, _, apiErr := collectImagesFromSSE([]byte(body))

	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	require.Contains(t, apiErr.Error(), "The safety system rejected this image request.")
	require.True(t, types.IsSkipRetryError(apiErr))
}

func newCodexImageStreamTest(t *testing.T, body string, mode int, responseFormat string) (*gin.Context, *httptest.ResponseRecorder, *http.Response, *relaycommon.RelayInfo) {
	t.Helper()
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	c.Set(ginKeyCodexImageResponseFormat, responseFormat)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	info := &relaycommon.RelayInfo{RelayMode: mode, IsStream: true, ChannelMeta: &relaycommon.ChannelMeta{}}
	return c, recorder, resp, info
}

func TestCodexImageStreamErrorBeforeOutputRemainsRetryable(t *testing.T) {
	c, recorder, resp, info := newCodexImageStreamTest(t, `data: {"type":"response.failed","error":{"message":"temporary"}}`+"\n\n", relayconstant.RelayModeImagesGenerations, "b64_json")

	usage, apiErr := handleImageStreamResponse(c, resp, info)

	require.Nil(t, usage)
	require.NotNil(t, apiErr)
	require.Empty(t, recorder.Body.String())
	require.False(t, c.GetBool(constant.ContextKeyCodexImageRealOutput))
}

func TestCodexImageSSEDecoderDoneFrame(t *testing.T) {
	decoder := newCodexImageSSEDecoder(strings.NewReader("data: [DONE]\n\n"))
	frame, err := decoder.Next()
	require.NoError(t, err)
	require.Equal(t, "[DONE]", string(frame.Data))
	_, err = decoder.Next()
	require.True(t, errors.Is(err, io.EOF))
}

type closeBlockingBody struct {
	closed chan struct{}
	once   sync.Once
}

func (b *closeBlockingBody) Read([]byte) (int, error) {
	<-b.closed
	return 0, io.EOF
}

func (b *closeBlockingBody) Close() error {
	b.once.Do(func() { close(b.closed) })
	return nil
}

func TestHandleCodexImageStreamDisconnectClosesUpstreamBody(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil).WithContext(requestContext)
	body := &closeBlockingBody{closed: make(chan struct{})}
	resp := &http.Response{StatusCode: http.StatusOK, Body: body, Header: http.Header{"Content-Type": []string{"text/event-stream"}}}
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesGenerations, IsStream: true}
	cancel()

	usage, apiErr := handleImageStreamResponse(c, resp, info)

	require.Nil(t, apiErr)
	require.NotNil(t, usage)
	select {
	case <-body.closed:
	default:
		t.Fatal("upstream body was not closed after client disconnect")
	}
	require.Equal(t, relaycommon.StreamEndReasonClientGone, info.StreamStatus.EndReason)
}

type testImageStreamTimer struct {
	ch chan time.Time
}

func (t *testImageStreamTimer) Chan() <-chan time.Time { return t.ch }
func (t *testImageStreamTimer) Reset(time.Duration)    {}
func (t *testImageStreamTimer) Stop()                  {}

type cancelOnHeartbeatWriter struct {
	gin.ResponseWriter
	cancel context.CancelFunc
	once   sync.Once
}

func (w *cancelOnHeartbeatWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if string(data) == ":\n\n" {
		w.once.Do(w.cancel)
	}
	return n, err
}

func TestHandleCodexImageStreamHeartbeatIsNotRealOutput(t *testing.T) {
	originalFactory := newImageStreamTimer
	originalInterval := constant.ImageStreamPingInterval
	timer := &testImageStreamTimer{ch: make(chan time.Time, 1)}
	timer.ch <- time.Now()
	newImageStreamTimer = func(time.Duration) imageStreamTimer { return timer }
	constant.ImageStreamPingInterval = 10
	t.Cleanup(func() {
		newImageStreamTimer = originalFactory
		constant.ImageStreamPingInterval = originalInterval
	})

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	requestContext, cancel := context.WithCancel(context.Background())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil).WithContext(requestContext)
	c.Writer = &cancelOnHeartbeatWriter{ResponseWriter: c.Writer, cancel: cancel}
	body := &closeBlockingBody{closed: make(chan struct{})}
	resp := &http.Response{StatusCode: http.StatusOK, Body: body, Header: http.Header{"Content-Type": []string{"text/event-stream"}}}
	info := &relaycommon.RelayInfo{RelayMode: relayconstant.RelayModeImagesGenerations, IsStream: true}

	_, apiErr := handleImageStreamResponse(c, resp, info)

	require.Nil(t, apiErr)
	require.Equal(t, ":\n\n", recorder.Body.String())
	require.True(t, c.GetBool(constant.ContextKeyCodexImageStreamCommitted))
	require.False(t, c.GetBool(constant.ContextKeyCodexImageRealOutput))
}
