package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mymeetily/mymeetily/internal/agent"
	"github.com/mymeetily/mymeetily/internal/audio"
	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/diskspace"
	"github.com/mymeetily/mymeetily/internal/hardware"
	"github.com/mymeetily/mymeetily/internal/modelmanager"
	"github.com/mymeetily/mymeetily/internal/workflow"
)

// =============================================================================
// App — 应用生命周期管理
// =============================================================================

type App struct {
	ctx             context.Context
	cfg             *config.Config
	cfgMu           sync.RWMutex
	appService      *AppService
	deviceService   *DeviceService
	recordService   *RecordService
	pipelineService *PipelineService
	resultService   *ResultService
}

func NewApp() *App {
	cfg, err := config.LoadWithUserOverrides()
	if err != nil {
		// Fall back to defaults — config.Load already populates defaults on error
		cfg, _ = config.Load("")
	}

	a := &App{cfg: cfg}
	a.appService = &AppService{app: a}
	a.deviceService = &DeviceService{app: a}
	a.recordService = &RecordService{app: a}
	a.pipelineService = &PipelineService{app: a}
	a.resultService = &ResultService{app: a}
	return a
}

func (a *App) configSnapshot() config.Config {
	a.cfgMu.RLock()
	defer a.cfgMu.RUnlock()
	return *a.cfg
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.appService.ctx = ctx
	a.deviceService.ctx = ctx
	a.recordService.ctx = ctx
	a.pipelineService.ctx = ctx
	a.resultService.ctx = ctx

	// Run startup pre-flight checks in background
	go a.runStartupChecks()
}

func (a *App) shutdown(ctx context.Context) {
	if a.recordService != nil && a.recordService.IsRecording() {
		a.recordService.StopRecording()
	}
	if a.pipelineService != nil {
		a.pipelineService.CancelPipeline()
	}
	if a.appService != nil {
		a.appService.CancelModelOperation()
	}
}

func (a *App) runStartupChecks() {
	// Emit status update
	runtime.EventsEmit(a.ctx, "app:status", map[string]interface{}{
		"phase":   "checking",
		"message": "正在檢查依赖...",
	})

	// Check whisper model file
	whisperModelOK := true
	whisperModelMsg := "語音模型已就緒"
	if _, err := os.Stat(a.cfg.ASR.ModelPath); os.IsNotExist(err) {
		whisperModelOK = false
		whisperModelMsg = "找不到語音模型，請執行 init-model 命令下載模型"
	}

	// Check Ollama
	summaryEnabled := false
	ollamaOK := false
	ollamaMsg := "Ollama 服務未連線"
	modelName := a.cfg.Ollama.Model

	startupStatus, err := workflow.DetectSummaryAvailability(a.ctx, a.cfg,
		"./llmmodels/summary/qwen2.5-1.5b-instruct-q4_k_m.gguf",
		"https://hf-mirror.com/Qwen/Qwen2.5-1.5B-Instruct-GGUF/resolve/main/qwen2.5-1.5b-instruct-q4_k_m.gguf",
	)
	if err == nil {
		ollamaOK = true
		summaryEnabled = startupStatus.SummaryEnabled
		if summaryEnabled {
			ollamaMsg = fmt.Sprintf("Ollama 已就緒 (%s)", modelName)
		} else {
			ollamaMsg = startupStatus.SummaryMessage
		}
	} else {
		ollamaMsg = fmt.Sprintf("Ollama 未連線: %v", err)
	}

	// Emit dependency check results
	runtime.EventsEmit(a.ctx, "app:dependency", map[string]interface{}{
		"whisperBinary": map[string]interface{}{
			"ok":      true,
			"message": "引擎已就緒",
		},
		"whisperModel": map[string]interface{}{
			"ok":      whisperModelOK,
			"message": whisperModelMsg,
		},
		"ollama": map[string]interface{}{
			"ok":      ollamaOK,
			"message": ollamaMsg,
		},
		"summaryEnabled": summaryEnabled,
	})

	// Emit ready status
	runtime.EventsEmit(a.ctx, "app:status", map[string]interface{}{
		"phase":          "ready",
		"message":        "就緒",
		"summaryEnabled": summaryEnabled,
	})

	// Emit app info
	runtime.EventsEmit(a.ctx, "app:info", map[string]interface{}{
		"engine":       "whisper.cpp + Ollama",
		"whisperModel": filepath.Base(a.cfg.ASR.ModelPath),
		"llmModel":     a.cfg.Ollama.Model,
	})
}

// =============================================================================
// AppService — 应用信息和系統操作 (暴露给前端)
// =============================================================================

type AppService struct {
	ctx         context.Context
	app         *App
	modelMu     sync.Mutex
	modelBusy   bool
	modelCancel context.CancelFunc
}

type AppInfo struct {
	Engine       string `json:"engine"`
	WhisperModel string `json:"whisperModel"`
	LLMModel     string `json:"llmModel"`
}

func (s *AppService) GetAppInfo() AppInfo {
	s.app.cfgMu.RLock()
	defer s.app.cfgMu.RUnlock()
	return AppInfo{
		Engine:       "whisper.cpp + Ollama",
		WhisperModel: filepath.Base(s.app.cfg.ASR.ModelPath),
		LLMModel:     s.app.cfg.Ollama.Model,
	}
}

type Preferences struct {
	WhisperModel string  `json:"whisperModel"`
	OllamaModel  string  `json:"ollamaModel"`
	Language     string  `json:"language"`
	OutputDir    string  `json:"outputDir"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"maxTokens"`
}

type ModelOption struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Active bool   `json:"active"`
}

type ModelCatalogItem struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
	Active      bool   `json:"active"`
	Recommended bool   `json:"recommended"`
}

type ModelOperationProgress struct {
	Kind       string  `json:"kind"`
	ModelID    string  `json:"modelId"`
	Phase      string  `json:"phase"`
	Message    string  `json:"message"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Percent    float64 `json:"percent"`
}

type MeetingHistoryItem struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	SourceAudio    string `json:"sourceAudio"`
	GeneratedAt    string `json:"generatedAt"`
	Summary        string `json:"summary"`
	HTMLPath       string `json:"htmlPath"`
	MarkdownPath   string `json:"markdownPath"`
	TranscriptPath string `json:"transcriptPath"`
}

func (s *AppService) GetPreferences() Preferences {
	s.app.cfgMu.RLock()
	defer s.app.cfgMu.RUnlock()
	return Preferences{
		WhisperModel: s.app.cfg.ASR.ModelPath,
		OllamaModel:  s.app.cfg.Ollama.Model,
		Language:     s.app.cfg.ASR.Language,
		OutputDir:    s.app.cfg.Output.OutputDir,
		Temperature:  s.app.cfg.Ollama.Temperature,
		MaxTokens:    s.app.cfg.Ollama.MaxTokens,
	}
}

func (s *AppService) SavePreferences(p Preferences) error {
	if s.app.recordService.IsRecording() || s.app.pipelineService.IsRunning() {
		return fmt.Errorf("錄音或 AI 處理期間不能切換設定")
	}
	if strings.TrimSpace(p.WhisperModel) == "" {
		return fmt.Errorf("請選擇 Whisper 模型")
	}
	if info, err := os.Stat(p.WhisperModel); err != nil || info.IsDir() {
		return fmt.Errorf("Whisper 模型不存在：%s", p.WhisperModel)
	}
	if strings.TrimSpace(p.OllamaModel) == "" {
		return fmt.Errorf("請選擇 Ollama 模型")
	}
	if strings.TrimSpace(p.OutputDir) == "" {
		return fmt.Errorf("輸出資料夾不能空白")
	}
	if p.Temperature < 0 || p.Temperature > 2 {
		return fmt.Errorf("AI 溫度必須介於 0 到 2")
	}
	if p.MaxTokens < 256 || p.MaxTokens > 32768 {
		return fmt.Errorf("最大輸出 token 必須介於 256 到 32768")
	}

	s.app.cfgMu.Lock()
	updated := *s.app.cfg
	updated.ASR.ModelPath = filepath.Clean(p.WhisperModel)
	updated.ASR.Language = strings.TrimSpace(p.Language)
	updated.Ollama.Model = strings.TrimSpace(p.OllamaModel)
	updated.Ollama.Temperature = p.Temperature
	updated.Ollama.MaxTokens = p.MaxTokens
	updated.Output.OutputDir = filepath.Clean(p.OutputDir)
	if err := config.SaveUser(&updated); err != nil {
		s.app.cfgMu.Unlock()
		return err
	}
	*s.app.cfg = updated
	s.app.cfgMu.Unlock()

	runtime.EventsEmit(s.ctx, "settings:changed", s.GetAppInfo())
	return nil
}

func (s *AppService) ListWhisperModels() ([]ModelOption, error) {
	s.app.cfgMu.RLock()
	activeModel := filepath.Clean(s.app.cfg.ASR.ModelPath)
	s.app.cfgMu.RUnlock()

	paths, err := filepath.Glob(filepath.Join("llmmodels", "whisper", "*.bin"))
	if err != nil {
		return nil, err
	}
	models := make([]ModelOption, 0, len(paths))
	for _, modelPath := range paths {
		info, err := os.Stat(modelPath)
		if err != nil || info.IsDir() {
			continue
		}
		cleanPath := filepath.Clean(modelPath)
		models = append(models, ModelOption{
			Name:   strings.TrimSuffix(filepath.Base(cleanPath), filepath.Ext(cleanPath)),
			Path:   cleanPath,
			Size:   info.Size(),
			Active: strings.EqualFold(cleanPath, activeModel),
		})
	}
	return models, nil
}

func (s *AppService) ListOllamaModels() ([]ModelOption, error) {
	s.app.cfgMu.RLock()
	endpoint := strings.TrimRight(s.app.cfg.Ollama.Endpoint, "/")
	activeModel := s.app.cfg.Ollama.Model
	s.app.cfgMu.RUnlock()

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(endpoint + "/api/tags")
	if err != nil {
		return nil, fmt.Errorf("連線 Ollama：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("Ollama 回傳 HTTP %d", response.StatusCode)
	}
	var payload struct {
		Models []struct {
			Name string `json:"name"`
			Size int64  `json:"size"`
		} `json:"models"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("解析 Ollama 模型清單：%w", err)
	}
	models := make([]ModelOption, 0, len(payload.Models))
	for _, model := range payload.Models {
		models = append(models, ModelOption{
			Name:   model.Name,
			Path:   model.Name,
			Size:   model.Size,
			Active: model.Name == activeModel,
		})
	}
	return models, nil
}

func (s *AppService) GetHardwareProfile() hardware.Profile {
	return hardware.Detect()
}

func (s *AppService) ListModelCatalog() ([]ModelCatalogItem, error) {
	profile := hardware.Detect()
	prefs := s.GetPreferences()
	whisperDir := filepath.Dir(prefs.WhisperModel)
	installedOllama := map[string]bool{}
	if models, err := s.ListOllamaModels(); err == nil {
		for _, model := range models {
			installedOllama[strings.ToLower(model.Path)] = true
		}
	}

	items := make([]ModelCatalogItem, 0)
	for _, catalog := range modelmanager.Catalog() {
		path := catalog.ID
		installed := false
		active := false
		if catalog.Kind == "whisper" {
			path = filepath.Clean(filepath.Join(whisperDir, catalog.FileName))
			if info, err := os.Stat(path); err == nil && !info.IsDir() {
				installed = true
			}
			active = strings.EqualFold(filepath.Clean(prefs.WhisperModel), path)
		} else {
			installed = installedOllama[strings.ToLower(catalog.ID)]
			active = strings.EqualFold(prefs.OllamaModel, catalog.ID)
		}
		items = append(items, ModelCatalogItem{
			ID: catalog.ID, Kind: catalog.Kind, Name: catalog.Name, Path: path,
			Size: catalog.Size, Description: catalog.Description,
			Installed: installed, Active: active,
			Recommended: catalog.Kind == "whisper" && catalog.ID == profile.RecommendedModel,
		})
	}
	return items, nil
}

func (s *AppService) beginModelOperation() (context.Context, context.CancelFunc, error) {
	if s.app.recordService.IsRecording() || s.app.pipelineService.IsRunning() {
		return nil, nil, fmt.Errorf("錄音或 AI 處理期間不能變更模型")
	}
	s.modelMu.Lock()
	defer s.modelMu.Unlock()
	if s.modelBusy {
		return nil, nil, fmt.Errorf("已有模型作業正在執行")
	}
	ctx, cancel := context.WithCancel(s.ctx)
	s.modelBusy = true
	s.modelCancel = cancel
	return ctx, cancel, nil
}

func (s *AppService) finishModelOperation(cancel context.CancelFunc) {
	cancel()
	s.modelMu.Lock()
	s.modelBusy = false
	s.modelCancel = nil
	s.modelMu.Unlock()
}

func (s *AppService) emitModelProgress(progress ModelOperationProgress) {
	if progress.Total > 0 {
		progress.Percent = float64(progress.Downloaded) / float64(progress.Total) * 100
	}
	runtime.EventsEmit(s.ctx, "model:progress", progress)
}

func (s *AppService) InstallWhisperModel(modelID string) error {
	item, ok := modelmanager.Find("whisper", modelID)
	if !ok {
		return fmt.Errorf("不支援的 Whisper 模型：%s", modelID)
	}
	ctx, cancel, err := s.beginModelOperation()
	if err != nil {
		return err
	}
	defer s.finishModelOperation(cancel)

	destination := filepath.Join(filepath.Dir(s.GetPreferences().WhisperModel), item.FileName)
	if available, diskErr := diskspace.FreeBytes(filepath.Dir(destination)); diskErr == nil {
		required := uint64(float64(item.Size) * 1.15)
		if available < required {
			return fmt.Errorf("硬碟空間不足：需要約 %.1f GB，可用空間只有 %.1f GB", float64(required)/1024/1024/1024, float64(available)/1024/1024/1024)
		}
	}
	s.emitModelProgress(ModelOperationProgress{Kind: "whisper", ModelID: modelID, Phase: "starting", Message: "準備下載 Whisper 模型"})
	err = modelmanager.Download(ctx, nil, item.URL, destination, func(downloaded, total int64) {
		s.emitModelProgress(ModelOperationProgress{Kind: "whisper", ModelID: modelID, Phase: "downloading", Message: "正在下載 Whisper 模型", Downloaded: downloaded, Total: total})
	})
	if err != nil {
		phase := "error"
		message := err.Error()
		if errors.Is(err, context.Canceled) {
			phase, message = "cancelled", "下載已取消"
		}
		s.emitModelProgress(ModelOperationProgress{Kind: "whisper", ModelID: modelID, Phase: phase, Message: message})
		return err
	}
	s.emitModelProgress(ModelOperationProgress{Kind: "whisper", ModelID: modelID, Phase: "done", Message: "Whisper 模型安裝完成", Downloaded: item.Size, Total: item.Size, Percent: 100})
	return nil
}

func (s *AppService) InstallOllamaModel(modelID string) error {
	item, ok := modelmanager.Find("ollama", modelID)
	if !ok {
		return fmt.Errorf("不支援的 Ollama 模型：%s", modelID)
	}
	ctx, cancel, err := s.beginModelOperation()
	if err != nil {
		return err
	}
	defer s.finishModelOperation(cancel)

	endpoint := strings.TrimRight(s.app.configSnapshot().Ollama.Endpoint, "/")
	body, _ := json.Marshal(map[string]interface{}{"name": item.ID, "stream": true})
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/api/pull", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	s.emitModelProgress(ModelOperationProgress{Kind: "ollama", ModelID: modelID, Phase: "starting", Message: "正在連線 Ollama"})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		s.emitModelProgress(ModelOperationProgress{Kind: "ollama", ModelID: modelID, Phase: "error", Message: err.Error()})
		return fmt.Errorf("下載 Ollama 模型：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Ollama 回傳 HTTP %d", response.StatusCode)
	}
	decoder := json.NewDecoder(response.Body)
	for decoder.More() {
		var payload struct {
			Status    string `json:"status"`
			Error     string `json:"error"`
			Completed int64  `json:"completed"`
			Total     int64  `json:"total"`
		}
		if err := decoder.Decode(&payload); err != nil {
			if errors.Is(err, context.Canceled) {
				s.emitModelProgress(ModelOperationProgress{Kind: "ollama", ModelID: modelID, Phase: "cancelled", Message: "下載已取消"})
			}
			return fmt.Errorf("讀取 Ollama 下載進度：%w", err)
		}
		if payload.Error != "" {
			return fmt.Errorf("Ollama：%s", payload.Error)
		}
		s.emitModelProgress(ModelOperationProgress{Kind: "ollama", ModelID: modelID, Phase: "downloading", Message: payload.Status, Downloaded: payload.Completed, Total: payload.Total})
	}
	s.emitModelProgress(ModelOperationProgress{Kind: "ollama", ModelID: modelID, Phase: "done", Message: "Ollama 模型安裝完成", Percent: 100})
	return nil
}

func (s *AppService) CancelModelOperation() {
	s.modelMu.Lock()
	cancel := s.modelCancel
	s.modelMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *AppService) DeleteModel(kind, modelID string) error {
	item, ok := modelmanager.Find(kind, modelID)
	if !ok {
		return fmt.Errorf("不支援的模型：%s", modelID)
	}
	if s.app.recordService.IsRecording() || s.app.pipelineService.IsRunning() {
		return fmt.Errorf("錄音或 AI 處理期間不能刪除模型")
	}
	prefs := s.GetPreferences()
	if kind == "whisper" {
		path := filepath.Clean(filepath.Join(filepath.Dir(prefs.WhisperModel), item.FileName))
		if strings.EqualFold(filepath.Clean(prefs.WhisperModel), path) {
			return fmt.Errorf("使用中的 Whisper 模型不能刪除，請先切換模型")
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("刪除 Whisper 模型：%w", err)
		}
		return nil
	}
	if strings.EqualFold(prefs.OllamaModel, item.ID) {
		return fmt.Errorf("使用中的 Ollama 模型不能刪除，請先切換模型")
	}
	endpoint := strings.TrimRight(s.app.configSnapshot().Ollama.Endpoint, "/")
	body, _ := json.Marshal(map[string]string{"name": item.ID})
	request, err := http.NewRequestWithContext(s.ctx, http.MethodDelete, endpoint+"/api/delete", bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("刪除 Ollama 模型：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Ollama 回傳 HTTP %d", response.StatusCode)
	}
	return nil
}

func (s *AppService) OpenOutputFolder(path string) {
	dir := filepath.Dir(path)
	runtime.BrowserOpenURL(s.ctx, "file:///"+filepath.ToSlash(dir))
}

func (s *AppService) ListMeetingHistory() ([]MeetingHistoryItem, error) {
	root := s.GetPreferences().OutputDir
	items := make([]MeetingHistoryItem, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".html") || !strings.Contains(filepath.Base(path), "_紀要_") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		base := strings.TrimSuffix(path, filepath.Ext(path))
		markdownPath := base + ".md"
		if _, err := os.Stat(markdownPath); err != nil {
			markdownPath = ""
		}
		item := MeetingHistoryItem{
			ID: path, Title: filepath.Base(filepath.Dir(path)),
			GeneratedAt: info.ModTime().Format(time.RFC3339), HTMLPath: path,
			MarkdownPath: markdownPath,
		}
		if markdownPath != "" {
			if data, err := os.ReadFile(markdownPath); err == nil {
				item.SourceAudio, item.Summary = historyMetadata(string(data))
			}
		}
		if item.SourceAudio != "" {
			item.Title = strings.TrimSuffix(item.SourceAudio, filepath.Ext(item.SourceAudio))
		}
		name := filepath.Base(base)
		if marker := strings.LastIndex(name, "_紀要_"); marker >= 0 {
			matches, _ := filepath.Glob(filepath.Join(filepath.Dir(path), name[:marker]+"_轉寫_*.txt"))
			if len(matches) > 0 {
				item.TranscriptPath = matches[len(matches)-1]
			}
		}
		items = append(items, item)
		return nil
	})
	if os.IsNotExist(err) {
		return items, nil
	}
	if err != nil {
		return nil, fmt.Errorf("讀取會議歷史：%w", err)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].GeneratedAt > items[j].GeneratedAt })
	return items, nil
}

func historyMetadata(markdown string) (source, summary string) {
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "> 源檔案：") {
			source = strings.TrimSpace(strings.TrimPrefix(trimmed, "> 源檔案："))
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ">") || trimmed == "---" {
			continue
		}
		if summary == "" {
			summary = strings.TrimLeft(trimmed, "-* ")
			if len([]rune(summary)) > 120 {
				summary = string([]rune(summary)[:120]) + "…"
			}
		}
	}
	return source, summary
}

func (s *AppService) OpenHistoryReport(path string) error {
	root, err := filepath.Abs(s.GetPreferences().OutputDir)
	if err != nil {
		return err
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("報告不在輸出資料夾內")
	}
	if info, err := os.Stat(target); err != nil || info.IsDir() {
		return fmt.Errorf("找不到會議報告")
	}
	runtime.BrowserOpenURL(s.ctx, "file:///"+filepath.ToSlash(target))
	return nil
}

func (s *AppService) CopyToClipboard(text string) error {
	return runtime.ClipboardSetText(s.ctx, text)
}

func (s *AppService) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(s.ctx, runtime.OpenDialogOptions{
		Title: "選擇音訊檔案",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Audio Files (*.mp3;*.wav;*.m4a;*.flac;*.ogg;*.wma;*.aac)",
				Pattern:     "*.mp3;*.wav;*.m4a;*.flac;*.ogg;*.wma;*.aac",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	})
}

// =============================================================================
// DeviceService — 音訊裝置枚举 (暴露给前端)
// =============================================================================

type DeviceService struct {
	ctx context.Context
	app *App
}

type DeviceInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

func (s *DeviceService) ListMicDevices() ([]DeviceInfo, error) {
	devices, err := audio.ListCaptureDevices(audio.BackendWASAPI)
	if err != nil {
		return nil, err
	}
	result := make([]DeviceInfo, 0, len(devices))
	for _, d := range devices {
		result = append(result, DeviceInfo{
			ID:   d,
			Name: d,
			Type: "mic",
		})
	}
	return result, nil
}

func (s *DeviceService) ListSpeakerDevices() ([]DeviceInfo, error) {
	devices, err := audio.ListLoopbackDevices(audio.BackendWASAPI)
	if err != nil {
		return nil, err
	}
	result := make([]DeviceInfo, 0, len(devices))
	for _, d := range devices {
		result = append(result, DeviceInfo{
			ID:   d,
			Name: d,
			Type: "speaker",
		})
	}
	return result, nil
}

// =============================================================================
// RecordService — 錄音控制和即時数据推送 (暴露给前端)
// =============================================================================

type RecordService struct {
	ctx         context.Context
	cancel      context.CancelFunc
	app         *App
	mu          sync.Mutex
	recorder    audio.Recorder
	transcriber *audio.LiveTranscriber
	running     bool
	outputPath  string
	startTime   time.Time
}

func (s *RecordService) StartRecording(micDevice, speakerDevice string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("已经在錄音中")
	}

	appConfig := s.app.configSnapshot()
	outputDir := appConfig.Output.OutputDir
	outputPath := audio.BuildRecordingPath(outputDir, time.Now())

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("建立輸出資料夾失敗: %w", err)
	}

	// Create live transcriber
	transcriber := audio.NewLiveTranscriber(audio.LiveTranscribeConfig{
		WhisperBinary: appConfig.ASR.WhisperBinary,
		ModelPath:     appConfig.ASR.ModelPath,
		Language:      appConfig.ASR.Language,
	})

	// Create recorder config
	cfg := audio.RecorderConfig{
		MicDevice:     micDevice,
		SpeakerDevice: speakerDevice,
		OutputPath:    outputPath,
		OnAudioData:   transcriber.Feed,
	}

	recorder, err := audio.NewRecorder(cfg)
	if err != nil {
		transcriber.Close()
		return fmt.Errorf("建立錄音器失敗: %w", err)
	}

	if err := recorder.Start(); err != nil {
		transcriber.Close()
		return fmt.Errorf("啟動錄音失敗: %w", err)
	}

	ctx, cancel := context.WithCancel(s.ctx)
	s.cancel = cancel
	s.recorder = recorder
	s.transcriber = transcriber
	s.running = true
	s.outputPath = outputPath
	s.startTime = time.Now()

	// Emit started event
	runtime.EventsEmit(s.ctx, "record:started", map[string]interface{}{
		"outputPath": outputPath,
	})

	// Start background goroutine for real-time data push
	go s.pushLoop(ctx)

	return nil
}

func (s *RecordService) StopRecording() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return "", fmt.Errorf("未在錄音")
	}

	// Cancel push loop
	if s.cancel != nil {
		s.cancel()
	}

	// Stop recorder
	if s.recorder != nil {
		s.recorder.Stop()
	}

	// Flush and close transcriber
	if s.transcriber != nil {
		s.transcriber.Close()
	}

	outputPath := s.outputPath
	s.running = false
	s.recorder = nil
	s.transcriber = nil

	// Emit stopped event
	runtime.EventsEmit(s.ctx, "record:stopped", map[string]interface{}{
		"outputPath": outputPath,
		"duration":   time.Since(s.startTime).Seconds(),
	})

	return outputPath, nil
}

func (s *RecordService) IsRecording() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *RecordService) GetElapsed() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return 0
	}
	return time.Since(s.startTime).Seconds()
}

// pushLoop runs in the background, pushing peak level and transcript data
func (s *RecordService) pushLoop(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.Lock()
			if !s.running || s.recorder == nil {
				s.mu.Unlock()
				return
			}

			// Push peak level
			peakLevel := s.recorder.PeakLevel()
			elapsed := time.Since(s.startTime).Seconds()
			runtime.EventsEmit(s.ctx, "record:peaklevel", map[string]interface{}{
				"level":   peakLevel,
				"elapsed": elapsed,
			})

			// Drain transcript results
			if s.transcriber != nil {
				for {
					select {
					case text, ok := <-s.transcriber.Results:
						if !ok {
							s.mu.Unlock()
							return
						}
						if text != "" && text != "__pending__" {
							runtime.EventsEmit(s.ctx, "record:transcript", map[string]interface{}{
								"text": text,
							})
						}
					default:
						goto doneTranscript
					}
				}
			doneTranscript:
			}

			s.mu.Unlock()
		}
	}
}

// =============================================================================
// PipelineService — 处理管线调用 (暴露给前端)
// =============================================================================

type PipelineService struct {
	ctx        context.Context
	app        *App
	mu         sync.Mutex
	currentCtx context.Context
	cancel     context.CancelFunc
	running    bool
}

func (s *PipelineService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *PipelineService) RunPipeline(audioPath string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("已有處理管線正在執行")
	}
	s.running = true
	ctx, cancel := context.WithCancel(s.ctx)
	s.currentCtx = ctx
	s.cancel = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.running = false
		s.currentCtx = nil
		s.cancel = nil
		s.mu.Unlock()
		cancel()
	}()

	// Create progress callback that emits Wails events
	progressFn := func(stage string) {
		runtime.EventsEmit(s.ctx, "pipeline:progress", map[string]interface{}{
			"stage": stage,
		})
	}

	appConfig := s.app.configSnapshot()
	state, err := workflow.RunMeetingPipeline(
		ctx,
		&appConfig,
		audioPath,
		appConfig.ASR.Language,
		progressFn,
	)

	if err != nil {
		if errors.Is(err, context.Canceled) {
			runtime.EventsEmit(s.ctx, "pipeline:cancelled", map[string]interface{}{
				"message": "已取消 AI 處理",
			})
			return nil
		}
		runtime.EventsEmit(s.ctx, "pipeline:error", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	// Store result in result service
	s.app.resultService.SetState(state)

	// Emit done with serialized state
	runtime.EventsEmit(s.ctx, "pipeline:done", stateToMap(state))

	return nil
}

func (s *PipelineService) CancelPipeline() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *PipelineService) RetrySummary() error {
	if s.app.resultService.state == nil {
		return fmt.Errorf("沒有可重試的會議紀錄")
	}

	runtime.EventsEmit(s.ctx, "pipeline:progress", map[string]interface{}{
		"stage": "LLM 整理",
	})

	appConfig := s.app.configSnapshot()
	newState, err := agent.RetrySummary(s.ctx, &appConfig, s.app.resultService.state)
	if err != nil {
		runtime.EventsEmit(s.ctx, "pipeline:error", map[string]interface{}{
			"error": err.Error(),
		})
		return err
	}

	s.app.resultService.SetState(newState)
	runtime.EventsEmit(s.ctx, "pipeline:done", stateToMap(newState))
	return nil
}

func (s *PipelineService) ImportAudioFile(filePath string) error {
	return s.RunPipeline(filePath)
}

// =============================================================================
// ResultService — 结果数据访问 (暴露给前端)
// =============================================================================

type ResultService struct {
	ctx   context.Context
	app   *App
	mu    sync.Mutex
	state *agent.MeetingState
}

func (s *ResultService) SetState(state *agent.MeetingState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = state
}

func (s *ResultService) GetState() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state == nil {
		return nil
	}
	return stateToMap(s.state)
}

// =============================================================================
// Helpers
// =============================================================================

func stateToMap(state *agent.MeetingState) map[string]interface{} {
	if state == nil {
		return nil
	}

	segments := make([]map[string]interface{}, 0, len(state.Segments))
	for _, seg := range state.Segments {
		segments = append(segments, map[string]interface{}{
			"start":      seg.Start,
			"end":        seg.End,
			"text":       seg.Text,
			"confidence": seg.Confidence,
		})
	}

	return map[string]interface{}{
		"audioFilePath":  state.AudioFilePath,
		"language":       state.Language,
		"rawTranscript":  state.RawTranscript,
		"meetingNotes":   state.MeetingNotes,
		"summaryContent": state.SummaryContent,
		"summaryError":   state.SummaryError,
		"summaryEnabled": state.SummaryEnabled,
		"audioDuration":  state.AudioDuration,
		"outputPath":     state.HTMLOutputPath,
		"htmlOutputPath": state.HTMLOutputPath,
		"transcriptPath": state.TranscriptPath,
		"markdownPath":   state.MarkdownPath,
		"segments":       segments,
		"error":          state.Error,
	}
}
