package asr

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/mymeetily/mymeetily/internal/processutil"
)

// Client runs whisper.cpp as a subprocess for speech recognition.
type Client struct {
	binary   string // path to whisper-cli executable
	model    string // path to GGUF model file
	language string // default language
}

// Segment represents a timed transcript fragment.
type Segment struct {
	Start      float64 `json:"start"`
	End        float64 `json:"end"`
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
}

// TranscribeResult holds the full ASR output.
type TranscribeResult struct {
	Text     string    `json:"text"`
	Language string    `json:"language"`
	Duration float64   `json:"duration"`
	Segments []Segment `json:"segments"`
}

// whisperJSON is the JSON output from `whisper-cli -oj`.
type whisperJSON struct {
	Text     string       `json:"text"`
	Language string       `json:"language"`
	Segments []whisperSeg `json:"segments"`
}

type whisperSeg struct {
	Start float64 `json:"t0"`
	End   float64 `json:"t1"`
	Text  string  `json:"text"`
}

// NewClient creates an ASR client backed by whisper.cpp.
func NewClient(binary, model, language string) *Client {
	return &Client{
		binary:   binary,
		model:    model,
		language: language,
	}
}

// Transcribe runs whisper-cli as a subprocess and parses the output.
func (c *Client) Transcribe(ctx context.Context, audioPath, language string) (*TranscribeResult, error) {
	return c.TranscribeWithProgress(ctx, audioPath, language, nil)
}

// TranscribeWithProgress processes long PCM WAV recordings in independent
// chunks and stores a checkpoint after each completed chunk. A retry can then
// continue from the last completed chunk instead of starting over.
func (c *Client) TranscribeWithProgress(ctx context.Context, audioPath, language string, progress func(current, total int)) (*TranscribeResult, error) {
	if language == "" {
		language = c.language
	}
	duration, err := wavDuration(audioPath)
	if err != nil || duration <= 300 {
		return c.transcribeRange(ctx, audioPath, language, 0, 0)
	}
	return c.transcribeChunks(ctx, audioPath, language, duration, progress)
}

func (c *Client) transcribeRange(ctx context.Context, audioPath, language string, offsetMS, durationMS int64) (*TranscribeResult, error) {

	args := []string{
		"-m", c.model,
		"-f", audioPath,
		"-l", language,
		"-t", strconv.Itoa(recommendedThreads()),
		"-oj", // JSON output
	}
	if offsetMS > 0 {
		args = append(args, "-ot", strconv.FormatInt(offsetMS, 10))
	}
	if durationMS > 0 {
		args = append(args, "-d", strconv.FormatInt(durationMS, 10))
	}
	cmd := exec.CommandContext(ctx, c.binary, args...)
	processutil.HideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("whisper-cli: %w\nstderr: %s", err, stderr.String())
	}

	raw := strings.TrimSpace(stdout.String())
	raw = stripMarkdownFence(raw)

	// Try JSON first, fall back to timestamp format.
	var wj whisperJSON
	if err := json.Unmarshal([]byte(raw), &wj); err == nil && wj.Text != "" {
		// JSON format (whisper.cpp -oj in some versions).
		result := &TranscribeResult{
			Text:     strings.TrimSpace(wj.Text),
			Language: wj.Language,
		}
		if result.Language == "" {
			result.Language = language
		}
		for _, s := range wj.Segments {
			result.Segments = append(result.Segments, Segment{
				Start:      s.Start / 100.0,
				End:        s.End / 100.0,
				Text:       strings.TrimSpace(s.Text),
				Confidence: 1.0,
			})
		}
		if len(result.Segments) > 0 {
			result.Duration = result.Segments[len(result.Segments)-1].End
		}
		return result, nil
	}

	// Timestamp format: [HH:MM:SS.mmm --> HH:MM:SS.mmm]  text
	lines := strings.Split(raw, "\n")
	result := &TranscribeResult{Language: language}
	var fullText strings.Builder
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		seg := parseTimestampLine(line)
		if seg.Text == "" {
			continue
		}
		result.Segments = append(result.Segments, seg)
		if fullText.Len() > 0 {
			fullText.WriteString(" ")
		}
		fullText.WriteString(seg.Text)
	}
	result.Text = fullText.String()
	if len(result.Segments) > 0 {
		result.Duration = result.Segments[len(result.Segments)-1].End
	}
	return result, nil
}

func recommendedThreads() int {
	threads := runtime.NumCPU()
	if threads > 12 {
		threads = 12
	} else if threads > 4 {
		threads-- // Keep one logical CPU available for the UI and system audio.
	}
	if threads < 1 {
		return 1
	}
	return threads
}

const asrChunkSeconds = 180.0

type chunkCheckpoint struct {
	AudioSize    int64               `json:"audioSize"`
	AudioModTime int64               `json:"audioModTime"`
	Model        string              `json:"model"`
	Language     string              `json:"language"`
	Duration     float64             `json:"duration"`
	Results      []*TranscribeResult `json:"results"`
}

func (c *Client) transcribeChunks(ctx context.Context, audioPath, language string, duration float64, progress func(current, total int)) (*TranscribeResult, error) {
	info, err := os.Stat(audioPath)
	if err != nil {
		return nil, err
	}
	checkpointPath := audioPath + ".asr-checkpoint.json"
	checkpoint := chunkCheckpoint{
		AudioSize: info.Size(), AudioModTime: info.ModTime().UnixNano(),
		Model: c.model, Language: language, Duration: duration,
	}
	if data, readErr := os.ReadFile(checkpointPath); readErr == nil {
		var saved chunkCheckpoint
		if json.Unmarshal(data, &saved) == nil && saved.AudioSize == checkpoint.AudioSize && saved.AudioModTime == checkpoint.AudioModTime && saved.Model == checkpoint.Model && saved.Language == checkpoint.Language {
			checkpoint.Results = saved.Results
		}
	}

	total := int((duration + asrChunkSeconds - 0.001) / asrChunkSeconds)
	if len(checkpoint.Results) > total {
		checkpoint.Results = nil
	}
	for index := len(checkpoint.Results); index < total; index++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if progress != nil {
			progress(index+1, total)
		}
		offset := float64(index) * asrChunkSeconds
		chunkDuration := asrChunkSeconds
		if remaining := duration - offset; remaining < chunkDuration {
			chunkDuration = remaining
		}
		result, err := c.transcribeRange(ctx, audioPath, language, int64(offset*1000), int64(chunkDuration*1000))
		if err != nil {
			return nil, fmt.Errorf("第 %d/%d 段轉錄失敗: %w", index+1, total, err)
		}
		adjustSegmentTimes(result.Segments, offset)
		checkpoint.Results = append(checkpoint.Results, result)
		data, _ := json.MarshalIndent(checkpoint, "", "  ")
		if err := os.WriteFile(checkpointPath, data, 0o600); err != nil {
			return nil, fmt.Errorf("儲存轉錄檢查點: %w", err)
		}
	}

	combined := combineChunkResults(checkpoint.Results, language, duration)
	_ = os.Remove(checkpointPath)
	return combined, nil
}

func adjustSegmentTimes(segments []Segment, offset float64) {
	if offset <= 0 || len(segments) == 0 || segments[0].Start >= offset-1 {
		return
	}
	for index := range segments {
		segments[index].Start += offset
		segments[index].End += offset
	}
}

func combineChunkResults(results []*TranscribeResult, language string, duration float64) *TranscribeResult {
	combined := &TranscribeResult{Language: language, Duration: duration}
	var text strings.Builder
	for _, result := range results {
		if result == nil {
			continue
		}
		if value := strings.TrimSpace(result.Text); value != "" {
			if text.Len() > 0 {
				text.WriteString("\n")
			}
			text.WriteString(value)
		}
		combined.Segments = append(combined.Segments, result.Segments...)
	}
	combined.Text = text.String()
	return combined
}

func wavDuration(path string) (float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	header := make([]byte, 12)
	if _, err := io.ReadFull(file, header); err != nil {
		return 0, err
	}
	if string(header[:4]) != "RIFF" || string(header[8:12]) != "WAVE" {
		return 0, fmt.Errorf("不是 PCM WAV 檔案")
	}
	var byteRate uint32
	var dataSize uint32
	for {
		chunk := make([]byte, 8)
		if _, err := io.ReadFull(file, chunk); err != nil {
			return 0, err
		}
		size := binary.LittleEndian.Uint32(chunk[4:8])
		switch string(chunk[:4]) {
		case "fmt ":
			format := make([]byte, size)
			if _, err := io.ReadFull(file, format); err != nil {
				return 0, err
			}
			if len(format) < 12 {
				return 0, fmt.Errorf("WAV fmt 區塊不完整")
			}
			byteRate = binary.LittleEndian.Uint32(format[8:12])
			if dataSize > 0 && byteRate > 0 {
				return float64(dataSize) / float64(byteRate), nil
			}
		case "data":
			dataSize = size
			if byteRate > 0 {
				return float64(dataSize) / float64(byteRate), nil
			}
			if _, err := file.Seek(int64(size), io.SeekCurrent); err != nil {
				return 0, err
			}
		default:
			if _, err := file.Seek(int64(size), io.SeekCurrent); err != nil {
				return 0, err
			}
		}
		if size%2 == 1 {
			_, _ = file.Seek(1, io.SeekCurrent)
		}
	}
}

// parseTimestampLine extracts a Segment from a whisper timestamp line like
// "[00:00:00.000 --> 00:00:02.500]  some text".
func parseTimestampLine(line string) Segment {
	var seg Segment
	// Find "]  " or "] " separator.
	idx := strings.Index(line, "]  ")
	if idx < 0 {
		idx = strings.Index(line, "] ")
	}
	if idx < 0 {
		return seg
	}
	text := strings.TrimSpace(line[idx+1:])
	// Trim the bracket prefix.
	if strings.HasPrefix(text, " ") {
		text = strings.TrimSpace(text)
	}
	seg.Text = text

	// Parse timestamps [HH:MM:SS.mmm --> HH:MM:SS.mmm]
	ts := line[1:idx] // "HH:MM:SS.mmm --> HH:MM:SS.mmm"
	parts := strings.Split(ts, " --> ")
	if len(parts) == 2 {
		seg.Start = parseTimestamp(parts[0])
		seg.End = parseTimestamp(parts[1])
	}
	seg.Confidence = 1.0
	return seg
}

// parseTimestamp converts "HH:MM:SS.mmm" to seconds as float64.
func parseTimestamp(s string) float64 {
	var h, m int
	var sec float64
	fmt.Sscanf(s, "%d:%d:%f", &h, &m, &sec)
	return float64(h*3600+m*60) + sec
}

// stripMarkdownFence removes ```json ... ``` wrapping if present.
func stripMarkdownFence(s string) string {
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
