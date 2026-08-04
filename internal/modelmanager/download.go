package modelmanager

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type ProgressFunc func(downloaded, total int64)

func Download(ctx context.Context, client *http.Client, sourceURL, destination string, progress ProgressFunc) error {
	if client == nil {
		client = http.DefaultClient
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("建立模型資料夾：%w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return fmt.Errorf("建立下載請求：%w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("下載模型：%w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("模型伺服器回傳 HTTP %d", response.StatusCode)
	}

	temporary := destination + ".part"
	_ = os.Remove(temporary)
	file, err := os.Create(temporary)
	if err != nil {
		return fmt.Errorf("建立暫存模型檔案：%w", err)
	}
	succeeded := false
	defer func() {
		_ = file.Close()
		if !succeeded {
			_ = os.Remove(temporary)
		}
	}()

	buffer := make([]byte, 256*1024)
	var downloaded int64
	for {
		n, readErr := response.Body.Read(buffer)
		if n > 0 {
			if _, err := file.Write(buffer[:n]); err != nil {
				return fmt.Errorf("寫入模型檔案：%w", err)
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, response.ContentLength)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("模型下載中斷：%w", readErr)
		}
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("關閉模型檔案：%w", err)
	}
	if downloaded == 0 {
		return fmt.Errorf("下載內容為空")
	}
	_ = os.Remove(destination)
	if err := os.Rename(temporary, destination); err != nil {
		return fmt.Errorf("完成模型安裝：%w", err)
	}
	succeeded = true
	return nil
}
