package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

type playgroundVideoStatus struct {
	RequestID string
	Status    string
	Progress  any
	VideoURL  string
}

func (s *PlaygroundService) SubmitJob(ctx context.Context, userID int64, input SubmitPlaygroundJobInput) (*PlaygroundTask, error) {
	input.Kind = strings.TrimSpace(input.Kind)
	input.Model = strings.TrimSpace(input.Model)
	input.APIKey = strings.TrimSpace(input.APIKey)
	if input.Kind == "" || input.Model == "" || input.APIKey == "" {
		return nil, fmt.Errorf("%w: task kind, model and api key are required", ErrPlaygroundInvalidInput)
	}
	if len(input.RequestPayload) == 0 {
		return nil, fmt.Errorf("%w: request payload is required", ErrPlaygroundInvalidInput)
	}
	task, err := s.CreateTask(ctx, userID, CreatePlaygroundTaskInput{
		Kind:           input.Kind,
		Status:         "pending",
		Model:          input.Model,
		RequestPayload: input.RequestPayload,
		ResultPayload:  json.RawMessage(`{}`),
	})
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.running[task.ID] = cancel
	s.mu.Unlock()
	go s.runJob(runCtx, userID, task.ID, input)
	return task, nil
}

func (s *PlaygroundService) runJob(ctx context.Context, userID, taskID int64, input SubmitPlaygroundJobInput) {
	defer func() {
		if recovered := recover(); recovered != nil {
			reportPlayground500ServiceDebugEvent("M", "playground_jobs.go:runJob", "[DEBUG] playground runJob panic", map[string]any{
				"task_id":   taskID,
				"user_id":   userID,
				"kind":      input.Kind,
				"model":     input.Model,
				"recovered": fmt.Sprint(recovered),
			})
		}
		s.mu.Lock()
		delete(s.running, taskID)
		s.mu.Unlock()
	}()
	reportPlayground500ServiceDebugEvent("N", "playground_jobs.go:runJob", "[DEBUG] playground runJob start", map[string]any{
		"task_id": taskID,
		"user_id": userID,
		"kind":    input.Kind,
		"model":   input.Model,
	})
	_, updateErr := s.UpdateTask(context.Background(), userID, taskID, UpdatePlaygroundTaskInput{
		Status:        "running",
		ResultPayload: json.RawMessage(`{}`),
	})
	if updateErr != nil {
		logger.L().Error("playground.job.update_running_failed",
			zap.Int64("task_id", taskID),
			zap.Int64("user_id", userID),
			zap.String("kind", input.Kind),
			zap.Error(updateErr),
		)
	}
	reportPlayground500ServiceDebugEvent("O", "playground_jobs.go:runJob", "[DEBUG] playground runJob update running result", map[string]any{
		"task_id": taskID,
		"err":     fmt.Sprint(updateErr),
	})
	err := s.executeJob(ctx, userID, taskID, input)
	reportPlayground500ServiceDebugEvent("P", "playground_jobs.go:runJob", "[DEBUG] playground runJob execute result", map[string]any{
		"task_id": taskID,
		"err":     fmt.Sprint(err),
	})
	if err == nil || ctx.Err() != nil {
		return
	}
	if _, failUpdateErr := s.UpdateTask(context.Background(), userID, taskID, UpdatePlaygroundTaskInput{
		Status:        "failed",
		ErrorMessage:  err.Error(),
		ResultPayload: json.RawMessage(fmt.Sprintf(`{"error":%q}`, err.Error())),
	}); failUpdateErr != nil {
		logger.L().Error("playground.job.update_failed_failed",
			zap.Int64("task_id", taskID),
			zap.Int64("user_id", userID),
			zap.String("kind", input.Kind),
			zap.Error(failUpdateErr),
		)
	}
}

func (s *PlaygroundService) executeJob(ctx context.Context, userID, taskID int64, input SubmitPlaygroundJobInput) error {
	payload, err := parsePlaygroundJobPayload(input.Kind, input.RequestPayload)
	if err != nil {
		return err
	}
	switch input.Kind {
	case "image", "edit", "image-translate", "watermark":
		return s.executeImageJob(ctx, userID, taskID, input, *payload)
	case "batch-main", "batch-clone":
		return s.executeBatchImageJob(ctx, userID, taskID, input, *payload)
	case "chat", "copywriting", "audio-transcribe":
		return s.executeTextJob(ctx, userID, taskID, input, *payload)
	case "audio-generate", "audio-voice-design", "audio-voice-clone":
		return s.executeAudioJob(ctx, userID, taskID, input, *payload)
	case "video":
		return s.executeVideoJob(ctx, userID, taskID, input, *payload)
	default:
		return fmt.Errorf("%w: unsupported playground job kind %s", ErrPlaygroundInvalidInput, input.Kind)
	}
}

func (s *PlaygroundService) executeImageJob(ctx context.Context, userID, taskID int64, input SubmitPlaygroundJobInput, payload playgroundJobPayload) error {
	endpoint, requestBody, err := payload.buildImageRequest(input.Kind, input.Model)
	if err != nil {
		return err
	}
	// #region debug-point J:backend-async-edit-request
	if input.Kind == "edit" {
		reportImageEditDebugEvent("J", "playground_jobs.go:executeImageJob", "[DEBUG] backend async edit request start", map[string]any{
			"task_id":      taskID,
			"model":        input.Model,
			"job_endpoint": endpoint,
			"asset_kind":   payload.AssetKind,
		})
	}
	// #endregion
	body, header, err := doPlaygroundJSONRequest(ctx, input.InternalBaseURL, input.APIKey, http.MethodPost, endpoint, requestBody)
	if err != nil {
		// #region debug-point K:backend-async-edit-error
		if input.Kind == "edit" {
			reportImageEditDebugEvent("K", "playground_jobs.go:executeImageJob", "[DEBUG] backend async edit upstream error", map[string]any{
				"task_id": taskID,
				"error":   err.Error(),
			})
		}
		// #endregion
		return err
	}
	requestID := firstNonEmptyJSON(body, "request_id", "id")
	imageURL := extractImageURL(body)
	if imageURL == "" {
		// #region debug-point L:backend-async-edit-empty-image
		if input.Kind == "edit" {
			reportImageEditDebugEvent("L", "playground_jobs.go:executeImageJob", "[DEBUG] backend async edit completed without image", map[string]any{
				"task_id":    taskID,
				"request_id": requestID,
				"body_keys":  sortedMapKeys(body),
			})
		}
		// #endregion
		return fmt.Errorf("图片任务完成但未返回图片")
	}
	if _, err := s.createTaskAsset(context.Background(), userID, taskID, input.InternalBaseURL, payload.AssetKind, payload.Title, imageURL, "image/png", mergePlaygroundAssetMetadata(nil, mergeStringAnyMaps(payload.Metadata, map[string]any{
		"request_id": requestID,
		"inline":     strings.HasPrefix(strings.ToLower(imageURL), "data:"),
		"headers":    header.Get("Content-Type"),
	}))); err != nil {
		return err
	}
	resultPayload := mergeStringAnyMaps(payload.Metadata, map[string]any{
		"request_id": requestID,
		"url":        imageURL,
	})
	return s.updateSucceededTask(context.Background(), userID, taskID, requestID, resultPayload)
}

func (s *PlaygroundService) executeBatchImageJob(ctx context.Context, userID, taskID int64, input SubmitPlaygroundJobInput, payload playgroundJobPayload) error {
	endpoint, requestBodies, err := payload.buildBatchImageRequests(input.Model)
	if err != nil {
		return err
	}
	succeeded := 0
	failed := 0
	var firstRequestID string
	var firstImageURL string
	for _, itemBody := range requestBodies {
		if ctx.Err() != nil {
			_, _ = s.UpdateTask(context.Background(), userID, taskID, UpdatePlaygroundTaskInput{
				Status:        "canceled",
				ResultPayload: json.RawMessage(fmt.Sprintf(`{"succeeded":%d,"failed":%d}`, succeeded, failed)),
			})
			return nil
		}
		body, _, err := doPlaygroundJSONRequest(ctx, input.InternalBaseURL, input.APIKey, http.MethodPost, endpoint, itemBody)
		if err != nil {
			failed++
			continue
		}
		imageURL := extractImageURL(body)
		if imageURL == "" {
			failed++
			continue
		}
		requestID := firstNonEmptyJSON(body, "request_id", "id")
		if firstRequestID == "" {
			firstRequestID = requestID
		}
		if firstImageURL == "" {
			firstImageURL = imageURL
		}
		if _, err := s.createTaskAsset(context.Background(), userID, taskID, input.InternalBaseURL, payload.AssetKind, payload.Title, imageURL, "image/png", mergePlaygroundAssetMetadata(nil, mergeStringAnyMaps(payload.Metadata, map[string]any{
			"request_id": requestID,
			"inline":     strings.HasPrefix(strings.ToLower(imageURL), "data:"),
		}))); err != nil {
			failed++
			continue
		}
		succeeded++
	}
	if succeeded == 0 {
		return s.updateFailedTask(context.Background(), userID, taskID, firstRequestID, "批量任务全部失败", map[string]any{
			"succeeded": succeeded,
			"failed":    failed,
		})
	}
	return s.updateSucceededTask(context.Background(), userID, taskID, firstRequestID, mergeStringAnyMaps(payload.Metadata, map[string]any{
		"request_id": requestIDOrEmpty(firstRequestID),
		"url":        firstImageURL,
		"succeeded":  succeeded,
		"failed":     failed,
	}))
}

func (s *PlaygroundService) executeTextJob(ctx context.Context, userID, taskID int64, input SubmitPlaygroundJobInput, payload playgroundJobPayload) error {
	endpoint, requestBody, err := payload.buildTextRequest(input.Kind, input.Model)
	if err != nil {
		return err
	}
	body, _, err := doPlaygroundJSONRequest(ctx, input.InternalBaseURL, input.APIKey, http.MethodPost, endpoint, requestBody)
	if err != nil {
		return err
	}
	requestID := firstNonEmptyJSON(body, "request_id", "id")
	text := extractMessageText(body)
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("文本任务完成但未返回文本")
	}
	if _, err := s.CreateAsset(context.Background(), userID, CreatePlaygroundAssetInput{
		TaskID:      &taskID,
		Kind:        "text",
		Title:       payload.Title,
		Content:     text,
		ContentType: "text/plain",
		Metadata:    mustJSON(mergeStringAnyMaps(payload.Metadata, map[string]any{"request_id": requestID})),
	}); err != nil {
		return err
	}
	result := mergeStringAnyMaps(payload.Metadata, map[string]any{
		"request_id": requestID,
		"content":    text,
		"text":       text,
		"transcript": text,
	})
	return s.updateSucceededTask(context.Background(), userID, taskID, requestID, result)
}

func (s *PlaygroundService) executeAudioJob(ctx context.Context, userID, taskID int64, input SubmitPlaygroundJobInput, payload playgroundJobPayload) error {
	endpoint, requestBody, err := payload.buildAudioRequest(input.Kind, input.Model)
	if err != nil {
		return err
	}
	body, _, err := doPlaygroundJSONRequest(ctx, input.InternalBaseURL, input.APIKey, http.MethodPost, endpoint, requestBody)
	if err != nil {
		return err
	}
	requestID := firstNonEmptyJSON(body, "request_id", "id")
	text := extractMessageText(body)
	audioURL, dataURL, contentType := extractAudioResult(body)
	if audioURL == "" && dataURL == "" && strings.TrimSpace(text) == "" {
		return fmt.Errorf("音频任务完成但未返回有效结果")
	}
	if dataURL != "" || audioURL != "" {
		metadata := mergeStringAnyMaps(payload.Metadata, map[string]any{
			"request_id": requestID,
			"mode":       playgroundFirstNonEmpty(payload.Mode, playgroundFirstNonEmptyString(payload.Metadata["mode"])),
		})
		if audioURL != "" {
			metadata["auth_token"] = input.APIKey
		}
		if _, err := s.createTaskAsset(context.Background(), userID, taskID, input.InternalBaseURL, "audio", payload.Title, playgroundFirstNonEmpty(dataURL, audioURL), contentType, mergePlaygroundAssetMetadata(nil, metadata)); err != nil {
			return err
		}
	}
	if strings.TrimSpace(text) != "" {
		title := "配音文本"
		if input.Kind == "audio-transcribe" {
			title = payload.Title
		}
		if _, err := s.CreateAsset(context.Background(), userID, CreatePlaygroundAssetInput{
			TaskID:      &taskID,
			Kind:        "text",
			Title:       title,
			Content:     text,
			ContentType: "text/plain",
			Metadata:    mustJSON(mergeStringAnyMaps(payload.Metadata, map[string]any{"request_id": requestID})),
		}); err != nil {
			return err
		}
	}
	return s.updateSucceededTask(context.Background(), userID, taskID, requestID, mergeStringAnyMaps(payload.Metadata, map[string]any{
		"request_id": requestID,
		"text":       text,
		"transcript": text,
		"audio_url":  playgroundFirstNonEmpty(audioURL, dataURL),
	}))
}

func (s *PlaygroundService) executeVideoJob(ctx context.Context, userID, taskID int64, input SubmitPlaygroundJobInput, payload playgroundJobPayload) error {
	endpoint, requestBody, err := payload.buildVideoRequest(input.Model)
	if err != nil {
		return err
	}
	body, _, err := doPlaygroundJSONRequest(ctx, input.InternalBaseURL, input.APIKey, http.MethodPost, endpoint, requestBody)
	if err != nil {
		return err
	}
	video := extractVideoStatus(body)
	if video.RequestID == "" {
		return fmt.Errorf("视频任务未返回 request_id")
	}
	_, _ = s.UpdateTask(context.Background(), userID, taskID, UpdatePlaygroundTaskInput{
		Status:    "submitted",
		RequestID: video.RequestID,
		ResultPayload: mustJSON(mergeStringAnyMaps(payload.Metadata, map[string]any{
			"status":   playgroundFirstNonEmpty(video.Status, "queued"),
			"progress": video.Progress,
		})),
	})
	status, err := s.pollVideoJob(ctx, input.InternalBaseURL, input.APIKey, video.RequestID)
	if err != nil {
		if ctx.Err() != nil {
			_, _ = s.UpdateTask(context.Background(), userID, taskID, UpdatePlaygroundTaskInput{
				Status:        "canceled",
				RequestID:     video.RequestID,
				ResultPayload: mustJSON(map[string]any{"status": "canceled"}),
			})
			return nil
		}
		return err
	}
	videoURL := status.VideoURL
	if videoURL == "" {
		videoURL = fmt.Sprintf("/v1/videos/%s/content", status.RequestID)
	}
	if _, err := s.createTaskAsset(context.Background(), userID, taskID, input.InternalBaseURL, "video", payload.Title, videoURL, "video/mp4", mergePlaygroundAssetMetadata(nil, mergeStringAnyMaps(payload.Metadata, map[string]any{
		"request_id": status.RequestID,
		"auth_token": input.APIKey,
	}))); err != nil {
		return err
	}
	return s.updateSucceededTask(context.Background(), userID, taskID, status.RequestID, mergeStringAnyMaps(payload.Metadata, map[string]any{
		"request_id": status.RequestID,
		"status":     playgroundFirstNonEmpty(status.Status, "completed"),
		"progress":   status.Progress,
		"video_url":  videoURL,
	}))
}

func (s *PlaygroundService) pollVideoJob(ctx context.Context, baseURL, apiKey, requestID string) (*playgroundVideoStatus, error) {
	for {
		body, _, err := doPlaygroundJSONRequest(ctx, baseURL, apiKey, http.MethodGet, fmt.Sprintf("/v1/videos/%s", requestID), nil)
		if err != nil {
			return nil, err
		}
		status := extractVideoStatus(body)
		if status.RequestID == "" {
			status.RequestID = requestID
		}
		if status.VideoURL != "" || isFinishedVideoStatus(status.Status) {
			return status, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}
}

func (s *PlaygroundService) createTaskAsset(ctx context.Context, userID, taskID int64, baseURL, kind, title, source, contentType string, metadata json.RawMessage) (*PlaygroundAsset, error) {
	input := CreatePlaygroundAssetInput{
		TaskID:          &taskID,
		Kind:            normalizePlaygroundAssetKind(kind),
		Title:           strings.TrimSpace(title),
		InternalBaseURL: baseURL,
		ContentType:     contentType,
		Metadata:        metadata,
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(source)), "data:") {
		input.Content = source
	} else {
		input.URL = source
	}
	return s.CreateAsset(ctx, userID, input)
}

func (s *PlaygroundService) updateSucceededTask(ctx context.Context, userID, taskID int64, requestID string, payload map[string]any) error {
	_, err := s.UpdateTask(ctx, userID, taskID, UpdatePlaygroundTaskInput{
		Status:        "succeeded",
		RequestID:     requestID,
		ResultPayload: mustJSON(payload),
	})
	if err != nil {
		logger.L().Error("playground.job.update_succeeded_failed",
			zap.Int64("task_id", taskID),
			zap.Int64("user_id", userID),
			zap.String("request_id", requestID),
			zap.Error(err),
		)
	}
	return err
}

func (s *PlaygroundService) updateFailedTask(ctx context.Context, userID, taskID int64, requestID, message string, payload map[string]any) error {
	_, err := s.UpdateTask(ctx, userID, taskID, UpdatePlaygroundTaskInput{
		Status:        "failed",
		RequestID:     requestID,
		ErrorMessage:  message,
		ResultPayload: mustJSON(mergeStringAnyMaps(payload, map[string]any{"error": message})),
	})
	return err
}

func doPlaygroundJSONRequest(ctx context.Context, baseURL, apiKey, method, endpoint string, body json.RawMessage) (map[string]any, http.Header, error) {
	target := strings.TrimRight(playgroundInternalBaseURL(baseURL), "/") + endpoint
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024*1024))
	if err != nil {
		return nil, resp.Header, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.Header, fmt.Errorf("%s", strings.TrimSpace(readPlaygroundErrorMessage(raw, resp.StatusCode)))
	}
	if len(raw) == 0 {
		return map[string]any{}, resp.Header, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, resp.Header, err
	}
	return payload, resp.Header, nil
}

func readPlaygroundErrorMessage(raw []byte, status int) string {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err == nil {
		if message := playgroundFirstNonEmptyString(payload["message"], playgroundNestedMapString(payload, "error", "message")); message != "" {
			return message
		}
	}
	if trimmed := strings.TrimSpace(string(raw)); trimmed != "" {
		return trimmed
	}
	return fmt.Sprintf("request failed: %d", status)
}

func extractImageURL(payload map[string]any) string {
	data, _ := payload["data"].([]any)
	if len(data) == 0 {
		return ""
	}
	first, _ := data[0].(map[string]any)
	if first == nil {
		return ""
	}
	if url := playgroundFirstNonEmptyString(first["url"]); url != "" {
		return url
	}
	if b64 := playgroundFirstNonEmptyString(first["b64_json"]); b64 != "" {
		return "data:image/png;base64," + b64
	}
	return ""
}

func extractMessageText(payload map[string]any) string {
	choices, _ := payload["choices"].([]any)
	if len(choices) == 0 {
		if text := playgroundFirstNonEmptyString(payload["transcript"], payload["text"]); text != "" {
			return text
		}
		return ""
	}
	choice, _ := choices[0].(map[string]any)
	message, _ := choice["message"].(map[string]any)
	content := message["content"]
	switch value := content.(type) {
	case string:
		return value
	case []any:
		parts := make([]string, 0, len(value))
		for _, item := range value {
			part, _ := item.(map[string]any)
			if text := playgroundFirstNonEmptyString(part["text"]); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "")
	default:
		return playgroundFirstNonEmptyString(payload["transcript"], payload["text"])
	}
}

func extractAudioResult(payload map[string]any) (string, string, string) {
	audio := firstNonEmptyMap(
		asMap(payload["audio"]),
		asMap(nestedMapValue(payload, "data", "audio")),
		asMap(payload["output_audio"]),
	)
	if base64Value := playgroundFirstNonEmptyString(
		audio["b64_json"],
		audio["data"],
		audio["audio_base64"],
		audio["base64"],
		payload["audio_base64"],
	); base64Value != "" {
		format := playgroundFirstNonEmptyString(audio["format"], payload["format"], "wav")
		return "", "data:" + audioMimeType(format) + ";base64," + base64Value, audioMimeType(format)
	}
	if url := playgroundFirstNonEmptyString(
		audio["url"],
		audio["audio_url"],
		audio["path"],
		audio["src"],
		payload["audio_url"],
		payload["url"],
	); url != "" {
		format := playgroundFirstNonEmptyString(audio["format"], payload["format"], "wav")
		return url, "", audioMimeType(format)
	}
	choices, _ := payload["choices"].([]any)
	if len(choices) == 0 {
		return "", "", "audio/wav"
	}
	choice, _ := choices[0].(map[string]any)
	message := asMap(choice["message"])
	messageAudio := firstNonEmptyMap(
		asMap(message["audio"]),
		asMap(message["output_audio"]),
	)
	if base64Value := playgroundFirstNonEmptyString(
		messageAudio["b64_json"],
		messageAudio["data"],
		messageAudio["audio_base64"],
		messageAudio["base64"],
	); base64Value != "" {
		format := playgroundFirstNonEmptyString(messageAudio["format"], "wav")
		return "", "data:" + audioMimeType(format) + ";base64," + base64Value, audioMimeType(format)
	}
	if url := playgroundFirstNonEmptyString(
		messageAudio["url"],
		messageAudio["audio_url"],
		messageAudio["path"],
		messageAudio["src"],
	); url != "" {
		format := playgroundFirstNonEmptyString(messageAudio["format"], "wav")
		return url, "", audioMimeType(format)
	}
	content, _ := message["content"].([]any)
	for _, item := range content {
		part := asMap(item)
		audioPart := firstNonEmptyMap(
			asMap(part["audio"]),
			asMap(part["output_audio"]),
		)
		if base64Value := playgroundFirstNonEmptyString(
			part["b64_json"],
			part["data"],
			part["audio_base64"],
			part["base64"],
			audioPart["b64_json"],
			audioPart["data"],
			audioPart["audio_base64"],
			audioPart["base64"],
		); base64Value != "" {
			format := playgroundFirstNonEmptyString(part["format"], audioPart["format"], "wav")
			return "", "data:" + audioMimeType(format) + ";base64," + base64Value, audioMimeType(format)
		}
		if url := playgroundFirstNonEmptyString(
			part["url"],
			part["audio_url"],
			part["path"],
			part["src"],
			audioPart["url"],
			audioPart["audio_url"],
			audioPart["path"],
			audioPart["src"],
		); url != "" {
			format := playgroundFirstNonEmptyString(part["format"], audioPart["format"], "wav")
			return url, "", audioMimeType(format)
		}
	}
	return "", "", "audio/wav"
}

func extractVideoStatus(payload map[string]any) *playgroundVideoStatus {
	return &playgroundVideoStatus{
		RequestID: playgroundFirstNonEmptyString(payload["request_id"], payload["id"]),
		Status:    playgroundFirstNonEmptyString(payload["status"]),
		Progress:  payload["progress"],
		VideoURL:  playgroundFirstNonEmptyString(nestedMapValue(payload, "video", "url"), nestedMapValue(payload, "output", "video", "url"), payload["output_url"], payload["url"]),
	}
}

func isFinishedVideoStatus(status string) bool {
	value := strings.ToLower(strings.TrimSpace(status))
	switch value {
	case "completed", "succeeded", "ready", "done":
		return true
	default:
		return false
	}
}

func audioMimeType(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "mp3", "mpeg":
		return "audio/mpeg"
	case "m4a", "mp4":
		return "audio/mp4"
	case "ogg":
		return "audio/ogg"
	default:
		return "audio/wav"
	}
}

func firstNonEmptyMap(candidates ...map[string]any) map[string]any {
	for _, candidate := range candidates {
		if len(candidate) > 0 {
			return candidate
		}
	}
	return map[string]any{}
}

func mustJSON(value any) json.RawMessage {
	raw, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

func mergeStringAnyMaps(base map[string]any, extra map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		if value != nil {
			merged[key] = value
		}
	}
	return merged
}

func asMap(value any) map[string]any {
	mapped, _ := value.(map[string]any)
	if mapped == nil {
		return map[string]any{}
	}
	return mapped
}

func nestedMapValue(payload map[string]any, path ...string) any {
	current := any(payload)
	for _, key := range path {
		mapped, _ := current.(map[string]any)
		if mapped == nil {
			return nil
		}
		current = mapped[key]
	}
	return current
}

func playgroundNestedMapString(payload map[string]any, path ...string) string {
	return playgroundFirstNonEmptyString(nestedMapValue(payload, path...))
}

func firstNonEmptyJSON(payload map[string]any, keys ...string) string {
	values := make([]any, 0, len(keys))
	for _, key := range keys {
		values = append(values, payload[key])
	}
	return playgroundFirstNonEmptyString(values...)
}

func playgroundFirstNonEmptyString(values ...any) string {
	for _, item := range values {
		switch value := item.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		case fmt.Stringer:
			if strings.TrimSpace(value.String()) != "" {
				return strings.TrimSpace(value.String())
			}
		}
	}
	return ""
}

func playgroundFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func sortedMapKeys(input map[string]any) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func requestIDOrEmpty(value string) string {
	return strings.TrimSpace(value)
}
