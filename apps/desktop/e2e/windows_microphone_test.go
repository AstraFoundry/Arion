//go:build arion_e2e && windows

package e2e

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"arion.local/core/contracts"
	"arion.local/desktop/infrastructure/audio"
)

// This opt-in device check retains only counters, never microphone samples.
func TestWindowsMicrophoneLifecycle(t *testing.T) {
	if os.Getenv("ARION_E2E_MICROPHONE") != "1" {
		t.Skip("set ARION_E2E_MICROPHONE=1 for a short real capture")
	}
	devices, err := audio.Devices()
	if err != nil || len(devices) == 0 {
		t.Fatalf("capture endpoints: count=%d error=%v", len(devices), err)
	}
	mic := audio.NewMicrophone(nil)
	for cycle := 0; cycle < 2; cycle++ {
		id := contracts.Identity{SessionID: "windows-microphone", Generation: uint64(cycle + 1)}
		var mu sync.Mutex
		var frames, samples uint64
		var invalid bool
		stream, err := mic.Start(context.Background(), id, func(frame contracts.Frame) {
			mu.Lock()
			defer mu.Unlock()
			invalid = invalid || frame.Identity != id || frame.SampleRate != 16000 || frame.Channels != 1 || frame.Discontinuity || frame.Sequence != frames || frame.SampleOffset != samples
			frames++
			samples += uint64(len(frame.Samples))
		})
		if err != nil {
			t.Fatalf("open capture cycle %d: %v", cycle, err)
		}
		t.Cleanup(func() { _ = stream.Stop() })
		time.Sleep(time.Second)
		if err := stream.Stop(); err != nil {
			t.Fatal(err)
		}
		mu.Lock()
		stoppedFrames, capturedSamples, bad := frames, samples, invalid
		mu.Unlock()
		if bad || stoppedFrames == 0 || capturedSamples < 8000 {
			t.Fatalf("invalid capture: frames=%d samples=%d invalid=%v", stoppedFrames, capturedSamples, bad)
		}
		time.Sleep(100 * time.Millisecond)
		mu.Lock()
		lateFrames := frames != stoppedFrames
		mu.Unlock()
		if lateFrames {
			t.Fatal("microphone callbacks continued after Stop returned")
		}
		t.Logf("cycle=%d endpoints=%d frames=%d samples=%d; stopped without late callbacks", cycle, len(devices), stoppedFrames, capturedSamples)
	}
}
