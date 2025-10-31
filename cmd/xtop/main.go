package main

import (
	"context"
	"log"

	"github.com/xconnio/xconn-go"
	"github.com/xconnio/xtop"
)

func main() {
	session, err := xconn.ConnectAnonymous(context.Background(), "ws://localhost:8080/ws", "io.xconn.mgmt")
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	screensMgr := xtop.NewScreenManager(session)
	if err = screensMgr.Run(); err != nil {
		log.Fatalf("app run failed: %v", err)
	}
}
