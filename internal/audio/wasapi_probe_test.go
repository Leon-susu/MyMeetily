//go:build windows

package audio

import (
	"os"
	"testing"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

// TestWASAPIInitializationProbe is opt-in because it needs a real Windows
// audio endpoint. It initializes the first active microphone but never starts
// capture, so no audio samples are recorded or retained.
func TestWASAPIInitializationProbe(t *testing.T) {
	if os.Getenv("MYMEETILY_AUDIO_PROBE") != "1" {
		t.Skip("set MYMEETILY_AUDIO_PROBE=1 to inspect the active microphone")
	}
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		t.Fatalf("COM initialization: %v", err)
	}
	defer ole.CoUninitialize()

	enumerator, err := newWASAPIDeviceEnumerator()
	if err != nil {
		t.Fatalf("device enumerator: %v", err)
	}
	defer enumerator.Release()

	device, descriptor, err := findWASAPIDeviceBySelection(enumerator, wca.ECapture, "")
	if err != nil {
		t.Fatalf("default capture device: %v", err)
	}
	defer device.Release()
	t.Logf("probing %s", descriptor.Label)

	client, format, latency, err := initializeWASAPIAudioClient(device, false)
	if err != nil {
		t.Fatalf("audio client initialization: %v", err)
	}
	defer client.Release()
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(format)))
	t.Logf("converted format=%dHz/%dch/%dbit tag=%d latency=%s", format.NSamplesPerSec, format.NChannels, format.WBitsPerSample, format.WFormatTag, latency)
}
