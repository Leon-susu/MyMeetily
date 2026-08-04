package agent

import (
	"context"
	"fmt"
	"log/slog"
)

func (n *Nodes) Summarize(ctx context.Context, state *MeetingState) (*MeetingState, error) {
	if state.ProgressFn != nil {
		state.ProgressFn("LLM 整理")
	}
	slog.Info("summarizing with Ollama", "endpoint", n.cfg.Ollama.Endpoint, "model", n.cfg.Ollama.Model)

	resp, err := n.ollama.Summarize(ctx, state.RawTranscript)
	if err != nil {
		state.SummaryError = err.Error()
		state.SummaryContent = fallbackMeetingNotes(state.SummaryError)
		state.MeetingNotes = state.SummaryContent
		slog.Warn("summarization failed, falling back to transcript-only output", "error", err)
		return state, nil
	}

	state.SummaryEnabled = true
	state.SummaryContent = resp
	state.MeetingNotes = resp
	slog.Info("summarization done", "notes_len", len(state.MeetingNotes))
	return state, nil
}

func fallbackMeetingNotes(summaryErr string) string {
	return fmt.Sprintf(`## 摘要產生失敗

本次未能完成自动總結，已保留完整轉寫内容。
- 可能原因：Ollama 服務未啟動、模型無法使用，或本次請求逾時
- 错误信息：%s
- 建議：檢查 Ollama 狀態後重試；目前 HTML 報告下方仍可展開查看完整原文
`, summaryErr)
}
