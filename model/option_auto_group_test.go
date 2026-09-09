package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOptionValueRejectsInvalidMaxTokenAutoGroups(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "1.5", "invalid"} {
		t.Run(value, func(t *testing.T) {
			assert.Error(t, validateOptionValue("MaxTokenAutoGroups", value))
		})
	}
	require.NoError(t, validateOptionValue("MaxTokenAutoGroups", "999999"))
}

func TestValidateOptionValueRejectsInvalidResponsesRetrySettings(t *testing.T) {
	tests := []struct {
		key     string
		valid   []string
		invalid []string
	}{
		{key: "SameChannelRetryTimes", valid: []string{"0", "5", "10"}, invalid: []string{"-1", "11", "1.5", "invalid"}},
		{key: "ResponsesRetryMaxDurationSeconds", valid: []string{"1", "60", "600"}, invalid: []string{"0", "601", "1.5", "invalid"}},
		{key: "ResponsesChannelCooldownSeconds", valid: []string{"0", "30", "3600"}, invalid: []string{"-1", "3601", "1.5", "invalid"}},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			for _, value := range tt.valid {
				require.NoError(t, validateOptionValue(tt.key, value))
			}
			for _, value := range tt.invalid {
				assert.Error(t, validateOptionValue(tt.key, value))
			}
		})
	}
}
