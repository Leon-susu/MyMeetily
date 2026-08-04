package main

import "testing"

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
