[CmdletBinding()]
param(
    [string]$WhisperVersion = "v1.8.5",
    [string]$VulkanSdkVersion = "1.4.350.0",
    [string]$VulkanSdkSha256 = "855b27ba05d2d8119c5114c5d4ff870ca38f2c632b11e1bb9923b9b7e6ecfe7b",
    [string]$OutputDirectory = (Join-Path $PSScriptRoot "..\dist\vulkan-engine")
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

function Invoke-Checked {
    param(
        [Parameter(Mandatory)][string]$FilePath,
        [Parameter(ValueFromRemainingArguments)][string[]]$Arguments
    )
    Write-Host "> $FilePath $($Arguments -join ' ')"
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "命令失敗（結束碼 $LASTEXITCODE）：$FilePath $($Arguments -join ' ')"
    }
}

function Enable-MsvcEnvironment {
    if (Get-Command cl.exe -ErrorAction SilentlyContinue) {
        return
    }

    $vswhereCandidates = @(
        (Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"),
        (Join-Path $env:ProgramFiles "Microsoft Visual Studio\Installer\vswhere.exe")
    ) | Where-Object { $_ -and (Test-Path -LiteralPath $_ -PathType Leaf) }
    $vswhere = $vswhereCandidates | Select-Object -First 1
    if (-not $vswhere) {
        throw "找不到 vswhere.exe。請安裝 Visual Studio 2022 C++ Build Tools。"
    }

    $installationPath = (& $vswhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath).Trim()
    if (-not $installationPath) {
        throw "找不到包含 MSVC x64 工具的 Visual Studio 2022。"
    }
    $devCmd = Join-Path $installationPath "Common7\Tools\VsDevCmd.bat"
    if (-not (Test-Path -LiteralPath $devCmd -PathType Leaf)) {
        throw "找不到 Visual Studio 開發環境腳本：$devCmd"
    }

    Write-Host "載入 Visual Studio 2022 x64 C++ 建置環境..."
    $environmentLines = & cmd.exe /s /c "`"$devCmd`" -no_logo -arch=x64 && set"
    if ($LASTEXITCODE -ne 0) {
        throw "Visual Studio 開發環境初始化失敗（結束碼 $LASTEXITCODE）。"
    }
    foreach ($line in $environmentLines) {
        if ($line -match '^([^=][^=]*)=(.*)$') {
            [Environment]::SetEnvironmentVariable($Matches[1], $Matches[2], "Process")
        }
    }
    if (-not (Get-Command cl.exe -ErrorAction SilentlyContinue)) {
        throw "Visual Studio 環境載入後仍找不到 cl.exe。"
    }
}

foreach ($commandName in @("git", "cmake")) {
    if (-not (Get-Command $commandName -ErrorAction SilentlyContinue)) {
        throw "找不到 $commandName。請先安裝 Git、CMake 與 Visual Studio 2022 C++ Build Tools。"
    }
}
Enable-MsvcEnvironment

$outputRoot = [IO.Path]::GetFullPath($OutputDirectory)
New-Item -ItemType Directory -Path $outputRoot -Force | Out-Null

$taskRoot = Join-Path ([IO.Path]::GetTempPath()) ("mymeetily-vulkan-" + [guid]::NewGuid().ToString("N"))
$sdkRoot = Join-Path $taskRoot "VulkanSDK"
$sourceRoot = Join-Path $taskRoot "whisper.cpp"
$buildRoot = Join-Path $sourceRoot "build-vulkan"
$packageRoot = Join-Path $taskRoot "package"
$sdkInstaller = Join-Path $taskRoot "vulkan-sdk.exe"
$archiveName = "whisper-vulkan-$($WhisperVersion.TrimStart('v'))-bin-x64.zip"
$archivePath = Join-Path $outputRoot $archiveName
$checksumPath = "$archivePath.sha256"

try {
    New-Item -ItemType Directory -Path $taskRoot, $packageRoot -Force | Out-Null

    $sdkUrl = "https://sdk.lunarg.com/sdk/download/$VulkanSdkVersion/windows/vulkan_sdk.exe"
    Write-Host "下載 LunarG Vulkan SDK $VulkanSdkVersion..."
    Invoke-WebRequest -Uri $sdkUrl -OutFile $sdkInstaller -UseBasicParsing
    $actualSdkHash = (Get-FileHash -LiteralPath $sdkInstaller -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualSdkHash -ne $VulkanSdkSha256.ToLowerInvariant()) {
        throw "Vulkan SDK SHA-256 不符。預期 $VulkanSdkSha256，實際 $actualSdkHash。"
    }

    Write-Host "以 copy-only 模式準備 Vulkan SDK..."
    $sdkProcess = Start-Process -FilePath $sdkInstaller -ArgumentList @(
        "--root", $sdkRoot,
        "--accept-licenses", "--default-answer", "--confirm-command",
        "install", "copy_only=1"
    ) -Wait -PassThru -NoNewWindow
    if ($sdkProcess.ExitCode -ne 0) {
        throw "Vulkan SDK 安裝器失敗（結束碼 $($sdkProcess.ExitCode)）。"
    }
    if (-not (Test-Path -LiteralPath (Join-Path $sdkRoot "Lib\vulkan-1.lib") -PathType Leaf)) {
        throw "Vulkan SDK 檔案不完整：找不到 Lib\vulkan-1.lib。"
    }

    $env:VULKAN_SDK = $sdkRoot
    $env:Path = "$(Join-Path $sdkRoot 'Bin');$env:Path"

    Invoke-Checked -FilePath "git" -Arguments @(
        "clone", "--depth", "1", "--branch", $WhisperVersion,
        "https://github.com/ggml-org/whisper.cpp.git", $sourceRoot
    )
    Invoke-Checked -FilePath "cmake" -Arguments @(
        "-S", $sourceRoot,
        "-B", $buildRoot,
        "-A", "x64",
        "-DCMAKE_BUILD_TYPE=Release",
        "-DBUILD_SHARED_LIBS=ON",
        "-DGGML_VULKAN=ON",
        "-DGGML_NATIVE=OFF",
        "-DGGML_BMI2=OFF",
        "-DWHISPER_BUILD_TESTS=OFF",
        "-DWHISPER_BUILD_EXAMPLES=ON"
    )
    Invoke-Checked -FilePath "cmake" -Arguments @("--build", $buildRoot, "--config", "Release", "--parallel")

    $binaryRoot = Join-Path $buildRoot "bin\Release"
    $requiredFiles = @(
        "whisper-cli.exe",
        "whisper.dll",
        "ggml.dll",
        "ggml-base.dll",
        "ggml-cpu.dll",
        "ggml-vulkan.dll"
    )
    foreach ($fileName in $requiredFiles) {
        $sourcePath = Join-Path $binaryRoot $fileName
        if (-not (Test-Path -LiteralPath $sourcePath -PathType Leaf)) {
            throw "Vulkan 引擎建置不完整：找不到 $fileName。"
        }
        Copy-Item -LiteralPath $sourcePath -Destination (Join-Path $packageRoot $fileName)
    }
    Copy-Item -LiteralPath (Join-Path $sourceRoot "LICENSE") -Destination (Join-Path $packageRoot "LICENSE-whisper.cpp")

    $fileEntries = foreach ($file in Get-ChildItem -LiteralPath $packageRoot -File | Sort-Object Name) {
        [ordered]@{
            name = $file.Name
            size = $file.Length
            sha256 = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        }
    }
    $manifest = [ordered]@{
        engine = "whisper.cpp"
        backend = "vulkan"
        architecture = "windows/amd64"
        sourceRepository = "https://github.com/ggml-org/whisper.cpp"
        sourceVersion = $WhisperVersion
        vulkanSdkVersion = $VulkanSdkVersion
        builtAtUtc = [DateTime]::UtcNow.ToString("o")
        files = @($fileEntries)
    }
    $manifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $packageRoot "engine-manifest.json") -Encoding utf8

    if (Test-Path -LiteralPath $archivePath) {
        Remove-Item -LiteralPath $archivePath -Force
    }
    Compress-Archive -Path (Join-Path $packageRoot "*") -DestinationPath $archivePath -CompressionLevel Optimal
    $archiveHash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    "$archiveHash  $archiveName" | Set-Content -LiteralPath $checksumPath -Encoding ascii

    Write-Host "Vulkan 引擎建置完成：$archivePath" -ForegroundColor Green
    Write-Host "SHA-256：$archiveHash" -ForegroundColor Green
} finally {
    $resolvedTemp = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
    $resolvedTask = [IO.Path]::GetFullPath($taskRoot)
    if ($resolvedTask.StartsWith($resolvedTemp, [StringComparison]::OrdinalIgnoreCase) -and (Test-Path -LiteralPath $resolvedTask)) {
        Remove-Item -LiteralPath $resolvedTask -Recurse -Force -ErrorAction SilentlyContinue
    }
}
