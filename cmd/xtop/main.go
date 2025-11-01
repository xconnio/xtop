package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/xconnio/xconn-go"
	"github.com/xconnio/xtop"
)

type Options struct {
	Host  string `short:"H" long:"host" description:"WAMP router hostname or IP" default:"localhost"`
	Port  string `short:"p" long:"port" description:"WAMP router port" default:"8080"`
	Realm string `short:"r" long:"realm" description:"WAMP realm to connect to" default:"io.xconn.mgmt"`
}

func main() {
	var opts Options
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Fatalf("failed to parse flags: %v", err)
	}

	url := fmt.Sprintf("ws://%s:%s/ws", opts.Host, opts.Port)
	if !(strings.HasPrefix(url, "ws://") || strings.HasPrefix(url, "wss://")) {
		log.Fatalf("invalid URL: %s (must start with ws:// or wss://)", url)
	}

	session, err := xconn.ConnectAnonymous(context.Background(), url, opts.Realm)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}

	screensMgr := xtop.NewScreenManager(session)
	if err = screensMgr.Run(); err != nil {
		log.Fatalf("app run failed: %v", err)
	}
}
