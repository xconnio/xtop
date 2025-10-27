package xtop

import (
	"log"

	"github.com/gdamore/tcell/v2"
)

type Screen struct {
	tcell.Screen
	style tcell.Style
}

func NewScreen() *Screen {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("creating screen: %v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("initializing screen: %v", err)
	}

	style := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorWhite)
	s.SetStyle(style)

	return &Screen{Screen: s, style: style}
}

func (sc *Screen) PrintText(x, y int, str string) {
	for i, r := range str {
		sc.SetContent(x+i, y, r, nil, sc.style)
	}
}

func (sc *Screen) RunUntilExit() {
	for {
		ev := sc.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				return
			}
		case *tcell.EventResize:
			sc.Sync()
		}
	}
}
