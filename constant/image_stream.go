package constant

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const DefaultImageStreamPingInterval = 10

var ImageStreamPingInterval = DefaultImageStreamPingInterval

func InitImageStreamPingInterval() error {
	raw := strings.TrimSpace(os.Getenv("IMAGE_STREAM_PING_INTERVAL"))
	if raw == "" {
		ImageStreamPingInterval = DefaultImageStreamPingInterval
		return nil
	}

	seconds, err := strconv.Atoi(raw)
	if err != nil || (seconds != 0 && (seconds < 5 || seconds > 60)) {
		return fmt.Errorf("IMAGE_STREAM_PING_INTERVAL must be 0 or an integer between 5 and 60 seconds")
	}
	ImageStreamPingInterval = seconds
	return nil
}
