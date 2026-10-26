package main

import (
	"arion.local/desktop/internal/center"
	"encoding/json"
	"log"
	"os"

	"arion.local/desktop/bootstrap"
	"arion.local/desktop/infrastructure/diagnostics"
	"github.com/egoist/mygo"
)

func main() {
	if handled, err := mygo.HandleNotificationRegistration(); handled || err != nil {
		if err != nil {
			log.Fatalf("notification registration failed code=%s", diagnostics.Failure(err))
		}
		return
	}
	if len(os.Args) == 2 && os.Args[1] == "--print-center-profile" {
		profile, err := center.Load()
		if err != nil {
			log.Fatal(err)
		}
		if err := json.NewEncoder(os.Stdout).Encode(struct {
			Profile     center.Profile `json:"profile"`
			Fingerprint string         `json:"fingerprint"`
		}{profile, profile.Fingerprint()}); err != nil {
			log.Fatal(err)
		}
		return
	}
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
