package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newProbeBypassContext(value string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	if value != "" {
		c.Request.Header.Set(ChannelProbeBypassHeader, value)
	}
	return c
}

func TestAllowChannelProbeBypass(t *testing.T) {
	tokenPin := dto.ChannelPin{ChannelId: 1, Source: dto.PinSourceToken}
	taskPin := dto.ChannelPin{ChannelId: 1, Source: dto.PinSourceOriginTask}
	autoDisabled := &model.Channel{Status: common.ChannelStatusAutoDisabled}
	manuallyDisabled := &model.Channel{Status: common.ChannelStatusManuallyDisabled}

	cases := []struct {
		name    string
		header  string
		pin     dto.ChannelPin
		channel *model.Channel
		want    bool
	}{
		{"token pin, auto-disabled, header 1", "1", tokenPin, autoDisabled, true},
		{"no header", "", tokenPin, autoDisabled, false},
		{"header 0", "0", tokenPin, autoDisabled, false},
		{"manually disabled stays blocked", "1", tokenPin, manuallyDisabled, false},
		{"origin-task pin never bypasses", "1", taskPin, autoDisabled, false},
		{"nil channel", "1", tokenPin, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, allowChannelProbeBypass(newProbeBypassContext(tc.header), tc.pin, tc.channel))
		})
	}
}
