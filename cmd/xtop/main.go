package main

import (
	"context"
	"fmt"
	"log"

	"github.com/xconnio/xconn-go"
	"github.com/xconnio/xtop"
)

func main() {
	session, err := xconn.ConnectAnonymous(context.Background(), "ws://localhost:8080/ws", "realm1")
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	screen := xtop.NewScreen()
	defer screen.Fini()

	screen.PrintText(1, 1, "Hello, xtop ! (Press Esc or Ctrl-C to quit)")
	screen.PrintText(1, 2, fmt.Sprintf("Session ID: %d", session.ID()))
	screen.Show()

	screen.RunUntilExit()
}
