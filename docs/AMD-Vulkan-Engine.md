# AMD／Vulkan 引擎建置與發布

MyMeetily 的 AMD GPU 加速使用 whisper.cpp 的 Vulkan backend。使用者端只需要支援 Vulkan 的 AMD 顯示卡驅動；Vulkan SDK 僅供建置引擎使用，不應安裝到每一台同事電腦。

## 安全原則

- whisper.cpp 固定使用 `v1.8.5`，避免上游變動造成不可重現的產物。
- LunarG Vulkan SDK 固定使用 `1.4.350.0`，下載後驗證官方 SHA-256。
- 建置包包含每個執行檔與 DLL 的 SHA-256 manifest。
- GitHub Actions 預設只產生保留 14 天的測試 artifact，不會自動公開發布。
- 完成 AMD 實機轉錄測試後，才以 `publish_release=true` 建立 Release。
- 正式提供公司電腦前仍應進行程式碼簽署或取得公司 IT 白名單。

## GitHub Actions 建置

在 GitHub repository 開啟 `Actions`，選擇 `Build Windows Vulkan Engine`，按 `Run workflow`：

1. 第一次保持 `publish_release=false`。
2. 下載 `whisper-vulkan-v1.8.5-windows-x64` artifact。
3. 核對 `.sha256`，再於 AMD 測試機執行 `whisper-cli.exe --help` 與實際短音檔轉錄。
4. 確認輸出包含 Vulkan GPU backend 後，再重新執行 workflow 並選擇發布。

下載 artifact 並解壓最外層後，可使用驗證腳本：

```powershell
.\scripts\Test-WhisperVulkanPackage.ps1 `
    -ArchivePath .\whisper-vulkan-1.8.5-bin-x64.zip `
    -ModelPath .\llmmodels\whisper\ggml-small.bin `
    -AudioPath .\test.wav
```

## 本機建置

需要 Git、CMake、Visual Studio 2022 C++ Build Tools 與網路連線。在 PowerShell 執行：

```powershell
.\scripts\Build-WhisperVulkan.ps1
```

腳本會在暫存資料夾下載並驗證 Vulkan SDK、取得固定版 whisper.cpp、以 `GGML_VULKAN=ON` 建置，最後將壓縮包與 SHA-256 寫入 `dist\vulkan-engine`。

## 尚未完成

在測試 artifact 通過 AMD 實機驗證以前，一鍵安裝器仍會讓 AMD 電腦使用 CPU 引擎。確認 Release URL 與 SHA-256 後，下一步才會把 Vulkan 自動下載及失敗時退回 CPU 接入安裝器。
