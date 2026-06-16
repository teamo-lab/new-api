package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestUpdateChannelStatusDirectTogglesChannelAndAbilities(t *testing.T) {
	clearPreferredOwnerTables(t)

	require.NoError(t, DB.Create(&Channel{
		Id:     101,
		Key:    "sk-test",
		Status: common.ChannelStatusEnabled,
		Name:   "status-direct",
		Models: "gpt-test",
		Group:  "default",
	}).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     "default",
		Model:     "gpt-test",
		ChannelId: 101,
		Enabled:   true,
	}).Error)

	channel, err := UpdateChannelStatusDirect(101, common.ChannelStatusAutoDisabled, "probe failed")
	require.NoError(t, err)
	require.Equal(t, common.ChannelStatusAutoDisabled, channel.Status)
	require.Equal(t, "probe failed", channel.GetOtherInfo()["status_reason"])

	var ability Ability
	require.NoError(t, DB.Where("channel_id = ?", 101).First(&ability).Error)
	require.False(t, ability.Enabled)

	channel, err = UpdateChannelStatusDirect(101, common.ChannelStatusEnabled, "")
	require.NoError(t, err)
	require.Equal(t, common.ChannelStatusEnabled, channel.Status)
	require.NotContains(t, channel.GetOtherInfo(), "status_reason")

	require.NoError(t, DB.Where("channel_id = ?", 101).First(&ability).Error)
	require.True(t, ability.Enabled)
}
