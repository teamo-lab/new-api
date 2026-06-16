package model

import "github.com/QuantumNous/new-api/common"

func UpdateChannelStatusDirect(channelId int, status int, reason string) (*Channel, error) {
	channel, err := GetChannelById(channelId, true)
	if err != nil {
		return nil, err
	}
	if channel.Status == status {
		return channel, nil
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

	if err := channel.SaveWithoutKey(); err != nil {
		return nil, err
	}
	if err := UpdateAbilityStatus(channelId, status == common.ChannelStatusEnabled); err != nil {
		return nil, err
	}
	InitChannelCache()
	return channel, nil
}
