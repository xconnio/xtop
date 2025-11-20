package main

import (
	"context"
	"log"

	"github.com/jessevdk/go-flags"

	"github.com/xconnio/xconn-go"
	"github.com/xconnio/xtop"
)

type Options struct {
	URL   string `short:"u" long:"url" description:"WAMP router URL" env:"XTOP_URL" default:"ws://localhost:8080/ws"`
	Realm string `short:"r" long:"realm" description:"WAMP realm to connect" env:"XTOP_REALM" default:"io.xconn.mgmt"`
}

func main() {
	var opts Options
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	session, err := xconn.ConnectAnonymous(context.Background(), opts.URL, opts.Realm)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	if err := xtop.SetupLogger(xtop.LogPath); err != nil {
		log.Fatalf("failed to setup logger: %v", err)
	}

	screensMgr := xtop.NewScreenManager(session)
	if err = screensMgr.Run(); err != nil {
		log.Fatalf("app run failed: %v", err)
	}
}
