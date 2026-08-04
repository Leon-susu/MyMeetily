package workflow

import (
	"context"

	"github.com/mymeetily/mymeetily/internal/config"
	"github.com/mymeetily/mymeetily/internal/ollama"
)

type StartupStatus struct {
	SummaryEnabled bool
	SummaryMessage string
}

func DetectSummaryAvailability(ctx context.Context, cfg *config.Config, summaryModelPath, summaryModelURL string) (StartupStatus, error) {
	client := ollama.NewClient(cfg.Ollama.Endpoint, cfg.Ollama.Model, cfg.Ollama.Temperature, cfg.Ollama.MaxTokens)
	if err := client.Ping(ctx); err != nil {
		return StartupStatus{
			SummaryEnabled: false,
			SummaryMessage: "未偵測到本機 Ollama 正在執行，本次將繼續提供錄音與轉錄，並直接輸出報告。",
		}, nil
	}

	if err := client.EnsureModelAvailable(ctx, summaryModelPath, summaryModelURL, false); err != nil {
		return StartupStatus{}, err
	}

	return StartupStatus{
		SummaryEnabled: true,
		SummaryMessage: "Ollama 已就緒，將啟用自動摘要。",
	}, nil
}
