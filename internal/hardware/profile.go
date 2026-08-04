package hardware

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/mymeetily/mymeetily/internal/processutil"
)

type Profile struct {
	CPUThreads       int      `json:"cpuThreads"`
	Architecture     string   `json:"architecture"`
	GPUs             []string `json:"gpus"`
	HasNVIDIA        bool     `json:"hasNvidia"`
	RecommendedModel string   `json:"recommendedModel"`
	Recommendation   string   `json:"recommendation"`
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
					if strings.Contains(strings.ToLower(name), "nvidia") {
						profile.HasNVIDIA = true
					}
				}
			}
		}
	}

	switch {
	case profile.HasNVIDIA:
		profile.RecommendedModel = "large-v3-turbo-q5"
		profile.Recommendation = "偵測到 NVIDIA 顯示卡，建議 Large v3 Turbo Q5；若顯示記憶體不足可改用 Small。"
	case profile.CPUThreads <= 4:
		profile.RecommendedModel = "base"
		profile.Recommendation = "CPU 執行緒較少，建議 Base 以縮短等待時間。"
	default:
		profile.RecommendedModel = "small"
		profile.Recommendation = "未偵測到 NVIDIA GPU，建議 Small；大型模型在筆電 CPU 上可能耗時很久。"
	}
	return profile
}
