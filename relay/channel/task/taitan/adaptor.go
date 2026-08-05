package taitan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type responsePayload struct {
	Object   string          `json:"object"`
	ID       string          `json:"id"`
	TaskID   string          `json:"task_id"`
	Model    string          `json:"model"`
	Status   string          `json:"status"`
	VideoURL string          `json:"video_url"`
	CoverURL string          `json:"cover_url"`
	Error    json.RawMessage `json:"error"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	apiKey  string
	baseURL string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.apiKey = info.ApiKey
	a.baseURL = strings.TrimRight(info.ChannelBaseUrl, "/")
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if taskErr := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); taskErr != nil {
		return taskErr
	}

	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Model) == "" {
		return service.TaskErrorWrapperLocal(errors.New("model field is required"), "missing_model", http.StatusBadRequest)
	}
	if utf8.RuneCountInString(strings.TrimSpace(req.Prompt)) < 10 {
		return service.TaskErrorWrapperLocal(errors.New("prompt must be at least 10 characters"), "invalid_prompt", http.StatusBadRequest)
	}

	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		form, formErr := common.ParseMultipartFormReusable(c)
		if formErr != nil {
			return service.TaskErrorWrapperLocal(formErr, "invalid_multipart_form", http.StatusBadRequest)
		}
		if durationText := firstFormValue(form, "duration", "duration_sec", "seconds"); durationText != "" {
			duration, parseErr := strconv.Atoi(durationText)
			if parseErr != nil {
				return service.TaskErrorWrapperLocal(errors.New("duration must be an integer"), "invalid_duration", http.StatusBadRequest)
			}
			req.Duration = duration
		}
		if req.Metadata == nil {
			req.Metadata = make(map[string]interface{})
		}
		if aspectRatio := firstFormValue(form, "aspect_ratio"); aspectRatio != "" {
			req.Metadata["aspect_ratio"] = aspectRatio
		}
		if referenceErr := validateReferenceFiles(form); referenceErr != nil {
			return service.TaskErrorWrapperLocal(referenceErr, "invalid_reference", http.StatusBadRequest)
		}
		c.Set("task_request", req)
	}

	duration := requestDuration(req)
	if !supportedDurations[duration] {
		return service.TaskErrorWrapperLocal(
			errors.New("duration must be one of 5, 8, 10, 12, or 15 seconds"),
			"invalid_duration",
			http.StatusBadRequest,
		)
	}
	aspectRatio := requestAspectRatio(req)
	if !supportedAspectRatios[aspectRatio] {
		return service.TaskErrorWrapperLocal(
			errors.New("aspect_ratio must be 9:16 or 16:9"),
			"invalid_aspect_ratio",
			http.StatusBadRequest,
		)
	}
	if len(req.Images) > 0 {
		return service.TaskErrorWrapperLocal(
			errors.New("URL references are not supported; upload reference files with multipart field references"),
			"unsupported_reference",
			http.StatusBadRequest,
		)
	}
	if req.Seconds != "" {
		if _, err := strconv.Atoi(req.Seconds); err != nil {
			return service.TaskErrorWrapperLocal(errors.New("seconds must be an integer"), "invalid_duration", http.StatusBadRequest)
		}
	}

	return nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	return map[string]float64{"seconds": float64(requestDuration(req))}
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return a.baseURL + SubmitEndpoint, nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", c.GetHeader("Content-Type"))
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fields := map[string]string{
		"prompt":       req.Prompt,
		"model":        info.UpstreamModelName,
		"duration":     strconv.Itoa(requestDuration(req)),
		"aspect_ratio": requestAspectRatio(req),
	}
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, errors.Wrapf(err, "write multipart field %s", name)
		}
	}

	if strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data") {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return nil, errors.Wrap(err, "parse multipart form")
		}
		if err := copyReferenceFiles(writer, form); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, errors.Wrap(err, "close multipart writer")
	}
	c.Request.Header.Set("Content-Type", writer.FormDataContentType())
	return bytes.NewReader(body.Bytes()), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var upstream responsePayload
	if err := common.Unmarshal(responseBody, &upstream); err != nil {
		return "", nil, service.TaskErrorWrapper(
			errors.Wrapf(err, "body: %s", responseBody),
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}
	upstreamTaskID := upstream.ID
	if upstreamTaskID == "" {
		upstreamTaskID = upstream.TaskID
	}
	if upstreamTaskID == "" {
		return "", nil, service.TaskErrorWrapper(errors.New("task_id is empty"), "invalid_response", http.StatusInternalServerError)
	}

	video := dto.NewOpenAIVideo()
	video.ID = info.PublicTaskID
	video.TaskID = info.PublicTaskID
	video.Model = info.OriginModelName
	video.CreatedAt = time.Now().Unix()
	if status := toVideoStatus(upstream.Status); status != dto.VideoStatusUnknown {
		video.Status = status
	}
	if upstream.VideoURL != "" {
		video.SetMetadata("url", upstream.VideoURL)
	}
	c.JSON(http.StatusOK, video)
	return upstreamTaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, errors.New("invalid task_id")
	}
	requestURL := strings.TrimRight(baseURL, "/") + SubmitEndpoint + "/" + url.PathEscape(taskID)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var upstream responsePayload
	if err := common.Unmarshal(respBody, &upstream); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result")
	}

	result := &relaycommon.TaskInfo{Code: 0}
	switch strings.ToLower(strings.TrimSpace(upstream.Status)) {
	case "created", "submitted", "queued", "pending":
		result.Status = model.TaskStatusQueued
		result.Progress = taskcommon.ProgressQueued
	case "processing", "running", "in_progress":
		result.Status = model.TaskStatusInProgress
		result.Progress = taskcommon.ProgressInProgress
	case "succeeded", "success", "completed":
		result.Status = model.TaskStatusSuccess
		result.Progress = taskcommon.ProgressComplete
		result.Url = upstream.VideoURL
	case "failed", "error", "cancelled", "canceled":
		result.Status = model.TaskStatusFailure
		result.Progress = taskcommon.ProgressComplete
		result.Reason, _ = responseError(upstream.Error)
		if result.Reason == "" {
			result.Reason = "task failed"
		}
	default:
		return nil, fmt.Errorf("unknown taitan task status %q", upstream.Status)
	}
	return result, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var upstream responsePayload
	if len(task.Data) > 0 {
		if err := common.Unmarshal(task.Data, &upstream); err != nil {
			return nil, errors.Wrap(err, "unmarshal taitan task data")
		}
	}

	video := task.ToOpenAIVideo()
	video.TaskID = task.TaskID
	if upstream.VideoURL != "" {
		video.SetMetadata("url", upstream.VideoURL)
	}
	if upstream.CoverURL != "" {
		video.SetMetadata("cover_url", upstream.CoverURL)
	}
	if task.Status == model.TaskStatusFailure {
		message, code := responseError(upstream.Error)
		if message == "" {
			message = task.FailReason
		}
		video.Error = &dto.OpenAIVideoError{Message: message, Code: code}
	}
	return common.Marshal(video)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func requestDuration(req relaycommon.TaskSubmitReq) int {
	if req.Duration > 0 {
		return req.Duration
	}
	if seconds, err := strconv.Atoi(req.Seconds); err == nil && seconds > 0 {
		return seconds
	}
	return DefaultDuration
}

func requestAspectRatio(req relaycommon.TaskSubmitReq) string {
	if req.Metadata != nil {
		if value, ok := req.Metadata["aspect_ratio"]; ok {
			if ratio := strings.TrimSpace(fmt.Sprint(value)); ratio != "" {
				return ratio
			}
		}
		if value, ok := req.Metadata["ratio"]; ok {
			if ratio := strings.TrimSpace(fmt.Sprint(value)); ratio != "" {
				return ratio
			}
		}
	}
	size := strings.ToLower(strings.TrimSpace(req.Size))
	switch size {
	case "9:16", "720x1280", "720*1280":
		return "9:16"
	case "16:9", "1280x720", "1280*720":
		return "16:9"
	case "":
		return DefaultAspectRatio
	default:
		return size
	}
}

func toVideoStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "created", "submitted", "queued", "pending":
		return dto.VideoStatusQueued
	case "processing", "running", "in_progress":
		return dto.VideoStatusInProgress
	case "succeeded", "success", "completed":
		return dto.VideoStatusCompleted
	case "failed", "error", "cancelled", "canceled":
		return dto.VideoStatusFailed
	default:
		return dto.VideoStatusUnknown
	}
}

func responseError(raw json.RawMessage) (string, string) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", ""
	}
	var message string
	if common.Unmarshal(raw, &message) == nil {
		return message, ""
	}
	var value map[string]any
	if common.Unmarshal(raw, &value) != nil {
		return string(raw), ""
	}
	message, _ = value["message"].(string)
	code := ""
	if rawCode, ok := value["code"]; ok {
		code = fmt.Sprint(rawCode)
	}
	return message, code
}

func firstFormValue(form *multipart.Form, names ...string) string {
	for _, name := range names {
		if values := form.Value[name]; len(values) > 0 {
			if value := strings.TrimSpace(values[0]); value != "" {
				return value
			}
		}
	}
	return ""
}

func referenceFiles(form *multipart.Form) []*multipart.FileHeader {
	files := make([]*multipart.FileHeader, 0)
	for _, name := range []string{"references", "input_reference", "image", "images"} {
		files = append(files, form.File[name]...)
	}
	return files
}

func validateReferenceFiles(form *multipart.Form) error {
	files := referenceFiles(form)
	if len(files) > MaxReferenceCount {
		return fmt.Errorf("references must contain at most %d files", MaxReferenceCount)
	}

	imageCount := 0
	mediaCount := 0
	for _, file := range files {
		kind, limit, err := referenceFilePolicy(file)
		if err != nil {
			return err
		}
		if file.Size > limit {
			return fmt.Errorf("reference file %s exceeds the %d MB limit", file.Filename, limit/(1024*1024))
		}
		if kind == "image" {
			imageCount++
		} else {
			mediaCount++
		}
	}
	if imageCount > MaxImageReferences {
		return fmt.Errorf("references must contain at most %d images", MaxImageReferences)
	}
	if mediaCount > MaxMediaReferences {
		return fmt.Errorf("video and audio references must contain at most %d files in total", MaxMediaReferences)
	}
	return nil
}

func referenceFilePolicy(file *multipart.FileHeader) (string, int64, error) {
	extension := strings.ToLower(filepath.Ext(file.Filename))
	contentType := strings.ToLower(strings.TrimSpace(file.Header.Get("Content-Type")))
	switch {
	case contentType == "image/jpeg" || contentType == "image/png" || contentType == "image/webp" ||
		extension == ".jpg" || extension == ".jpeg" || extension == ".png" || extension == ".webp":
		return "image", MaxImageReferenceMB * 1024 * 1024, nil
	case contentType == "video/mp4" || contentType == "video/quicktime" || contentType == "video/webm" || contentType == "video/x-m4v" ||
		extension == ".mp4" || extension == ".mov" || extension == ".webm" || extension == ".m4v":
		return "video", MaxVideoReferenceMB * 1024 * 1024, nil
	case contentType == "audio/mpeg" || contentType == "audio/wav" || contentType == "audio/x-wav" || contentType == "audio/mp4" ||
		extension == ".mp3" || extension == ".wav" || extension == ".m4a":
		return "audio", MaxAudioReferenceMB * 1024 * 1024, nil
	default:
		return "", 0, fmt.Errorf("unsupported reference file type: %s", file.Filename)
	}
}

func copyReferenceFiles(writer *multipart.Writer, form *multipart.Form) error {
	for _, fileHeader := range referenceFiles(form) {
		file, err := fileHeader.Open()
		if err != nil {
			return errors.Wrapf(err, "open reference file %s", fileHeader.Filename)
		}
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
			"name":     "references",
			"filename": filepath.Base(fileHeader.Filename),
		}))
		if contentType := fileHeader.Header.Get("Content-Type"); contentType != "" {
			header.Set("Content-Type", contentType)
		}
		part, err := writer.CreatePart(header)
		if err == nil {
			_, err = io.Copy(part, file)
		}
		closeErr := file.Close()
		if err != nil {
			return errors.Wrapf(err, "copy reference file %s", fileHeader.Filename)
		}
		if closeErr != nil {
			return errors.Wrapf(closeErr, "close reference file %s", fileHeader.Filename)
		}
	}
	return nil
}
