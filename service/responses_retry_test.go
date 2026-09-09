package service

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetResponsesCooldownCacheForTest() {
	responsesCooldownOnce = sync.Once{}
	responsesCooldownCache = nil
	responsesCooldownLocal = nil
}

func TestResponsesChannelCooldownUsesRedisAndFallsBackLocally(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	originalRDB := common.RDB
	originalCooldown := common.ResponsesChannelCooldownSeconds
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	resetResponsesCooldownCacheForTest()
	common.RedisEnabled = true
	common.RDB = client
	common.ResponsesChannelCooldownSeconds = 30
	t.Cleanup(func() {
		_ = client.Close()
		common.RedisEnabled = originalRedisEnabled
		common.RDB = originalRDB
		common.ResponsesChannelCooldownSeconds = originalCooldown
		resetResponsesCooldownCacheForTest()
	})

	MarkResponsesChannelCooldown(41, "GPT-5-Codex")

	keys := server.Keys()
	require.Len(t, keys, 1)
	assert.Contains(t, keys[0], "41:gpt-5-codex")
	assert.True(t, IsResponsesChannelCooling(41, "gpt-5-codex"))

	server.Close()
	assert.True(t, IsResponsesChannelCooling(41, "GPT-5-CODEX"), "local fallback must survive Redis failure")
}

func TestResponsesChannelCooldownLocalEntryExpires(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	originalCooldown := common.ResponsesChannelCooldownSeconds
	resetResponsesCooldownCacheForTest()
	common.RedisEnabled = false
	common.ResponsesChannelCooldownSeconds = 30
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		common.ResponsesChannelCooldownSeconds = originalCooldown
		resetResponsesCooldownCacheForTest()
	})

	cache, local := getResponsesCooldownCaches()
	key := responsesCooldownKey(42, "gpt-5-codex")
	local.SetWithTTL(cache.FullKey(key), 1, 20*time.Millisecond)
	require.True(t, IsResponsesChannelCooling(42, "gpt-5-codex"))

	require.Eventually(t, func() bool {
		return !IsResponsesChannelCooling(42, "gpt-5-codex")
	}, time.Second, 10*time.Millisecond)
}

func TestResponsesTransientFailureDoesNotPermanentlyDisableChannel(t *testing.T) {
	originalEnabled := common.AutomaticDisableChannelEnabled
	originalRanges := operation_setting.AutomaticDisableStatusCodesToString()
	common.AutomaticDisableChannelEnabled = true
	require.NoError(t, operation_setting.AutomaticDisableStatusCodesFromString("503"))
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled = originalEnabled
		require.NoError(t, operation_setting.AutomaticDisableStatusCodesFromString(originalRanges))
	})

	transient := types.NewOpenAIError(
		errors.New("Selected model is at capacity"),
		types.ErrorCodeBadResponseBody,
		http.StatusServiceUnavailable,
		types.ErrOptionWithSameChannelRetry(),
	)

	assert.False(t, ShouldDisableChannel(transient))
}

func TestResponsesSameChannelRetryCountAndBackoff(t *testing.T) {
	originalRetryTimes := common.SameChannelRetryTimes
	common.SameChannelRetryTimes = 5
	t.Cleanup(func() { common.SameChannelRetryTimes = originalRetryTimes })
	transient := types.NewOpenAIError(
		errors.New("capacity"),
		types.ErrorCodeBadResponseBody,
		http.StatusServiceUnavailable,
		types.ErrOptionWithSameChannelRetry(),
	)

	for attempt := 1; attempt <= 5; attempt++ {
		assert.True(t, CanRetryResponsesOnSameChannel(transient, attempt))
	}
	assert.False(t, CanRetryResponsesOnSameChannel(transient, 6))
	assert.Equal(t, []time.Duration{
		500 * time.Millisecond,
		time.Second,
		2 * time.Second,
		4 * time.Second,
		8 * time.Second,
	}, []time.Duration{
		responsesRetryDelay(1, 0),
		responsesRetryDelay(2, 0),
		responsesRetryDelay(3, 0),
		responsesRetryDelay(4, 0),
		responsesRetryDelay(5, 0),
	})
	assert.Equal(t, 3*time.Second, responsesRetryDelay(5, 3*time.Second))
}
