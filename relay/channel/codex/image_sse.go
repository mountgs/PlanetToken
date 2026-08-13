package codex

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

const maxCodexImageSSEFrameSize = 128 << 20

var errCodexImageSSEFrameTooLarge = errors.New("codex image SSE frame exceeds 128 MiB")

type codexImageSSEFrame struct {
	Event    string
	Data     []byte
	Comments []string
}

type codexImageSSEDecoder struct {
	reader       *bufio.Reader
	maxFrameSize int
}

type imageStreamTimer interface {
	Chan() <-chan time.Time
	Reset(time.Duration)
	Stop()
}

type realImageStreamTimer struct {
	timer *time.Timer
}

func (t *realImageStreamTimer) Chan() <-chan time.Time { return t.timer.C }
func (t *realImageStreamTimer) Reset(d time.Duration) {
	if !t.timer.Stop() {
		select {
		case <-t.timer.C:
		default:
		}
	}
	t.timer.Reset(d)
}
func (t *realImageStreamTimer) Stop() { t.timer.Stop() }

var newImageStreamTimer = func(d time.Duration) imageStreamTimer {
	return &realImageStreamTimer{timer: time.NewTimer(d)}
}

func newCodexImageSSEDecoder(reader io.Reader) *codexImageSSEDecoder {
	return &codexImageSSEDecoder{
		reader:       bufio.NewReaderSize(reader, 64<<10),
		maxFrameSize: maxCodexImageSSEFrameSize,
	}
}

func (d *codexImageSSEDecoder) Next() (codexImageSSEFrame, error) {
	var frame codexImageSSEFrame
	var dataLines []string
	var line []byte
	frameSize := 0

	for {
		fragment, err := d.reader.ReadSlice('\n')
		if len(fragment) > 0 {
			frameSize += len(fragment)
			if frameSize > d.maxFrameSize {
				return codexImageSSEFrame{}, errCodexImageSSEFrameTooLarge
			}
			line = append(line, fragment...)
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if len(line) > 0 {
			line = bytesTrimSuffix(line, '\n')
			line = bytesTrimSuffix(line, '\r')
			lineText := string(line)

			if lineText == "" {
				frame.Data = []byte(strings.Join(dataLines, "\n"))
				if frame.Event != "" || len(frame.Data) > 0 || len(frame.Comments) > 0 {
					return frame, nil
				}
				frameSize = 0
			} else if strings.HasPrefix(lineText, ":") {
				frame.Comments = append(frame.Comments, strings.TrimPrefix(lineText, ":"))
			} else {
				field, value, found := strings.Cut(lineText, ":")
				if found && strings.HasPrefix(value, " ") {
					value = value[1:]
				}
				switch field {
				case "event":
					frame.Event = value
				case "data":
					dataLines = append(dataLines, value)
				}
			}
			line = line[:0]
		}

		if err != nil {
			if err == io.EOF && (frame.Event != "" || len(dataLines) > 0 || len(frame.Comments) > 0) {
				frame.Data = []byte(strings.Join(dataLines, "\n"))
				return frame, nil
			}
			return codexImageSSEFrame{}, err
		}
	}
}

func bytesTrimSuffix(data []byte, suffix byte) []byte {
	if len(data) > 0 && data[len(data)-1] == suffix {
		return data[:len(data)-1]
	}
	return data
}

type codexImageStreamRead struct {
	frame codexImageSSEFrame
	err   error
}

func handleImageStreamResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(fmt.Errorf("invalid response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}

	ctx, cancel := context.WithCancel(c.Request.Context())
	reads := make(chan codexImageStreamRead)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		decoder := newCodexImageSSEDecoder(resp.Body)
		for {
			frame, err := decoder.Next()
			select {
			case reads <- codexImageStreamRead{frame: frame, err: err}:
			case <-ctx.Done():
				return
			}
			if err != nil {
				return
			}
		}
	}()
	defer func() {
		cancel()
		service.CloseResponseBodyGracefully(resp)
		<-readerDone
	}()

	info.StreamStatus = relaycommon.NewStreamStatus()
	usage := &dto.Usage{}
	var doneResults []imageCallResult
	completedImages := int64(0)
	realOutput := false

	pingInterval := time.Duration(constant.ImageStreamPingInterval) * time.Second
	var pingTimer imageStreamTimer
	var ping <-chan time.Time
	if pingInterval > 0 {
		pingTimer = newImageStreamTimer(pingInterval)
		ping = pingTimer.Chan()
		defer pingTimer.Stop()
	}
	resetPing := func() {
		if pingTimer != nil {
			pingTimer.Reset(pingInterval)
		}
	}

	writeEvent := func(event string, payload []byte, isRealOutput bool) error {
		helper.SetEventStreamHeaders(c)
		c.Set(constant.ContextKeyCodexImageStreamCommitted, true)
		if err := helper.ResponseChunkData(c, dto.ResponsesStreamResponse{Type: event}, string(payload)); err != nil {
			return err
		}
		if isRealOutput {
			realOutput = true
			c.Set(constant.ContextKeyCodexImageRealOutput, true)
			info.SetFirstResponseTime()
			info.ReceivedResponseCount++
		}
		resetPing()
		return nil
	}
	finishWithError := func(err error) (*dto.Usage, *types.NewAPIError) {
		if c.Request.Context().Err() != nil {
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, c.Request.Context().Err())
			return usage, nil
		}
		if !realOutput {
			return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusBadGateway)
		}
		info.StreamStatus.RecordError(err.Error())
		info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonHandlerStop, err)
		_ = writeCodexImageStreamError(c, "upstream image generation failed")
		_ = helper.StringData(c, "[DONE]")
		return usage, nil
	}

	for {
		select {
		case <-c.Request.Context().Done():
			info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, c.Request.Context().Err())
			return usage, nil
		case <-ping:
			helper.SetEventStreamHeaders(c)
			c.Set(constant.ContextKeyCodexImageStreamCommitted, true)
			if _, err := c.Writer.Write([]byte(":\n\n")); err != nil {
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, err)
				return usage, nil
			}
			if err := helper.FlushWriter(c); err != nil {
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, err)
				return usage, nil
			}
			resetPing()
		case read := <-reads:
			if read.err != nil {
				if read.err == io.EOF {
					return finishWithError(fmt.Errorf("codex image stream ended before response.completed"))
				}
				return finishWithError(read.err)
			}

			payload := read.frame.Data
			if len(payload) == 0 {
				continue
			}
			if strings.TrimSpace(string(payload)) == "[DONE]" {
				return finishWithError(fmt.Errorf("codex image stream ended before response.completed"))
			}

			eventType := gjson.GetBytes(payload, "type").String()
			if eventType == "" {
				eventType = read.frame.Event
			}
			switch eventType {
			case "response.image_generation_call.partial_image":
				partialB64 := strings.TrimSpace(gjson.GetBytes(payload, "partial_image_b64").String())
				if partialB64 == "" {
					continue
				}
				event := codexImageEventName(info, "partial_image")
				partial := map[string]any{
					"type":                event,
					"partial_image_index": gjson.GetBytes(payload, "partial_image_index").Int(),
				}
				if c.GetString(ginKeyCodexImageResponseFormat) == "url" {
					partial["url"] = "data:" + mimeTypeFromOutputFormat(gjson.GetBytes(payload, "output_format").String()) + ";base64," + partialB64
				} else {
					partial["b64_json"] = partialB64
				}
				out, err := common.Marshal(partial)
				if err != nil {
					return finishWithError(err)
				}
				if err := writeEvent(event, out, true); err != nil {
					info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, err)
					return usage, nil
				}
			case "response.output_item.done":
				item := gjson.GetBytes(payload, "item")
				if item.Get("type").String() == dto.ResponsesOutputTypeImageGenerationCall {
					result := imageCallResultFromGJSON(item)
					if result.Result != "" {
						doneResults = append(doneResults, result)
					}
				}
			case "response.completed":
				results, createdAt, completedUsage, _, apiErr := extractImagesFromCompletedJSON(payload)
				if apiErr != nil {
					return finishWithError(apiErr)
				}
				if len(results) == 0 {
					results = doneResults
				}
				if len(results) == 0 {
					return finishWithError(fmt.Errorf("upstream did not return image output"))
				}
				usage = completedUsage
				for _, result := range results {
					event := codexImageEventName(info, "completed")
					out, err := buildCodexImageStreamEvent(event, result, createdAt, usage, c.GetString(ginKeyCodexImageResponseFormat))
					if err != nil {
						return finishWithError(err)
					}
					if err := writeEvent(event, out, true); err != nil {
						info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, err)
						return usage, nil
					}
					completedImages++
				}
				updateCodexImageCount(info, completedImages)
				if err := helper.StringData(c, "[DONE]"); err != nil {
					info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, err)
					return usage, nil
				}
				info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonDone, nil)
				return usage, nil
			case "response.error", "response.failed":
				message := extractCodexErrorMessage(payload)
				if message == "" {
					message = "codex upstream returned an image stream error"
				}
				return finishWithError(fmt.Errorf("%s", truncateErrorMessage(message)))
			}
		}
	}

}

func codexImageEventName(info *relaycommon.RelayInfo, suffix string) string {
	prefix := "image_generation"
	if info != nil && info.RelayMode == relayconstant.RelayModeImagesEdits {
		prefix = "image_edit"
	}
	return prefix + "." + suffix
}

func buildCodexImageStreamEvent(event string, result imageCallResult, createdAt int64, usage *dto.Usage, responseFormat string) ([]byte, error) {
	payload := map[string]any{
		"type":       event,
		"created_at": createdAt,
	}
	if result.RevisedPrompt != "" {
		payload["revised_prompt"] = result.RevisedPrompt
	}
	if result.OutputFormat != "" {
		payload["output_format"] = result.OutputFormat
	}
	if result.Size != "" {
		payload["size"] = result.Size
	}
	if result.Background != "" {
		payload["background"] = result.Background
	}
	if result.Quality != "" {
		payload["quality"] = result.Quality
	}
	if usage != nil && usage.TotalTokens > 0 {
		payload["usage"] = usage
	}
	if strings.EqualFold(strings.TrimSpace(responseFormat), "url") {
		payload["url"] = "data:" + mimeTypeFromOutputFormat(result.OutputFormat) + ";base64," + result.Result
	} else {
		payload["b64_json"] = result.Result
	}
	return common.Marshal(payload)
}

func writeCodexImageStreamError(c *gin.Context, message string) error {
	payload, err := common.Marshal(map[string]any{
		"type": "error",
		"error": map[string]any{
			"message": message,
			"type":    "upstream_error",
			"code":    "image_stream_error",
		},
	})
	if err != nil {
		return err
	}
	return helper.ResponseChunkData(c, dto.ResponsesStreamResponse{Type: "error"}, string(payload))
}

func updateCodexImageCount(info *relaycommon.RelayInfo, count int64) {
	if info == nil || !info.PriceData.UsePrice || count <= 0 || count > int64(dto.MaxImageN) {
		return
	}
	info.PriceData.AddOtherRatio("n", float64(count))
}
