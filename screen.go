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
			"[white]SESSION ID: [yellow]%d[white]", session.ID()))

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

func buildRouterTable(session *xconn.Session) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetBorder(true)
	table.SetBorderColor(tcell.ColorBlue)

	headers := []string{"REALM", "CLIENT", "MESSAGES/s", "STATUS"}
	for col, h := range headers {
		cell := tview.NewTableCell(fmt.Sprintf("[yellow::b]%s", h))
		cell.SetAlign(tview.AlignLeft)
		cell.SetExpansion(1)
		table.SetCell(0, col, cell)
	}

	realms, err := FetchRealms(session)
	if err != nil {
		table.SetCell(1, 0, tview.NewTableCell("[red]Error fetching realms"))
		return table
	}

	table.SetTitle(fmt.Sprintf(" [white]Routers(all)[%d] ", len(realms)))
	table.SetTitleColor(tcell.ColorWhite)
	table.SetTitleAlign(tview.AlignCenter)

	for row, realm := range realms {
		table.SetCell(row+1, 0, tview.NewTableCell("[white]"+realm).SetExpansion(1))
		table.SetCell(row+1, 1, tview.NewTableCell("[white]0").SetExpansion(1))
		table.SetCell(row+1, 2, tview.NewTableCell("[white]0").SetExpansion(1))
		table.SetCell(row+1, 3, tview.NewTableCell("[yellow]"+StatusIdle).SetExpansion(1))
	}

	return table
}

func setupTableInput(table *tview.Table) {
	table.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyEsc || ev.Rune() == 'q' {
			app.Stop()
		}
		return ev
	})
}

func NewXTopScreen(session *xconn.Session) tview.Primitive {
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	header := buildHeader(session)
	table := buildRouterTable(session)
	setupTableInput(table)

	flex.AddItem(header, 8, 0, false)
	flex.AddItem(table, 0, 1, true)

	return flex
}
