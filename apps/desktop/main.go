package main

import (
	"log"
	"os"

	"arion.local/desktop/bootstrap"
	"arion.local/desktop/infrastructure/diagnostics"
)

func main() {
	// Environment overrides exist for isolated E2E profiles only.
	app, err := bootstrap.New(bootstrap.Options{
		DataDir:        os.Getenv("ARION_DATA_DIR"),
		E2EControl:     os.Getenv("ARION_E2E_CONTROL"),
		Offline:        os.Getenv("ARION_OFFLINE") == "1",
		SkipOnboarding: os.Getenv("ARION_E2E_CONTROL") != "",
	})
	if err != nil {
		log.Fatalf("application failed code=%s", diagnostics.Failure(err))
	}
	if err := app.Run(); err != nil {
		log.Fatalf("application failed code=%s", diagnostics.Failure(err))
	}
}
