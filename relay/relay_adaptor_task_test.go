package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTaskAdaptorTaitanVideo(t *testing.T) {
	platform := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeTaitanVideo))
	adaptor := GetTaskAdaptor(platform)

	require.NotNil(t, adaptor)
	assert.Equal(t, "taitan-video", adaptor.GetChannelName())
}
