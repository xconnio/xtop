package xtop

import (
	"fmt"
	"log"
	"math"

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
	app  *tview.Application
	mgmt *ManagementAPI
}

func NewScreenManager(session *xconn.Session) *ScreenManager {
	app := tview.NewApplication()
	return &ScreenManager{
		app:  app,
		mgmt: NewManagementAPI(session),
	}
}

func (s *ScreenManager) showRealmSessions(table *tview.Table, realm string) {
	table.Clear()

	headers := []string{"SESSION ID", "AUTHID", "AUTHROLE", "SERIALIZER"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetExpansion(1)
		table.SetCell(0, col, cell)
	}

	sessions, err := s.mgmt.SessionDetailsByRealm(realm)
	if err != nil {
		errMsg := fmt.Sprintf("[red]Failed to fetch session details: %s", err.Error())
		table.SetCell(1, 0, tview.NewTableCell(errMsg))
		return
	}

	for row, s := range sessions {
		table.SetCell(row+1, 0, tview.NewTableCell(fmt.Sprintf("[white]%d", s.SessionID)).SetExpansion(1))
		table.SetCell(row+1, 1, tview.NewTableCell("[white]"+s.AuthID).SetExpansion(1))
		table.SetCell(row+1, 2, tview.NewTableCell("[white]"+s.AuthRole).SetExpansion(1))
		table.SetCell(row+1, 3, tview.NewTableCell("[white]"+s.Serializer).SetExpansion(1))
	}

	table.SetTitle(fmt.Sprintf(" [white]%s - Sessions: %d ", realm, len(sessions))).
		SetTitleColor(tcell.ColorWhite).
		SetTitleAlign(tview.AlignCenter)
	table.SetSelectedFunc(func(row, col int) {
		if row == 0 || row > len(sessions) {
			return
		}

		selected := sessions[row-1]
		s.showSessionLogs(table, realm, selected.SessionID)
	})

	s.setupTableInput(table, func() {
		s.showAllRealms(table)
	})
}

func (s *ScreenManager) showSessionLogs(table *tview.Table, realm string, sessionID uint64) {
	table.Clear()

	headers := []string{"SESSION LOGS"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		table.SetCell(0, col, cell)
	}

	row := 1
	err := s.mgmt.FetchSessionLogs(realm, sessionID, func(line string) {
		s.app.QueueUpdateDraw(func() {
			table.SetCell(row, 0, tview.NewTableCell("[white]"+line))
			row++
			if row > 2000 {
				table.RemoveRow(1)
				row--
			}
		})
	})

	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell(fmt.Sprintf("[red]%v", err)))
	}

	table.SetTitle(fmt.Sprintf(" [white]%s - Session %d Logs ", realm, sessionID)).
		SetTitleColor(tcell.ColorWhite).
		SetTitleAlign(tview.AlignCenter)

	s.setupTableInput(table, func() {
		var sub xconn.SubscribeResponse
		s.mgmt.Lock()
		sub = s.mgmt.subscription
		s.mgmt.Unlock()

		_ = sub.Unsubscribe()

		s.showRealmSessions(table, realm)
	})
}

func (s *ScreenManager) showAllRealms(table *tview.Table) {
	table.Clear()

	headers := []string{"REALMS", "CLIENTS", "MESSAGES/s", "STATUS"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetExpansion(1)
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

		statusColor := "[white]"
		switch status {
		case StatusRunning:
			statusColor = "[green]"
		case StatusIdle:
			statusColor = "[yellow]"
		case StatusOffline:
			statusColor = "[red]"
		}

		table.SetCell(row+1, 0, tview.NewTableCell("[white]"+realm).SetExpansion(1))
		table.SetCell(row+1, 1, tview.NewTableCell(fmt.Sprintf("[white]%d", clients)).SetExpansion(1))
		table.SetCell(row+1, 2, tview.NewTableCell("[white]0").SetExpansion(1))
		table.SetCell(row+1, 3, tview.NewTableCell(statusColor+status).SetExpansion(1))
	}

	table.SetTitle(fmt.Sprintf(" [white]Realms [%d] ", len(realms))).
		SetTitleColor(tcell.ColorWhite).
		SetTitleAlign(tview.AlignCenter)

	table.SetSelectedFunc(func(row, col int) {
		if row > 0 && row-1 < len(realms) {
			s.showRealmSessions(table, realms[row-1])
		}
	})

	s.setupTableInput(table, func() {
	})
}

func (s *ScreenManager) buildRouterTable() *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false)
	table.SetFixed(1, 1)
	table.SetBorder(true)
	table.SetBorderColor(tcell.ColorBlue)

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
				s.app.Stop()
				return nil
			}
		}
		return ev
	})
}

func (s *ScreenManager) Run() error {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	info := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	logo := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight).
		SetText(`[cyan]
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
		log.Fatalln("failed to request stats", err)
	}

	eventHandler := func(event *xconn.Event) {
		statsDict, err := event.ArgDict(0)
		if err != nil {
			fmt.Printf("Could not get stats: %v\n", err)
			return
		}
		statsMap := statsDict.Raw()

		s.app.QueueUpdateDraw(func() {
			info.SetText(fmt.Sprintf(
				"\n[white]XTOP: [yellow]v0.1.0[white]\n"+
					"[white]XConn: [yellow]v0.1.0[white]\n"+
					"[white]CPU: [yellow]%.1f%%[white]\n"+
					"[white]MEM: [yellow]%.1fMB[white]\n"+
					"[white]UPTIME: [yellow]%02d:%02d:%02d[white]\n"+
					"[white]SESSION: [yellow]%d[white]",
				math.Min(statsMap["cpu_usage"].(float64), 100),
				float64(statsMap["res_memory"].(uint64))/(1024*1024), int(statsMap["uptime"].(float64)/3600),
				int(statsMap["uptime"].(float64))%3600/60,
				int(statsMap["uptime"].(float64))%60,
				s.mgmt.session.ID()))
		})
	}

	subResp := s.mgmt.session.Subscribe(xconn.ManagementTopicStats, eventHandler).Do()
	if subResp.Err != nil {
		fmt.Printf("Error subscribing to stats: %v", subResp.Err)
	}

	return s.app.SetRoot(flex, true).EnableMouse(true).Run()
}
