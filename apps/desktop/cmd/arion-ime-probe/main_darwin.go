// arion-ime-probe invokes the production registration adapter for authorized
// native acceptance. Installation owns its temporary transport and performs the
// same selection-preserving, build-verified handover as the desktop app.
package main

import (
	"encoding/json"
	"flag"
	"os"
	"runtime"

	"arion.local/core/domain"
	"arion.local/desktop/infrastructure/macos"
)

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	op := flag.String("op", "query", "query, install or remove")
	bundle := flag.String("bundle", "", "bundled input source used by install")
	flag.Parse()
	var err error
	switch *op {
	case "query":
	case "install":
		if *bundle == "" {
			err = domain.Fail(domain.CodeTargetUnsupported, "input bundle path required")
		} else {
			transport := macos.NewIME(macos.DefaultIMESocket(), nil)
			err = transport.Listen()
			if err == nil {
				err = macos.InstallIME(*bundle, transport.WaitReady)
			}
			transport.Close()
		}
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
