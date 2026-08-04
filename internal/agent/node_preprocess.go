package agent

import (
	"context"
	"log/slog"
)

func (n *Nodes) PreprocessAudio(ctx context.Context, state *MeetingState) (*MeetingState, error) {
	if state.ProgressFn != nil {
		state.ProgressFn("音訊預處理")
	}
	slog.Info("preprocessing audio", "file", state.AudioFilePath)

	state.WavFilePath = state.AudioFilePath
	return state, nil
}
