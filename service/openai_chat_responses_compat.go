package service

import (
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service/relayconvert"
)

func ChatCompletionsRequestToResponsesRequest(req *dto.GeneralOpenAIRequest) (*dto.OpenAIResponsesRequest, error) {
	return relayconvert.ChatCompletionsRequestToResponsesRequest(req)
}

func ResponsesRequestToChatCompletionsRequest(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, error) {
	return relayconvert.ResponsesRequestToChatCompletionsRequest(req)
}

func ResponsesRequestToChatCompletionsRequestWithToolContext(req *dto.OpenAIResponsesRequest) (*dto.GeneralOpenAIRequest, *relayconvert.ResponsesToChatToolContext, error) {
	return relayconvert.ResponsesRequestToChatCompletionsRequestWithToolContext(req)
}

func ChatCompletionsResponseToResponsesResponse(resp *dto.OpenAITextResponse, id string) (*dto.OpenAIResponsesResponse, *dto.Usage, error) {
	return relayconvert.ChatCompletionsResponseToResponsesResponse(resp, id)
}

func ChatCompletionsResponseToResponsesResponseWithToolContext(resp *dto.OpenAITextResponse, id string, toolContext *relayconvert.ResponsesToChatToolContext) (*dto.OpenAIResponsesResponse, *dto.Usage, error) {
	return relayconvert.ChatCompletionsResponseToResponsesResponseWithToolContext(resp, id, toolContext)
}

func ResponsesResponseToChatCompletionsResponse(resp *dto.OpenAIResponsesResponse, id string) (*dto.OpenAITextResponse, *dto.Usage, error) {
	return relayconvert.ResponsesResponseToChatCompletionsResponse(resp, id)
}

func ResponsesFinishReasonFromStatus(resp *dto.OpenAIResponsesResponse) (string, bool) {
	return relayconvert.ResponsesFinishReasonFromStatus(resp)
}

func ExtractOutputTextFromResponses(resp *dto.OpenAIResponsesResponse) string {
	return relayconvert.ExtractOutputTextFromResponses(resp)
}
