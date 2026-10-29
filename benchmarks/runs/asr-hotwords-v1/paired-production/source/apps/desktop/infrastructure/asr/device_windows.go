package asr

import (
	"os"
	"path/filepath"
	"syscall"
)

// Vulkan is selected before inference only when the system loader is present.
// A failure to load the selected runtime is surfaced; it never retries on CPU.
func workerName() string {
	loader, err := syscall.LoadDLL(filepath.Join(os.Getenv("SystemRoot"), "System32", "vulkan-1.dll"))
	if err == nil {
		_ = loader.Release()
		return "speech-vulkan.exe"
	}
	return "speech.exe"
}
