package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteHTMLWithTranscriptSegments(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "report.html")

	err := WriteHTML("# 會議紀要\n\n摘要内容", "第一句\n第二句", []TranscriptSegment{
		{Start: 5, End: 12, Text: "第一句"},
		{Start: 62, End: 75, Text: "第二句"},
	}, outputPath)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	html := string(data)

	for _, want := range []string{
		"<summary>原文轉寫</summary>",
		"00:05 - 00:12",
		"01:02 - 01:15",
		"第一句",
		"第二句",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected HTML to contain %q", want)
		}
	}
}

func TestWriteHTMLFallsBackToRawTranscript(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "report.html")

	err := WriteHTML("# 會議紀要\n\n未啟用自動摘要，僅輸出原文轉錄", "完整原文內容", nil, outputPath)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	html := string(data)

	if !strings.Contains(html, "完整原文內容") {
		t.Fatalf("expected HTML to include raw transcript")
	}
	if !strings.Contains(html, "transcript-raw") {
		t.Fatalf("expected HTML to use raw transcript fallback")
	}
}
