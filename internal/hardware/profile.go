package hardware

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mymeetily/mymeetily/internal/processutil"
)

type Profile struct {
	CPUThreads        int      `json:"cpuThreads"`
	Architecture      string   `json:"architecture"`
	GPUs              []string `json:"gpus"`
	HasNVIDIA         bool     `json:"hasNvidia"`
	HasAMD            bool     `json:"hasAmd"`
	HasIntel          bool     `json:"hasIntel"`
	VulkanRuntime     bool     `json:"vulkanRuntime"`
	RecommendedEngine string   `json:"recommendedEngine"`
	RecommendedModel  string   `json:"recommendedModel"`
	Recommendation    string   `json:"recommendation"`
	DriverAdvice      string   `json:"driverAdvice"`
	DriverURL         string   `json:"driverUrl"`
}

func Detect() Profile {
	profile := Profile{CPUThreads: runtime.NumCPU(), Architecture: runtime.GOARCH}
	if runtime.GOOS == "windows" {
		command := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "(Get-CimInstance Win32_VideoController | Select-Object -ExpandProperty Name) -join [Environment]::NewLine")
		processutil.HideWindow(command)
		if output, err := command.Output(); err == nil {
			for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
				name := strings.TrimSpace(strings.TrimSuffix(line, "\r"))
				if name != "" {
					profile.GPUs = append(profile.GPUs, name)
					isNVIDIA, isAMD, isIntel := detectGPUVendor(name)
					profile.HasNVIDIA = profile.HasNVIDIA || isNVIDIA
					profile.HasAMD = profile.HasAMD || isAMD
					profile.HasIntel = profile.HasIntel || isIntel
				}
			}
		}
		windowsDir := os.Getenv("WINDIR")
		if windowsDir == "" {
			windowsDir = `C:\Windows`
		}
		_, err := os.Stat(filepath.Join(windowsDir, "System32", "vulkan-1.dll"))
		profile.VulkanRuntime = err == nil
	}

	switch {
	case profile.HasNVIDIA:
		profile.RecommendedEngine = "cuda-12.4"
		profile.RecommendedModel = "large-v3-turbo-q5"
		profile.Recommendation = "偵測到 NVIDIA 顯示卡，建議 Large v3 Turbo Q5；若顯示記憶體不足可改用 Small。"
		profile.DriverAdvice = "若 CUDA 引擎找不到 GPU，請更新 NVIDIA 顯示卡驅動。"
		profile.DriverURL = "https://www.nvidia.com/Download/index.aspx"
	case profile.HasAMD:
		profile.RecommendedEngine = "vulkan"
		profile.RecommendedModel = "small"
		profile.Recommendation = "偵測到 AMD Radeon，建議使用 Vulkan 引擎與 Small 模型；若 Vulkan runtime 不可用，安裝器會安全改用 CPU 引擎。"
		if profile.VulkanRuntime {
			profile.DriverAdvice = "已偵測到 Vulkan runtime。AMD 筆電請優先使用電腦品牌官網提供的顯示卡驅動；若原廠未提供新版，再使用 AMD Software: Adrenalin Edition。"
		} else {
			profile.DriverAdvice = "未偵測到 Vulkan runtime。AMD 筆電請先安裝電腦品牌官網提供的顯示卡驅動；必要時再使用 AMD Software: Adrenalin Edition。"
		}
		profile.DriverURL = "https://www.amd.com/en/support/download/drivers.html"
	case profile.CPUThreads <= 4:
		profile.RecommendedEngine = "cpu"
		profile.RecommendedModel = "base"
		profile.Recommendation = "CPU 執行緒較少，建議 Base 以縮短等待時間。"
	default:
		profile.RecommendedEngine = "cpu"
		profile.RecommendedModel = "small"
		profile.Recommendation = "目前建議使用 CPU 引擎與 Small 模型；大型模型在筆電 CPU 上可能耗時很久。"
		if profile.HasIntel {
			profile.DriverAdvice = "偵測到 Intel 顯示晶片；目前先使用 CPU 引擎，後續可評估 Vulkan 加速。"
		}
	}
	return profile
}

func detectGPUVendor(name string) (nvidia, amd, intel bool) {
	lowerName := strings.ToLower(name)
	return strings.Contains(lowerName, "nvidia"),
		strings.Contains(lowerName, "amd") || strings.Contains(lowerName, "radeon"),
		strings.Contains(lowerName, "intel")
}
