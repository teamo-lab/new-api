package model

import "github.com/QuantumNous/new-api/common"

// UpdateChannelStatusDirect sets a channel's status for the Teamo channel
// status action APIs. Unlike UpdateChannelStatus it also re-enables a
// channel and does not track per-key multi-key state. The status reason is
// written through UpdateChannelOtherInfo so a concurrent other_info writer
// is not overwritten.
func UpdateChannelStatusDirect(channelId int, status int, reason string) (*Channel, error) {
	var updated Channel
	written, err := UpdateChannelOtherInfo(channelId, func(channel *Channel) (map[string]any, error) {
		updated = *channel
		if channel.Status == status {
			return nil, nil
		}
		info := channel.GetOtherInfo()
		if status == common.ChannelStatusEnabled {
			delete(info, "status_reason")
			delete(info, "status_time")
		} else {
			info["status_reason"] = reason
			info["status_time"] = common.GetTimestamp()
		}
		channel.SetOtherInfo(info)
		channel.Status = status
		updated = *channel
		return map[string]any{"status": status}, nil
	})
	if err != nil {
		return nil, err
	}
	if !written {
		return &updated, nil
	}
	if err := UpdateAbilityStatus(channelId, status == common.ChannelStatusEnabled); err != nil {
		return nil, err
	}
	BumpSyncTopic(SyncTopicChannels)
	InitChannelCache()
	return &updated, nil
}
