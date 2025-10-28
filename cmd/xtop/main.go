package main

import (
	"context"
	"log"

	"github.com/rivo/tview"

	"github.com/xconnio/xconn-go"
	"github.com/xconnio/xtop"
)

func main() {
	app := tview.NewApplication()
	xtop.SetApp(app)

	session, err := xconn.ConnectAnonymous(context.Background(), "ws://localhost:8080/ws", "io.xconn.mgmt")
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	screen := xtop.NewXTopScreen(session)
	if err := app.SetRoot(screen, true).EnableMouse(true).Run(); err != nil {
		log.Fatalf("app run failed: %v", err)
	}
}
