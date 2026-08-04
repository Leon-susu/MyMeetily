package hardware

import "testing"

func TestDetectGPUVendors(t *testing.T) {
	tests := []struct {
		name      string
		gpu       string
		hasNVIDIA bool
		hasAMD    bool
		hasIntel  bool
	}{
		{name: "nvidia", gpu: "NVIDIA GeForce RTX 4060", hasNVIDIA: true},
		{name: "amd", gpu: "AMD Radeon 8060S Graphics", hasAMD: true},
		{name: "radeon without amd", gpu: "Radeon RX 7800 XT", hasAMD: true},
		{name: "intel", gpu: "Intel(R) Arc(TM) Graphics", hasIntel: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nvidia, amd, intel := detectGPUVendor(tt.gpu)
			if nvidia != tt.hasNVIDIA || amd != tt.hasAMD || intel != tt.hasIntel {
				t.Fatalf("detectGPUVendor(%q) = (%v, %v, %v)", tt.gpu, nvidia, amd, intel)
			}
		})
	}
}
