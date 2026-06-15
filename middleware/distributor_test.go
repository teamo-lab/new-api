package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newProbeBypassContext(value string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	if value != "" {
		c.Request.Header.Set(channelProbeBypassHeader, value)
	}
	return c
}

func TestAllowChannelProbeBypassRequiresAutoDisabledChannelAndHeader(t *testing.T) {
	require.True(t, allowChannelProbeBypass(
		newProbeBypassContext("1"),
		&model.Channel{Status: common.ChannelStatusAutoDisabled},
	))
	require.False(t, allowChannelProbeBypass(
		newProbeBypassContext(""),
		&model.Channel{Status: common.ChannelStatusAutoDisabled},
	))
	require.False(t, allowChannelProbeBypass(
		newProbeBypassContext("0"),
		&model.Channel{Status: common.ChannelStatusAutoDisabled},
	))
	require.False(t, allowChannelProbeBypass(
		newProbeBypassContext("1"),
		&model.Channel{Status: common.ChannelStatusManuallyDisabled},
	))
	require.False(t, allowChannelProbeBypass(newProbeBypassContext("1"), &model.Channel{Status: common.ChannelStatusEnabled}))
}
