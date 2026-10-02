package oaichat

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/relayconvert/convmeta"
	"github.com/stretchr/testify/require"
)

func TestOpenAIChatRequestToGeminiPreservesCachedContent(t *testing.T) {
	request := dto.GeneralOpenAIRequest{
		Model: "gemini-2.5-flash",
		Messages: []dto.Message{{Role: "user", Content: "continue"}},
		ExtraBody: json.RawMessage(`{"google":{"cached_content":"cachedContents/test"}}`),
	}

	converted, err := OpenAIChatRequestToGeminiGenerateContent(nil, request, &convmeta.Values{UpstreamModelName: request.Model})
	require.NoError(t, err)
	require.Equal(t, "cachedContents/test", converted.CachedContent)
}

