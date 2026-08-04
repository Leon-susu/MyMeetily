package modelmanager

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownload(t *testing.T) {
	payload := []byte("test-model-content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "models", "model.bin")
	var reported int64
	if err := Download(context.Background(), server.Client(), server.URL, destination, func(downloaded, _ int64) {
		reported = downloaded
	}); err != nil {
		t.Fatalf("download: %v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read destination: %v", err)
	}
	if string(got) != string(payload) || reported != int64(len(payload)) {
		t.Fatalf("unexpected download result: %q, progress=%d", got, reported)
	}
}

func TestDownloadRemovesPartialFileOnCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, 1024))
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	destination := filepath.Join(t.TempDir(), "model.bin")
	if err := Download(ctx, server.Client(), server.URL, destination, nil); err == nil {
		t.Fatal("expected cancellation error")
	}
	if _, err := os.Stat(destination + ".part"); !os.IsNotExist(err) {
		t.Fatal("partial file should be removed")
	}
}
