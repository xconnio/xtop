package xtop

import (
	"fmt"
	"math"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/xconnio/xconn-go"
)

//nolint:gochecknoglobals
var (
	app *tview.Application
)

func SetApp(a *tview.Application) { app = a }

func showRealmSessions(table *tview.Table, session *xconn.Session, realm string) {
	table.Clear()

	headers := []string{"SESSION ID", "AUTH ID", "AUTH ROLE", "SERIALIZER"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetExpansion(1)
		table.SetCell(0, col, cell)
	}

	sessions, err := FetchSessionDetails(session, realm)
	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell("[red]Error fetching session details"))
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
		showSessionLogs(table, session, realm, selected.SessionID)
	})
}

func showSessionLogs(table *tview.Table, s *xconn.Session, realm string, sessionID uint64) {
	table.Clear()

	headers := []string{"SESSION LOGS"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false)
		table.SetCell(0, col, cell)
	}

	row := 1
	err := FetchSessionLogs(s, realm, sessionID, func(line string) {
		app.QueueUpdateDraw(func() {
			table.SetCell(row, 0, tview.NewTableCell("[white]"+line))
			row++
			if row > 2000 { // safety limit to prevent infinite growth
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
}

func showAllRealms(table *tview.Table, session *xconn.Session) {
	table.Clear()

	headers := []string{"REALMS", "CLIENTS", "MESSAGES/s", "STATUS"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h)).
			SetAlign(tview.AlignLeft).
			SetSelectable(false).
			SetExpansion(1)
		table.SetCell(0, col, cell)
	}

	realms, err := FetchRealms(session)
	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell("[red]Error fetching realms"))
		return
	}

	for row, realm := range realms {
		clients, err := FetchSessions(session, realm)
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
			showRealmSessions(table, session, realms[row-1])
		}
	})
}

func buildRouterTable(session *xconn.Session) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false)
	table.SetFixed(1, 1)
	table.SetBorder(true)
	table.SetBorderColor(tcell.ColorBlue)

	showAllRealms(table, session)

	return table
}

func setupTableInput(table *tview.Table, session *xconn.Session) {
	table.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Key() {
		case tcell.KeyEsc:
			showAllRealms(table, session)
			return nil
		case tcell.KeyRune:
			if ev.Rune() == 'q' {
				app.Stop()
				return nil
			}
		}
		return ev
	})
}

func NewXTopScreen(session *xconn.Session) tview.Primitive {
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

	table := buildRouterTable(session)
	setupTableInput(table, session)

	flex.AddItem(header, 8, 0, false)
	flex.AddItem(table, 0, 1, true)

	resp := session.Call("io.xconn.mgmt.stats.status.set").Kwarg("enable", true).Do()
	if resp.Err != nil {
		fmt.Printf("Could not enable stats: %v\n", resp.Err)
	}

	eventHandler := func(event *xconn.Event) {
		args := event.Args()
		if len(args) == 0 {
			return
		}
		statsMap, ok := args[0].(map[string]any)
		if !ok {
			return
		}
		app.QueueUpdateDraw(func() {
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
				session.ID()))
		})
	}

	subResp := session.Subscribe("io.xconn.mgmt.stats.on_update", eventHandler).Do()
	if subResp.Err != nil {
		fmt.Printf("Error subscribing to stats: %v\n", subResp.Err)
	}

	return flex
}
