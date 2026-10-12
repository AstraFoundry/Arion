package main

import (
	"log"

	"arion.local/desktop/bootstrap"
	"github.com/egoist/mygo"
)

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(bootstrap.WindowOptions())
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
