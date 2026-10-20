// arion-ime-probe invokes the production registration adapter for authorized
// native acceptance. Installation owns its temporary transport and performs the
// same selection-preserving, build-verified handover as the desktop app.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/macos"
)

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	op := flag.String("op", "query", "query, install, activate or remove")
	bundle := flag.String("bundle", "", "bundled input source used by install")
	socket := flag.String("socket", macos.DefaultIMESocket(), "listener socket; the host must use the matching profile")
	timeout := flag.Duration("timeout", 30*time.Second, "activation deadline")
	cancelAfter := flag.Duration("cancel-after", 0, "cancel activation after this interval")
	flag.Parse()
	var err error
	switch *op {
	case "query":
	case "install":
		if *bundle == "" {
			err = domain.Fail(domain.CodeTargetUnsupported, "input bundle path required")
		} else {
			transport := macos.NewIME(*socket, nil)
			err = transport.Listen()
			if err == nil {
				err = macos.InstallIME(*bundle, transport.WaitReady)
			}
			transport.Close()
		}
	case "activate":
		transport := macos.NewIME(*socket, nil)
		err = transport.Listen()
		if err == nil {
			parent, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			ctx, cancel := context.WithTimeout(parent, *timeout)
			var timer *time.Timer
			if *cancelAfter > 0 {
				timer = time.AfterFunc(*cancelAfter, cancel)
			}
			err = macos.ActivateIME(ctx, transport.WaitReady)
			if timer != nil {
				timer.Stop()
			}
			cancel()
			stop()
		}
		transport.Close()
	case "remove":
		err = macos.RemoveIME()
	default:
		err = domain.Fail(domain.CodeTargetUnsupported, "unknown input source operation")
	}
	result := map[string]any{"state": macos.QueryIME()}
	if err != nil {
		result["errorCode"] = domain.CodeOf(err)
		result["detail"] = err.Error()
	}
	if encode := json.NewEncoder(os.Stdout).Encode(result); encode != nil || err != nil {
		os.Exit(1)
	}
}
