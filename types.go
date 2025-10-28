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
