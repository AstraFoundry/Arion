//go:build windows

package windows

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"arion.local/core/domain"
)

// Window text recognition: the foreground window is captured with
// PrintWindow and recognized by Windows.Media.Ocr through a fixed
// PowerShell WinRT script. This adapter is compile-checked only; it has not
// run on Windows (ADR 013).

var (
	gdi32                      = syscall.NewLazyDLL("gdi32.dll")
	procGetWindowRect          = user32.NewProc("GetWindowRect")
	procGetWindowDC            = user32.NewProc("GetWindowDC")
	procReleaseDC              = user32.NewProc("ReleaseDC")
	procPrintWindow            = user32.NewProc("PrintWindow")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
)

const (
	pwRenderFullContent = 2
	dibRGBColors        = 0
	createNoWindow      = 0x08000000
)

type rect struct{ Left, Top, Right, Bottom int32 }

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// captureWindow renders hwnd into an RGBA image.
func captureWindow(hwnd uintptr) (*image.RGBA, error) {
	var r rect
	if ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return nil, domain.Fail(domain.CodeTargetUnsupported, "window bounds")
	}
	width, height := int(r.Right-r.Left), int(r.Bottom-r.Top)
	if width <= 0 || height <= 0 || width > 16384 || height > 16384 {
		return nil, domain.Fail(domain.CodeTargetUnsupported, "window size")
	}
	window, _, _ := procGetWindowDC.Call(hwnd)
	if window == 0 {
		return nil, domain.Fail(domain.CodeTargetUnsupported, "window dc")
	}
	defer procReleaseDC.Call(hwnd, window)
	memory, _, _ := procCreateCompatibleDC.Call(window)
	defer procDeleteDC.Call(memory)
	bitmap, _, _ := procCreateCompatibleBitmap.Call(window, uintptr(width), uintptr(height))
	if bitmap == 0 {
		return nil, domain.Fail(domain.CodeTargetUnsupported, "bitmap")
	}
	defer procDeleteObject.Call(bitmap)
	old, _, _ := procSelectObject.Call(memory, bitmap)
	defer procSelectObject.Call(memory, old)
	if ok, _, _ := procPrintWindow.Call(hwnd, memory, pwRenderFullContent); ok == 0 {
		return nil, domain.Fail(domain.CodeTargetUnsupported, "print window")
	}
	// A negative height asks for top-down rows of 32-bit BGRA.
	header := bitmapInfoHeader{Width: int32(width), Height: -int32(height), Planes: 1, BitCount: 32}
	header.Size = uint32(unsafe.Sizeof(header))
	pixels := make([]byte, width*height*4)
	if n, _, _ := procGetDIBits.Call(memory, bitmap, 0, uintptr(height), uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&header)), dibRGBColors); n == 0 {
		return nil, domain.Fail(domain.CodeTargetUnsupported, "bitmap bits")
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for i := 0; i < len(pixels); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = pixels[i+2], pixels[i+1], pixels[i], 255
	}
	return img, nil
}

// ocrScript recognizes the image at $env:ARION_OCR_IMAGE with the user's
// profile languages and prints one line of text per recognized line.
const ocrScript = `$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Runtime.WindowsRuntime
$null = [Windows.Storage.StorageFile, Windows.Storage, ContentType = WindowsRuntime]
$null = [Windows.Media.Ocr.OcrEngine, Windows.Foundation, ContentType = WindowsRuntime]
$null = [Windows.Graphics.Imaging.BitmapDecoder, Windows.Graphics, ContentType = WindowsRuntime]
$asTask = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation` + "`" + `1' })[0]
function Await($op, $type) { $t = $asTask.MakeGenericMethod($type).Invoke($null, @($op)); $t.Wait(-1) | Out-Null; $t.Result }
$file = Await ([Windows.Storage.StorageFile]::GetFileFromPathAsync($env:ARION_OCR_IMAGE)) ([Windows.Storage.StorageFile])
$stream = Await ($file.OpenAsync([Windows.Storage.FileAccessMode]::Read)) ([Windows.Storage.Streams.IRandomAccessStream])
$decoder = Await ([Windows.Graphics.Imaging.BitmapDecoder]::CreateAsync($stream)) ([Windows.Graphics.Imaging.BitmapDecoder])
$bitmap = Await ($decoder.GetSoftwareBitmapAsync()) ([Windows.Graphics.Imaging.SoftwareBitmap])
$engine = [Windows.Media.Ocr.OcrEngine]::TryCreateFromUserProfileLanguages()
$result = Await ($engine.RecognizeAsync($bitmap)) ([Windows.Media.Ocr.OcrResult])
$stream.Dispose()
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$result.Lines | ForEach-Object { $_.Text }`

// recognizeWindow captures hwnd and returns its recognized text. The image
// exists only in a private temporary file removed before returning.
func recognizeWindow(ctx context.Context, hwnd uintptr) (string, error) {
	img, err := captureWindow(hwnd)
	if err != nil {
		return "", err
	}
	file, err := os.CreateTemp("", "arion-ocr-*.png")
	if err != nil {
		return "", err
	}
	defer os.Remove(file.Name())
	if err := png.Encode(file, img); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	encoded := utf16.Encode([]rune(ocrScript))
	script := make([]byte, len(encoded)*2)
	for i, u := range encoded {
		script[2*i], script[2*i+1] = byte(u), byte(u>>8)
	}
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-EncodedCommand", base64.StdEncoding.EncodeToString(script))
	cmd.Env = append(os.Environ(), "ARION_OCR_IMAGE="+file.Name())
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", domain.Fail(domain.CodeTargetUnsupported, "windows ocr")
	}
	return strings.TrimSpace(strings.ReplaceAll(out.String(), "\r\n", "\n")), nil
}

// screenTimeout bounds capture and recognition together.
const screenTimeout = 3 * time.Second
