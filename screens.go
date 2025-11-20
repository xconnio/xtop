package xtop

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/xconnio/xconn-go"
)

const (
	StatusIdle    = "Idle"
	StatusRunning = "Running"
	StatusOffline = "Offline"
)

type ScreenManager struct {
	app        *tview.Application
	mgmt       *ManagementAPI
	shutdown   chan struct{}
	logChannel chan struct{}
}

func NewScreenManager(session *xconn.Session) *ScreenManager {
	app := tview.NewApplication()
	return &ScreenManager{
		app:        app,
		mgmt:       NewManagementAPI(session),
		shutdown:   make(chan struct{}),
		logChannel: make(chan struct{}, 1),
	}
}

func (s *ScreenManager) showRealmSessions(table *tview.Table, realm string) {
	table.Clear()

	headers := []string{"SESSION ID", "AUTHID", "AUTHROLE", "SERIALIZER"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h)).
			SetAlign(tview.AlignLeft).SetSelectable(false).SetExpansion(1)
		table.SetCell(0, col, cell)
	}

	sessions, err := s.mgmt.SessionDetailsByRealm(realm)
	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("[red]%s", err.Error())))
		return
	}

	for row, sd := range sessions {
		table.SetCell(row+1, 0, tview.NewTableCell(fmt.Sprintf("[white]%d", sd.SessionID)).SetExpansion(1))
		table.SetCell(row+1, 1, tview.NewTableCell("[white]"+sd.AuthID).SetExpansion(1))
		table.SetCell(row+1, 2, tview.NewTableCell("[white]"+sd.AuthRole).SetExpansion(1))
		table.SetCell(row+1, 3, tview.NewTableCell("[white]"+sd.Serializer).SetExpansion(1))
	}

	table.SetTitle(fmt.Sprintf(" [white]%s - Sessions: %d ", realm, len(sessions))).
		SetTitleColor(tcell.ColorWhite).SetTitleAlign(tview.AlignCenter)

	table.SetSelectedFunc(func(row, _ int) {
		if row == 0 || row > len(sessions) {
			return
		}
		selected := sessions[row-1]
		s.showSessionLogs(table, realm, selected.SessionID)
	})

	s.setupTableInput(table, func() { s.showAllRealms(table) })
}

func (s *ScreenManager) showSessionLogs(table *tview.Table, realm string, sessionID uint64) {
	table.Clear()
	table.SetCell(0, 0, tview.NewTableCell("[yellow::b]SESSION LOGS").
		SetAlign(tview.AlignLeft).SetSelectable(false))

	const maxRows = 2000
	active := true
	logUpdates := make(chan string, 500)

	go func() {
		row := 1
		var buffer []string
		var mu sync.Mutex
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case line, ok := <-logUpdates:
				if !ok || !active {
					return
				}
				mu.Lock()
				buffer = append(buffer, line)
				if len(buffer) > 100 {
					buffer = buffer[len(buffer)-100:]
				}
				mu.Unlock()

			case <-ticker.C:
				if !active {
					return
				}
				mu.Lock()
				lines := buffer
				buffer = nil
				mu.Unlock()
				if len(lines) == 0 {
					continue
				}
				s.app.QueueUpdateDraw(func() {
					if !active {
						return
					}
					for _, l := range lines {
						table.SetCell(row, 0, tview.NewTableCell("[white]"+l))
						row++
						if row > maxRows {
							table.RemoveRow(1)
							row = maxRows
						}
					}
					table.ScrollToEnd()
				})
			case <-s.logChannel:
				return
			}
		}
	}()

	sendLog := func(line string) {
		if !active {
			return
		}
		select {
		case logUpdates <- line:
		default:
			select {
			case <-logUpdates:
			default:
			}
			select {
			case logUpdates <- line:
			default:
			}
		}
	}

	err := s.mgmt.FetchSessionLogs(realm, sessionID, sendLog)
	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("[red]subscribe to session logs failed: %v", err)))
	}

	table.SetTitle(fmt.Sprintf(" [white]%s - Session %d Logs ", realm, sessionID)).
		SetTitleColor(tcell.ColorWhite).SetTitleAlign(tview.AlignCenter)

	s.setupTableInput(table, func() {
		s.logChannel <- struct{}{}
		active = false
		s.mgmt.StopSessionLogs()
		close(logUpdates)
		s.showRealmSessions(table, realm)
	})
}

func (s *ScreenManager) showAllRealms(table *tview.Table) {
	table.Clear()
	headers := []string{"REALMS", "CLIENTS", "MESSAGES/s", "STATUS"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h)).
			SetAlign(tview.AlignLeft).SetSelectable(false).SetExpansion(1)
		table.SetCell(0, col, cell)
	}

	realms, err := s.mgmt.Realms()
	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell("[red]Error fetching realms"))
		return
	}

	for row, realm := range realms {
		clients, err := s.mgmt.SessionsCount(realm)
		status := StatusIdle
		if clients > 0 {
			status = StatusRunning
		}
		if err != nil {
			status = StatusOffline
			clients = 0
		}

		color := map[string]string{
			StatusRunning: "[green]",
			StatusIdle:    "[yellow]",
			StatusOffline: "[red]",
		}[status]

		table.SetCell(row+1, 0, tview.NewTableCell("[white]"+realm).SetExpansion(1))
		table.SetCell(row+1, 1, tview.NewTableCell(fmt.Sprintf("[white]%d", clients)).SetExpansion(1))
		table.SetCell(row+1, 2, tview.NewTableCell("[white]0").SetExpansion(1))
		table.SetCell(row+1, 3, tview.NewTableCell(color+status).SetExpansion(1))
	}

	table.SetTitle(fmt.Sprintf(" [white]Realms [%d] ", len(realms))).
		SetTitleColor(tcell.ColorWhite).SetTitleAlign(tview.AlignCenter)

	table.SetSelectedFunc(func(row, _ int) {
		if row > 0 && row-1 < len(realms) {
			s.showRealmSessions(table, realms[row-1])
		}
	})
	s.setupTableInput(table, nil)
}

func (s *ScreenManager) buildRouterTable() *tview.Table {
	table := tview.NewTable().SetSelectable(true, false).SetFixed(1, 1)
	table.SetBorder(true).SetBorderColor(tcell.ColorBlue)
	s.showAllRealms(table)
	return table
}

func (s *ScreenManager) setupTableInput(table *tview.Table, onEsc func()) {
	table.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEsc:
			if onEsc != nil {
				onEsc()
				return nil
			}
		case tcell.KeyRune:
			if ev.Rune() == 'q' {
				s.Stop()
				return nil
			}
		case tcell.KeyCtrlC:
			s.Stop()
			return nil
		}
		return ev
	})
}

func (s *ScreenManager) Run() error {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)
	info := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	logo := tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignRight).SetText(`[cyan]
██╗  ██╗████████╗ ██████╗ ██████╗ 
╚██╗██╔╝╚══██╔══╝██╔═══██╗██╔══██╗
 ╚███╔╝    ██║   ██║   ██║██████╔╝
 ██╔██╗    ██║   ██║   ██║██╔═══╝ 
██╔╝ ██╗   ██║   ╚██████╔╝██║     
╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝     
[white]`)

	header := tview.NewFlex().
		AddItem(info, 0, 1, false).
		AddItem(logo, 0, 1, false)

	table := s.buildRouterTable()

	flex.AddItem(header, 8, 0, false)
	flex.AddItem(table, 0, 1, true)

	if err := s.mgmt.RequestStats(); err != nil {
		log.Printf("failed to request stats: %v", err)
	}

	statsUpdates := make(chan map[string]interface{}, 10)

	go func() {
		defer close(statsUpdates)

		for {
			select {
			case statsMap, ok := <-statsUpdates:
				if !ok {
					return
				}
				s.app.QueueUpdateDraw(func() {
					cpuUsage := math.Min(statsMap["cpu_usage"].(float64), 100)
					memUsage := float64(statsMap["res_memory"].(uint64)) / (1024 * 1024)
					uptime := statsMap["uptime"].(float64)
					info.SetText(fmt.Sprintf(
						"\n[white]XTOP: [yellow]v0.1.0[white]\n"+
							"[white]XConn: [yellow]v0.1.0[white]\n"+
							"[white]CPU: [yellow]%.1f%%[white]\n"+
							"[white]MEM: [yellow]%.1fMB[white]\n"+
							"[white]UPTIME: [yellow]%02d:%02d:%02d[white]\n"+
							"[white]SESSION: [yellow]%d[white]",
						cpuUsage, memUsage, int(uptime/3600),
						int(uptime)%3600/60, int(uptime)%60,
						s.mgmt.session.ID()))
				})
			case <-s.shutdown:
				return
			}
		}
	}()

	eventHandler := func(ev *xconn.Event) {
		select {
		case <-s.shutdown:
			return
		default:
		}
		statsDict, err := ev.ArgDict(0)
		if err != nil {
			return
		}
		select {
		case statsUpdates <- statsDict.Raw():
		default:
		}
	}

	subResp := s.mgmt.session.Subscribe(xconn.ManagementTopicStats, eventHandler).Do()
	if subResp.Err != nil {
		log.Printf("Error subscribing to stats: %v", subResp.Err)
	}

	err := s.app.SetRoot(flex, true).EnableMouse(true).Run()
	if s.shutdown != nil {
		s.Stop()
	}
	return err
}

func (s *ScreenManager) Stop() {
	if s.shutdown == nil {
		return
	}
	close(s.shutdown)
	close(s.logChannel)
	s.shutdown = nil
	if s.app != nil {
		s.app.Stop()
	}
	if s.mgmt != nil {
		s.mgmt.Close()
	}
}
