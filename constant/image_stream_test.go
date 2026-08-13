package constant

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitImageStreamPingInterval(t *testing.T) {
	original := ImageStreamPingInterval
	t.Cleanup(func() { ImageStreamPingInterval = original })

	for _, tt := range []struct {
		name    string
		value   string
		want    int
		wantErr bool
	}{
		{name: "absent defaults", want: DefaultImageStreamPingInterval},
		{name: "zero disables", value: "0", want: 0},
		{name: "minimum", value: "5", want: 5},
		{name: "maximum", value: "60", want: 60},
		{name: "below minimum", value: "4", wantErr: true},
		{name: "above maximum", value: "61", wantErr: true},
		{name: "not an integer", value: "fast", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("IMAGE_STREAM_PING_INTERVAL", tt.value)
			err := InitImageStreamPingInterval()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, ImageStreamPingInterval)
		})
	}
}
