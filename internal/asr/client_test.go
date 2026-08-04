package asr

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestWAVDuration(t *testing.T) {
	const sampleRate = 16000
	const seconds = 2
	dataSize := sampleRate * seconds * 2
	buffer := &bytes.Buffer{}
	buffer.WriteString("RIFF")
	_ = binary.Write(buffer, binary.LittleEndian, uint32(36+dataSize))
	buffer.WriteString("WAVEfmt ")
	_ = binary.Write(buffer, binary.LittleEndian, uint32(16))
	_ = binary.Write(buffer, binary.LittleEndian, uint16(1))
	_ = binary.Write(buffer, binary.LittleEndian, uint16(1))
	_ = binary.Write(buffer, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buffer, binary.LittleEndian, uint32(sampleRate*2))
	_ = binary.Write(buffer, binary.LittleEndian, uint16(2))
	_ = binary.Write(buffer, binary.LittleEndian, uint16(16))
	buffer.WriteString("data")
	_ = binary.Write(buffer, binary.LittleEndian, uint32(dataSize))
	buffer.Write(make([]byte, dataSize))

	path := filepath.Join(t.TempDir(), "sample.wav")
	if err := os.WriteFile(path, buffer.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	duration, err := wavDuration(path)
	if err != nil {
		t.Fatalf("wavDuration: %v", err)
	}
	if duration != seconds {
		t.Fatalf("duration=%v, want %v", duration, seconds)
	}
}

func TestAdjustAndCombineChunkResults(t *testing.T) {
	second := []Segment{{Start: 0, End: 2, Text: "第二段", Confidence: 1}}
	adjustSegmentTimes(second, 180)
	if second[0].Start != 180 || second[0].End != 182 {
		t.Fatalf("segment was not offset: %+v", second[0])
	}
	combined := combineChunkResults([]*TranscribeResult{
		{Text: "第一段", Segments: []Segment{{Start: 1, End: 2, Text: "第一段"}}},
		{Text: "第二段", Segments: second},
	}, "zh", 360)
	if combined.Text != "第一段\n第二段" || len(combined.Segments) != 2 || combined.Duration != 360 {
		t.Fatalf("unexpected combined result: %+v", combined)
	}
}

func TestRecommendedThreadsIsBounded(t *testing.T) {
	threads := recommendedThreads()
	if threads < 1 || threads > 12 {
		t.Fatalf("recommendedThreads=%d, want 1..12", threads)
	}
}
