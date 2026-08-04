package main

import (
	"path/filepath"
	"testing"
)

func TestHistoryMetadata(t *testing.T) {
	source, summary := historyMetadata(`# 會議紀要

> 源檔案：weekly.wav
> 音訊時長：600 秒

---

## 總結

本週完成第一階段，下一步進行驗收。
`)
	if source != "weekly.wav" {
		t.Fatalf("source=%q", source)
	}
	if summary != "本週完成第一階段，下一步進行驗收。" {
		t.Fatalf("summary=%q", summary)
	}
}

func TestResolveHistoryTarget(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "meeting", "report.html")
	resolved, err := resolveHistoryTarget(root, inside)
	if err != nil {
		t.Fatalf("inside path rejected: %v", err)
	}
	if resolved != inside {
		t.Fatalf("resolved=%q, want %q", resolved, inside)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.html")
	if _, err := resolveHistoryTarget(root, outside); err == nil {
		t.Fatal("outside path was accepted")
	}
}
