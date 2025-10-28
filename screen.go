package xtop

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/xconnio/xconn-go"
)

func buildHeader(session *xconn.Session) tview.Primitive {
	header := tview.NewFlex()

	info := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	info.SetText(fmt.Sprintf(
		"\n[white]XTOP: [yellow]v0.1.0[white]\n"+
			"[white]XConn: [yellow]v1.0.0[white]\n"+
			"[white]CPU: [yellow]9%%[white]\n"+
			"[white]MEM: [yellow]26%%[white]\n"+
			"[white]UPTIME: [yellow][white]\n"+
			"[white]SESSION: [yellow]%d[white]", session.ID()))

	logo := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignRight)
	logo.SetText(`[cyan]
██╗  ██╗████████╗ ██████╗ ██████╗ 
╚██╗██╔╝╚══██╔══╝██╔═══██╗██╔══██╗
 ╚███╔╝    ██║   ██║   ██║██████╔╝
 ██╔██╗    ██║   ██║   ██║██╔═══╝ 
██╔╝ ██╗   ██║   ╚██████╔╝██║     
╚═╝  ╚═╝   ╚═╝    ╚═════╝ ╚═╝     
[white]`)

	header.AddItem(info, 0, 1, false)
	header.AddItem(logo, 0, 1, false)
	return header
}

func showRealmSessions(table *tview.Table, session *xconn.Session, realm string) {
	for row := table.GetRowCount() - 1; row >= 1; row-- {
		table.RemoveRow(row)
	}

	sessionHeaders := []string{"SESSION ID", "AUTH ID", "AUTH ROLE", "SERIALIZER"}
	for col, h := range sessionHeaders {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h))
		cell.SetAlign(tview.AlignLeft)
		cell.SetSelectable(false)
		cell.SetExpansion(1)
		table.SetCell(0, col, cell)
	}

	sessions, err := FetchSessionDetails(session, realm)
	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell("[red]Error fetching session details"))
		return
	}

	for row, session := range sessions {
		table.SetCell(row+1, 0, tview.NewTableCell(fmt.Sprintf("[white]%d", session.SessionID)).SetExpansion(1))
		table.SetCell(row+1, 1, tview.NewTableCell("[white]"+session.AuthID).SetExpansion(1))
		table.SetCell(row+1, 2, tview.NewTableCell("[white]"+session.AuthRole).SetExpansion(1))
		table.SetCell(row+1, 3, tview.NewTableCell("[white]"+session.Serializer).SetExpansion(1))
	}

	table.SetTitle(fmt.Sprintf(" [white]%s - Sessions: %d ", realm, len(sessions)))
	table.SetTitleColor(tcell.ColorWhite)
	table.SetTitleAlign(tview.AlignCenter)
}

func showAllRealms(table *tview.Table, session *xconn.Session) {
	for row := table.GetRowCount() - 1; row >= 1; row-- {
		table.RemoveRow(row)
	}

	realmHeaders := []string{"REALMS", "CLIENTS", "MESSAGES/s", "STATUS"}
	for col, h := range realmHeaders {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h))
		cell.SetAlign(tview.AlignLeft)
		cell.SetSelectable(false)
		cell.SetExpansion(1)
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

	table.SetTitle(fmt.Sprintf(" [white]Realms [%d] ", len(realms)))
	table.SetTitleColor(tcell.ColorWhite)
	table.SetTitleAlign(tview.AlignCenter)
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
		case tcell.KeyEnter:
			row, _ := table.GetSelection()
			if row > 0 {
				realms, err := FetchRealms(session)
				if err == nil && row-1 < len(realms) {
					realm := realms[row-1]
					showRealmSessions(table, session, realm)
				}
			}
			return nil
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

	header := buildHeader(session)
	table := buildRouterTable(session)
	setupTableInput(table, session)

	flex.AddItem(header, 8, 0, false)
	flex.AddItem(table, 0, 1, true)

	return flex
}
