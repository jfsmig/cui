// Copyright (c) 2022 Jean-Francois Smigielski
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package monitor

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"log"
	"strings"
)

var (
	heightQuery = 1
	heightError = 1
	widthList   = 40
)

type MonitoredItem interface {
	PrimaryKey() string
	GetKeys() []string
	GetValue(k string) string
}

type Monitorable interface {
	FetchAll(query string) ([]MonitoredItem, error)
}

// Monitor displays a terminal application that navigates in the data source
func Monitor(listable Monitorable) error {
	var x0, y0, x1, y1 int
	var err error

	var app monitorApp
	app.source = listable
	app.gui, err = gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicln("GUI open error: %v", err)
	}
	defer app.gui.Close()
	app.gui.Cursor = true
	app.query = "/tmp"
	app.gui.SetManagerFunc(func(_ *gocui.Gui) error { return app.layout() })

	x0, y0, x1, y1 = app.dimensionQuery()
	app.panelQuery, err = app.gui.SetView("query", x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			log.Panicln("Panel open error: %v", err)
		} else {
			app.panelQuery.Title = "Query"
			app.panelQuery.BgColor = gocui.ColorDefault
			app.panelQuery.SelBgColor = gocui.ColorYellow
			app.panelQuery.Highlight = true
			app.panelQuery.Editable = true
			app.panelQuery.Editor = gocui.DefaultEditor
		}
	} else {
		log.Panicln("WTF")
	}

	x0, y0, x1, y1 = app.dimensionError()
	app.panelError, err = app.gui.SetView("error", x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			log.Panicln("Panel open error: %v", err)
		} else {
			app.panelError.Title = "Error"
			app.panelError.BgColor = gocui.ColorDefault
			app.panelError.SelBgColor = gocui.ColorYellow
			app.panelError.Highlight = false
		}
	} else {
		log.Panicln("WTF")
	}

	x0, y0, x1, y1 = app.dimensionList()
	app.panelList, err = app.gui.SetView("list", x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			log.Panicln("Panel open error: %v", err)
		} else {
			app.panelList.Title = "Objects"
			app.panelList.BgColor = gocui.ColorDefault
			app.panelList.SelBgColor = gocui.ColorYellow
			app.panelList.Highlight = true
		}
	} else {
		log.Panicln("WTF")
	}

	x0, y0, x1, y1 = app.dimensionDetail()
	app.panelDetail, err = app.gui.SetView("detail", x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			log.Panicln("Panel open error: %v", err)
		} else {
			app.panelDetail.Title = "Detail"
			app.panelDetail.BgColor = gocui.ColorDefault
			app.panelDetail.SelBgColor = gocui.ColorDefault
			app.panelDetail.Highlight = false
		}
	} else {
		log.Panicln("WTF")
	}

	// General commands, whatever the current panel
	err = app.gui.SetKeybinding("", 'q', gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error { return app.editQuery(v) })
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error { return app.signalQuit(v) })
	if err != nil {
		log.Panicln(err)
	}

	err = app.gui.SetKeybinding(app.panelQuery.Name(), gocui.KeyEnter, gocui.ModNone,
		func(_ *gocui.Gui, _ *gocui.View) error {
			app.doQuery()
			return nil
		})
	if err != nil {
		log.Panicln(err)
	}

	// Specific bindings for the list panel
	err = app.gui.SetKeybinding(app.panelList.Name(), gocui.KeyArrowUp, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error { return app.arrowUp(v) })
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding(app.panelList.Name(), gocui.KeyArrowDown, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error { return app.arrowDown(v) })
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding(app.panelList.Name(), gocui.KeyPgup, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error { return app.pageUp(v) })
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding(app.panelList.Name(), gocui.KeyPgdn, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error { return app.pageDown(v) })
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding(app.panelList.Name(), gocui.KeyEnter, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error { return app.chooseObject() })
	if err != nil {
		log.Panicln(err)
	}

	app.gui.SetCurrentView(app.panelQuery.Name())

	if err := app.gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}

	return nil
}

type monitorApp struct {
	gui *gocui.Gui

	source  Monitorable
	query   string
	err     error
	items   []MonitoredItem
	current MonitoredItem

	panelQuery  *gocui.View
	panelError  *gocui.View
	panelList   *gocui.View
	panelDetail *gocui.View
}

func (app *monitorApp) dimensionQuery() (x0, y0, x1, y1 int) {
	maxX, _ := app.gui.Size()
	return 0, 0, maxX - 1, heightQuery + 1
}

func (app *monitorApp) dimensionError() (x0, y0, x1, y1 int) {
	maxX, _ := app.gui.Size()
	return 0, heightQuery + 2, maxX - 1, heightQuery + 2 + heightError + 1
}

func (app *monitorApp) dimensionList() (x0, y0, x1, y1 int) {
	_, maxY := app.gui.Size()
	return 0, heightQuery + 2 + heightError + 2, widthList, maxY - 1
}

func (app *monitorApp) dimensionDetail() (x0, y0, x1, y1 int) {
	maxX, maxY := app.gui.Size()
	return widthList + 1, heightQuery + 2 + heightError + 2, maxX - 1, maxY - 1
}

func (app *monitorApp) layoutQuery() error {
	x0, y0, x1, y1 := app.dimensionQuery()

	_, err := app.gui.SetView(app.panelQuery.Name(), x0, y0, x1, y1)
	if err != nil {
		log.Panicln(err)
	}

	return nil
}

func (app *monitorApp) layoutError() error {
	x0, y0, x1, y1 := app.dimensionError()

	v, err := app.gui.SetView(app.panelError.Name(), x0, y0, x1, y1)
	if err != nil {
		log.Panicln(err)
	}

	v.Clear()
	if app.err != nil {
		fmt.Fprint(v, app.err.Error())
	}
	return nil
}

func (app *monitorApp) layoutList() error {
	x0, y0, x1, y1 := app.dimensionList()

	v, err := app.gui.SetView(app.panelList.Name(), x0, y0, x1, y1)
	if err != nil {
		log.Panicln(err)
	}

	v.Clear()
	for _, item := range app.items {
		fmt.Fprintf(v, "%v\n", item.PrimaryKey())
	}
	return nil
}

func (app *monitorApp) layoutDetail() error {
	x0, y0, x1, y1 := app.dimensionDetail()

	v, err := app.gui.SetView(app.panelDetail.Name(), x0, y0, x1, y1)
	if err != nil {
		log.Panicln(err)
	}

	v.Clear()
	if app.current != nil {
		fmt.Fprintf(v, "%v", app.current)
	}
	return nil
}

func (app *monitorApp) layout() error {
	if err := app.layoutQuery(); err != nil {
		return err
	}
	if err := app.layoutError(); err != nil {
		return err
	}
	if err := app.layoutList(); err != nil {
		return err
	}
	if err := app.layoutDetail(); err == nil {
		return err
	}
	return nil
}

func (app *monitorApp) signalQuit(v *gocui.View) error {
	return gocui.ErrQuit
}

func (app *monitorApp) arrowUp(v *gocui.View) error {
	v.MoveCursor(0, -1, false)
	return nil
}

func (app *monitorApp) arrowDown(v *gocui.View) error {
	v.MoveCursor(0, 1, false)
	return nil
}

func (app *monitorApp) shiftByOnePage(v *gocui.View, nb int) error {
	count := len(app.items)
	_, vy := v.Size()
	ox0, oy0 := v.Origin()
	cx0, cy0 := v.Cursor()
	// final positions of the cursor (c*) and the origin (o*)
	cx, cy, ox, oy := 0, 0, 0, 0

	// Compute the current index (x,y) then shift it by a page
	x0, y0 := ox0+cx0, oy0+cy0
	x, y := x0, y0+(nb*vy)
	if y > count {
		y = count
	} else if y < 0 {
		y = 0
	}

	if y < vy/2 {
		// head
		cx, cy = cx0, y
		ox, oy = ox0, 0
	} else if y > count-vy/2 {
		// tail
		cx, cy = cx0, vy-(count-y)-1
		ox, oy = ox0, count-vy
	} else {
		// body
		cx, cy = cx0, vy/2
		ox, oy = ox0, y-vy/2
	}

	if err := v.SetCursor(cx, cy); err != nil {
		log.Panicf("PAGE count=%d pos=(%d,%d)..(%d,%d) cursor=(%d,%d)..(%d,%d) origin=(%d,%d)..(%d,%d): %v",
			count, x0, y0, x, y, cx0, cy0, cx, cy, ox0, oy0, ox, oy, err)
	}
	if err := v.SetOrigin(ox, oy); err != nil {
		log.Panicf("PAGE count=%d pos=(%d,%d)..(%d,%d) cursor=(%d,%d)..(%d,%d) origin=(%d,%d)..(%d,%d): %v",
			count, x0, y0, x, y, cx0, cy0, cx, cy, ox0, oy0, ox, oy, err)
	}

	//log.Printf("PAGE count=%d pos=(%d,%d)..(%d,%d) cursor=(%d,%d)..(%d,%d) origin=(%d,%d)..(%d,%d)",
	//		count, x0, y0, x, y, cx0, cy0, cx, cy, ox0, oy0, ox, oy)
	return nil
}

func (app *monitorApp) pageUp(v *gocui.View) error { return app.shiftByOnePage(v, -1) }

func (app *monitorApp) pageDown(v *gocui.View) error { return app.shiftByOnePage(v, 1) }

func (app *monitorApp) editQuery(v *gocui.View) error {
	app.choosePanel(app.panelQuery)
	return nil
}

func (app *monitorApp) choosePanel(panel *gocui.View) {
	app.gui.SetCurrentView(panel.Name())
	app.panelQuery.BgColor = gocui.ColorDefault
	app.panelError.BgColor = gocui.ColorDefault
	app.panelList.BgColor = gocui.ColorDefault
	app.panelDetail.BgColor = gocui.ColorDefault
	panel.BgColor = gocui.ColorCyan
}

func (app *monitorApp) doQuery() {
	app.query = app.panelQuery.ViewBuffer()
	app.query = strings.Trim(app.query, "  \r\n\t")
	items, err := app.source.FetchAll(app.query)
	if err != nil {
		app.err = err
	} else {
		app.err = nil
		app.items = items
	}
	app.choosePanel(app.panelList)
}

func (app *monitorApp) chooseObject() error {
	_, cy := app.panelList.Cursor()
	_, oy := app.panelList.Origin()
	index := cy + oy
	if index < len(app.items) {
		app.current = app.items[index]
	} else {
		app.current = nil
	}
	return nil
}
