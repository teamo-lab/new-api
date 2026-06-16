package controller

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type ChannelStatusActionRequest struct {
	Reason string `json:"reason"`
}

type ChannelStatusActionResponse struct {
	Id     int    `json:"id"`
	Status int    `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func parseChannelId(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "渠道 ID 无效")
		return 0, false
	}
	return id, true
}

func bindChannelStatusActionRequest(c *gin.Context) (ChannelStatusActionRequest, bool) {
	req := ChannelStatusActionRequest{}
	if c.Request != nil && c.Request.Body != nil && c.Request.ContentLength != 0 {
		if err := common.DecodeJson(c.Request.Body, &req); err != nil {
			common.ApiError(c, err)
			return ChannelStatusActionRequest{}, false
		}
	}
	req.Reason = strings.TrimSpace(req.Reason)
	return req, true
}

func updateChannelStatusAction(c *gin.Context, status int, defaultReason string) {
	id, ok := parseChannelId(c)
	if !ok {
		return
	}

	req, ok := bindChannelStatusActionRequest(c)
	if !ok {
		return
	}
	reason := req.Reason
	if reason == "" {
		reason = defaultReason
	}

	channel, err := model.UpdateChannelStatusDirect(id, status, reason)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, ChannelStatusActionResponse{
		Id:     channel.Id,
		Status: channel.Status,
		Reason: reason,
	})
}

func AutoDisableChannel(c *gin.Context) {
	updateChannelStatusAction(c, common.ChannelStatusAutoDisabled, "manual auto-disable via API")
}

func EnableChannel(c *gin.Context) {
	updateChannelStatusAction(c, common.ChannelStatusEnabled, "")
}
