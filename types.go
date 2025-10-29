package xtop

import "github.com/rivo/tview"

//nolint:gochecknoglobals
var app *tview.Application

func SetApp(a *tview.Application) { app = a }

const (
	StatusIdle    = "Idle"
	StatusRunning = "Running"
	StatusOffline = "Offline"
)

type SessionInfo struct {
	AuthID     string `json:"authid"`
	AuthRole   string `json:"authrole"`
	Serializer string `json:"serializer"`
	SessionID  uint64 `json:"sessionID"`
}
