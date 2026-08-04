[CmdletBinding()]
param(
    [Parameter(Mandatory)][string]$ArchivePath,
    [string]$ModelPath,
    [string]$AudioPath
)

$ErrorActionPreference = "Stop"
$archive = Get-Item -LiteralPath $ArchivePath -ErrorAction Stop
$checksumPath = "$($archive.FullName).sha256"
if (-not (Test-Path -LiteralPath $checksumPath -PathType Leaf)) {
    throw "找不到 SHA-256 檔案：$checksumPath"
}

$expectedHash = ((Get-Content -LiteralPath $checksumPath -Raw -Encoding ascii).Trim() -split '\s+')[0].ToLowerInvariant()
$actualHash = (Get-FileHash -LiteralPath $archive.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actualHash -ne $expectedHash) {
    throw "引擎壓縮包 SHA-256 不符。預期 $expectedHash，實際 $actualHash。"
}

$taskRoot = Join-Path ([IO.Path]::GetTempPath()) ("mymeetily-vulkan-test-" + [guid]::NewGuid().ToString("N"))
try {
    New-Item -ItemType Directory -Path $taskRoot -Force | Out-Null
    Expand-Archive -LiteralPath $archive.FullName -DestinationPath $taskRoot

    $requiredFiles = @(
        "whisper-cli.exe",
        "whisper.dll",
        "ggml.dll",
        "ggml-base.dll",
        "ggml-cpu.dll",
        "ggml-vulkan.dll",
        "engine-manifest.json"
    )
    foreach ($fileName in $requiredFiles) {
        if (-not (Test-Path -LiteralPath (Join-Path $taskRoot $fileName) -PathType Leaf)) {
            throw "測試包不完整：找不到 $fileName。"
        }
    }

    $manifest = Get-Content -LiteralPath (Join-Path $taskRoot "engine-manifest.json") -Raw -Encoding utf8 | ConvertFrom-Json
    if ($manifest.backend -ne "vulkan" -or $manifest.architecture -ne "windows/amd64") {
        throw "引擎 manifest 不符合預期的 Vulkan windows/amd64。"
    }

    $whisperCli = Join-Path $taskRoot "whisper-cli.exe"
    $previousErrorAction = $ErrorActionPreference
    try {
        # whisper.cpp writes backend discovery to stderr even on success.
        # Windows PowerShell 5 turns redirected native stderr into ErrorRecord
        # objects when Stop is active, so capture it under Continue explicitly.
        $ErrorActionPreference = "Continue"
        $helpOutput = (& $whisperCli --help 2>&1) -join "`n"
        $helpExitCode = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $previousErrorAction
    }
    if ($helpExitCode -ne 0) {
        throw "whisper-cli 啟動失敗（結束碼 $helpExitCode）：`n$helpOutput"
    }
    Write-Host "基本啟動驗證通過。" -ForegroundColor Green

    if (($ModelPath -and -not $AudioPath) -or ($AudioPath -and -not $ModelPath)) {
        throw "實際 GPU 驗證必須同時指定 -ModelPath 與 -AudioPath。"
    }
    if ($ModelPath -and $AudioPath) {
        $model = Get-Item -LiteralPath $ModelPath -ErrorAction Stop
        $audio = Get-Item -LiteralPath $AudioPath -ErrorAction Stop
        $outputBase = Join-Path $taskRoot "vulkan-probe"
        $previousErrorAction = $ErrorActionPreference
        try {
            $ErrorActionPreference = "Continue"
            $probeOutput = (& $whisperCli -m $model.FullName -f $audio.FullName -l zh -bs 1 -bo 1 -otxt -of $outputBase 2>&1) -join "`n"
            $probeExitCode = $LASTEXITCODE
        } finally {
            $ErrorActionPreference = $previousErrorAction
        }
        if ($probeExitCode -ne 0) {
            throw "AMD Vulkan 實際轉錄失敗（結束碼 $probeExitCode）：`n$probeOutput"
        }
        if ($probeOutput -match "no GPU found") {
            throw "引擎啟動成功，但沒有找到 GPU：`n$probeOutput"
        }
        if ($probeOutput -notmatch "(?i)vulkan") {
            throw "轉錄完成，但記錄中沒有 Vulkan backend 證據，暫不接受此建置包：`n$probeOutput"
        }
        Write-Host "AMD Vulkan 實際轉錄驗證通過。" -ForegroundColor Green
    } else {
        Write-Host "尚未執行實際 GPU 轉錄；請加上 -ModelPath 與 -AudioPath 完成驗收。" -ForegroundColor Yellow
    }
} finally {
    $resolvedTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    $resolvedTask = [IO.Path]::GetFullPath($taskRoot)
    if ($resolvedTask.StartsWith($resolvedTemp, [StringComparison]::OrdinalIgnoreCase) -and (Test-Path -LiteralPath $resolvedTask)) {
        Remove-Item -LiteralPath $resolvedTask -Recurse -Force -ErrorAction SilentlyContinue
    }
}
