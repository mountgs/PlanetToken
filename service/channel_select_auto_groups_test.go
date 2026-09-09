package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelSelectAutoGroupsTest(t *testing.T) *gorm.DB {
	t.Helper()

	originalDB := model.DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalRetryTimes := common.RetryTimes
	originalAutoGroups := setting.AutoGroups2JsonString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalGroupRatios := ratio_setting.GroupRatio2JSONString()
	originalMaxTokenAutoGroups := setting.GetMaxTokenAutoGroups()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}))
	model.DB = db
	common.MemoryCacheEnabled = true
	common.RetryTimes = 0

	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`[]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":2}`))
	require.NoError(t, setting.UpdateMaxTokenAutoGroups("2"))

	t.Cleanup(func() {
		model.DB = originalDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.RetryTimes = originalRetryTimes
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAutoGroups))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatios))
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(fmt.Sprintf("%d", originalMaxTokenAutoGroups)))

		if originalMemoryCacheEnabled && originalDB != nil &&
			originalDB.Migrator().HasTable(&model.Channel{}) && originalDB.Migrator().HasTable(&model.Ability{}) {
			model.InitChannelCache()
		}
		sqlDB, err := db.DB()
		if err == nil {
			require.NoError(t, sqlDB.Close())
		}
	})

	return db
}

func createChannelSelectAutoGroupsChannel(t *testing.T, db *gorm.DB, id int, group, modelName string) {
	t.Helper()
	priority := int64(0)
	weight := uint(100)
	require.NoError(t, db.Create(&model.Channel{
		Id:       id,
		Type:     constant.ChannelTypeOpenAI,
		Key:      fmt.Sprintf("key-%d", id),
		Status:   common.ChannelStatusEnabled,
		Name:     fmt.Sprintf("channel-%d", id),
		Weight:   &weight,
		Models:   modelName,
		Group:    group,
		Priority: &priority,
	}).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     group,
		Model:     modelName,
		ChannelId: id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
}

func TestCacheGetRandomSatisfiedChannelUsesTokenAutoGroupsWhenGlobalAutoIsEmpty(t *testing.T) {
	db := setupChannelSelectAutoGroupsTest(t)
	const modelName = "auto-groups-runtime-model"
	createChannelSelectAutoGroupsChannel(t, db, 2101, "vip", modelName)
	createChannelSelectAutoGroupsChannel(t, db, 2102, "default", modelName)
	model.InitChannelCache()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenAutoGroups, []string{"vip", "default"})
	common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, true)

	retry := 0
	param := &RetryParam{
		Ctx:         ctx,
		TokenGroup:  "auto",
		ModelName:   modelName,
		RequestPath: "/v1/chat/completions",
		Retry:       &retry,
	}

	first, selectedGroup, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, 2101, first.Id)
	assert.Equal(t, "vip", selectedGroup)
	assert.Equal(t, "vip", common.GetContextKeyString(ctx, constant.ContextKeyAutoGroup))
	assert.Empty(t, setting.GetAutoGroups(), "the selection must not depend on the global Auto list")

	param.IncreaseRetry()
	second, selectedGroup, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, 2102, second.Id)
	assert.Equal(t, "default", selectedGroup)
	assert.Equal(t, "default", common.GetContextKeyString(ctx, constant.ContextKeyAutoGroup))
}

func TestCacheGetRandomSatisfiedChannelExcludesFailedResponsesChannel(t *testing.T) {
	db := setupChannelSelectAutoGroupsTest(t)
	const modelName = "responses-retry-model"
	createChannelSelectAutoGroupsChannel(t, db, 2201, "default", modelName)
	createChannelSelectAutoGroupsChannel(t, db, 2202, "default", modelName)
	model.InitChannelCache()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	param := &RetryParam{
		Ctx:                ctx,
		TokenGroup:         "default",
		ModelName:          modelName,
		RequestPath:        "/v1/responses",
		Retry:              common.GetPointer(0),
		ExcludedChannelIDs: map[int]struct{}{2201: {}},
	}

	selected, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, 2202, selected.Id)
}

func TestCacheGetRandomSatisfiedChannelSelectsHighestRemainingResponsesPriority(t *testing.T) {
	db := setupChannelSelectAutoGroupsTest(t)
	const modelName = "responses-priority-retry-model"
	createChannelSelectAutoGroupsChannel(t, db, 2211, "default", modelName)
	createChannelSelectAutoGroupsChannel(t, db, 2212, "default", modelName)
	createChannelSelectAutoGroupsChannel(t, db, 2213, "default", modelName)
	for channelID, priority := range map[int]int64{2211: 100, 2212: 90, 2213: 0} {
		require.NoError(t, db.Model(&model.Channel{}).Where("id = ?", channelID).Update("priority", priority).Error)
		require.NoError(t, db.Model(&model.Ability{}).Where("channel_id = ?", channelID).Update("priority", priority).Error)
	}
	model.InitChannelCache()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	param := &RetryParam{
		Ctx:                ctx,
		TokenGroup:         "default",
		ModelName:          modelName,
		RequestPath:        "/v1/responses",
		Retry:              common.GetPointer(1),
		ExcludedChannelIDs: map[int]struct{}{2211: {}},
	}

	selected, _, err := CacheGetRandomSatisfiedChannel(param)
	require.NoError(t, err)
	require.NotNil(t, selected)
	assert.Equal(t, 2212, selected.Id)
}

func TestResponsesRetryClassifiesTransientAndDeterministicErrors(t *testing.T) {
	transient := types.NewOpenAIError(errors.New("overloaded"), types.ErrorCodeBadResponseStatusCode, http.StatusServiceUnavailable)
	auth := types.NewOpenAIError(errors.New("auth"), types.ErrorCodeBadResponseStatusCode, http.StatusUnauthorized)
	credentialFailure := types.NewOpenAIError(
		errors.New("upstream credential disabled"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusBadGateway,
		types.ErrOptionWithNextChannelRetry(),
	)
	deterministic := types.NewOpenAIError(errors.New("invalid"), types.ErrorCodeBadRequestBody, http.StatusBadRequest, types.ErrOptionWithSkipRetry())

	assert.True(t, ShouldRetryResponsesOnSameChannel(transient))
	assert.True(t, ShouldRetryResponsesOnNextChannel(transient))
	assert.False(t, ShouldRetryResponsesOnSameChannel(auth))
	assert.True(t, ShouldRetryResponsesOnNextChannel(auth))
	assert.False(t, ShouldRetryResponsesOnSameChannel(credentialFailure))
	assert.True(t, ShouldRetryResponsesOnNextChannel(credentialFailure))
	assert.False(t, ShouldRetryResponsesOnSameChannel(deterministic))
	assert.False(t, ShouldRetryResponsesOnNextChannel(deterministic))
}

func TestResponsesRetryWaitStopsWhenRetryAfterExceedsBudget(t *testing.T) {
	state := &ResponsesRetryState{deadline: time.Now().Add(20 * time.Millisecond)}
	started := time.Now()

	retried := state.Wait(context.Background(), 1, time.Second)

	assert.False(t, retried)
	assert.Less(t, time.Since(started), 100*time.Millisecond)
}
