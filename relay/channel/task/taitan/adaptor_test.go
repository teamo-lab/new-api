package taitan

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRequestAndSetAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		body       string
		wantCode   string
		wantStatus int
	}{
		{
			name: "valid request",
			body: `{"model":"seedance-2.0","prompt":"电影感镜头，一位角色在雨夜霓虹街道中奔跑。","duration":8,"metadata":{"aspect_ratio":"16:9"}}`,
		},
		{
			name:       "unsupported duration",
			body:       `{"model":"seedance-2.0","prompt":"电影感镜头，一位角色在雨夜霓虹街道中奔跑。","duration":7}`,
			wantCode:   "invalid_duration",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid seconds string",
			body:       `{"model":"seedance-2.0","prompt":"电影感镜头，一位角色在雨夜霓虹街道中奔跑。","seconds":"invalid"}`,
			wantCode:   "invalid_duration",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "short prompt",
			body:       `{"model":"seedance-2.0","prompt":"太短了","duration":5}`,
			wantCode:   "invalid_prompt",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported aspect ratio",
			body:       `{"model":"seedance-2.0","prompt":"电影感镜头，一位角色在雨夜霓虹街道中奔跑。","size":"1:1"}`,
			wantCode:   "invalid_aspect_ratio",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "url reference is rejected",
			body:       `{"model":"seedance-2.0","prompt":"电影感镜头，一位角色在雨夜霓虹街道中奔跑。","images":["https://example.com/ref.png"]}`,
			wantCode:   "unsupported_reference",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			info := &relaycommon.RelayInfo{TaskRelayInfo: &relaycommon.TaskRelayInfo{}}
			adaptor := &TaskAdaptor{}

			taskErr := adaptor.ValidateRequestAndSetAction(c, info)
			if tt.wantCode != "" {
				require.NotNil(t, taskErr)
				assert.Equal(t, tt.wantCode, taskErr.Code)
				assert.Equal(t, tt.wantStatus, taskErr.StatusCode)
				return
			}

			require.Nil(t, taskErr)
			assert.Equal(t, constant.TaskActionGenerate, info.Action)
			assert.Equal(t, map[string]float64{"seconds": 8}, adaptor.EstimateBilling(c, info))
		})
	}
}

func TestBuildRequestBodyForwardsMultipartReferences(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var input bytes.Buffer
	inputWriter := multipart.NewWriter(&input)
	require.NoError(t, inputWriter.WriteField("model", "seedance-2.0"))
	require.NoError(t, inputWriter.WriteField("prompt", "电影感镜头，一位角色在雨夜霓虹街道中奔跑。"))
	require.NoError(t, inputWriter.WriteField("duration", "15"))
	require.NoError(t, inputWriter.WriteField("aspect_ratio", "16:9"))
	referencePart, err := inputWriter.CreateFormFile("references", "role.png")
	require.NoError(t, err)
	_, err = referencePart.Write([]byte("png-data"))
	require.NoError(t, err)
	require.NoError(t, inputWriter.Close())

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", bytes.NewReader(input.Bytes()))
	c.Request.Header.Set("Content-Type", inputWriter.FormDataContentType())
	storage, err := common.CreateBodyStorage(input.Bytes())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })
	c.Set(common.KeyBodyStorage, storage)
	c.Request.Body = io.NopCloser(storage)
	info := &relaycommon.RelayInfo{
		OriginModelName: "seedance-2.0",
		ChannelMeta:     &relaycommon.ChannelMeta{UpstreamModelName: "upstream-seedance"},
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{},
	}
	adaptor := &TaskAdaptor{}
	require.Nil(t, adaptor.ValidateRequestAndSetAction(c, info))

	requestBody, err := adaptor.BuildRequestBody(c, info)
	require.NoError(t, err)
	body, err := io.ReadAll(requestBody)
	require.NoError(t, err)
	mediaType, params, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	require.NoError(t, err)
	require.Equal(t, "multipart/form-data", mediaType)
	reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])

	fields := make(map[string]string)
	var referenceName string
	var referenceBody string
	for {
		part, readErr := reader.NextPart()
		if readErr == io.EOF {
			break
		}
		require.NoError(t, readErr)
		value, readErr := io.ReadAll(part)
		require.NoError(t, readErr)
		if part.FileName() == "" {
			fields[part.FormName()] = string(value)
			continue
		}
		referenceName = part.FileName()
		referenceBody = string(value)
		assert.Equal(t, "references", part.FormName())
	}

	assert.Equal(t, "电影感镜头，一位角色在雨夜霓虹街道中奔跑。", fields["prompt"])
	assert.Equal(t, "upstream-seedance", fields["model"])
	assert.Equal(t, "15", fields["duration"])
	assert.Equal(t, "16:9", fields["aspect_ratio"])
	assert.Equal(t, "role.png", referenceName)
	assert.Equal(t, "png-data", referenceBody)
}

func TestParseTaskResult(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus model.TaskStatus
		wantURL    string
		wantReason string
	}{
		{name: "queued", body: `{"status":"queued"}`, wantStatus: model.TaskStatusQueued},
		{name: "processing", body: `{"status":"processing"}`, wantStatus: model.TaskStatusInProgress},
		{name: "succeeded", body: `{"status":"succeeded","video_url":"https://cdn.example/video.mp4"}`, wantStatus: model.TaskStatusSuccess, wantURL: "https://cdn.example/video.mp4"},
		{name: "failed", body: `{"status":"failed","error":{"code":"generation_failed","message":"upstream failed"}}`, wantStatus: model.TaskStatusFailure, wantReason: "upstream failed"},
	}

	adaptor := &TaskAdaptor{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := adaptor.ParseTaskResult([]byte(tt.body))
			require.NoError(t, err)
			assert.Equal(t, tt.wantStatus, model.TaskStatus(result.Status))
			assert.Equal(t, tt.wantURL, result.Url)
			assert.Equal(t, tt.wantReason, result.Reason)
		})
	}
}

func TestDoResponseHidesUpstreamTaskID(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	response := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"id":"upstream-task","status":"queued"}`)),
	}
	info := &relaycommon.RelayInfo{
		TaskRelayInfo:   &relaycommon.TaskRelayInfo{PublicTaskID: "task_public"},
		OriginModelName: "seedance-2.0",
	}

	upstreamID, taskData, taskErr := (&TaskAdaptor{}).DoResponse(c, response, info)
	require.Nil(t, taskErr)
	assert.Equal(t, "upstream-task", upstreamID)
	assert.Contains(t, string(taskData), "upstream-task")
	assert.NotContains(t, recorder.Body.String(), "upstream-task")
	assert.Contains(t, recorder.Body.String(), "task_public")
}

func TestFetchTaskUsesTaitanEndpoint(t *testing.T) {
	service.InitHttpClient()
	var receivedPath string
	var receivedAuthorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		receivedAuthorization = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"queued"}`))
	}))
	t.Cleanup(server.Close)

	resp, err := (&TaskAdaptor{}).FetchTask(server.URL, "secret", map[string]any{"task_id": "task_123"}, "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NoError(t, resp.Body.Close())
	assert.Equal(t, SubmitEndpoint+"/task_123", receivedPath)
	assert.Equal(t, "Bearer secret", receivedAuthorization)
}

func TestFetchTaskResolvesQueuedIDFromTaskList(t *testing.T) {
	service.InitHttpClient()
	const queuedID = "queued-1785896646729-72e05916-e9af-44c2-ba44-6429bd109486"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.RawQuery == "limit=100" {
			_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"2084829681764495361","task_id":"2084829681764495361","status":"succeeded","video_url":"/media/result.mp4","cover_url":"/media/cover.jpg","created_at":"2026-08-05T02:24:06.896Z"}]}`))
			return
		}
		assert.Equal(t, SubmitEndpoint+"/"+queuedID, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"task_not_found"}}`))
	}))
	t.Cleanup(server.Close)

	resp, err := (&TaskAdaptor{}).FetchTask(server.URL, "secret", map[string]any{"task_id": queuedID}, "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	t.Cleanup(func() { require.NoError(t, resp.Body.Close()) })
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	responseBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var result responsePayload
	require.NoError(t, common.Unmarshal(responseBody, &result))
	assert.Equal(t, "2084829681764495361", result.TaskID)
	assert.Equal(t, "succeeded", result.Status)
	assert.Equal(t, server.URL+"/media/result.mp4", result.VideoURL)
	assert.Equal(t, server.URL+"/media/cover.jpg", result.CoverURL)
}

func TestConvertToOpenAIVideoIncludesResultMetadata(t *testing.T) {
	task := &model.Task{
		TaskID:    "task_public",
		Status:    model.TaskStatusSuccess,
		Progress:  "100%",
		CreatedAt: 100,
		UpdatedAt: 200,
		Properties: model.Properties{
			OriginModelName: "seedance-2.0",
		},
		Data: []byte(`{"status":"succeeded","video_url":"https://cdn.example/video.mp4","cover_url":"https://cdn.example/cover.jpg"}`),
	}

	data, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	var video dto.OpenAIVideo
	require.NoError(t, common.Unmarshal(data, &video))
	assert.Equal(t, dto.VideoStatusCompleted, video.Status)
	assert.Equal(t, "https://cdn.example/video.mp4", video.Metadata["url"])
	assert.Equal(t, "https://cdn.example/cover.jpg", video.Metadata["cover_url"])
}
