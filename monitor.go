// Copyright (c) 2022 Jean-Francois Smigielski
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cui

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"log"
	"regexp"
	"sort"
	"strings"
	"text/tabwriter"
)

const (
	heightQuery  = 1
	heightFilter = 1
	widthList    = 20
	widthError   = 50
)

const (
	panelNameQuery  = "query"
	panelNameFilter = "filter"
	panelNameError  = "error"
	panelNameList   = "list"
	panelNameDetail = "detail"
)

const (
	modeDetail detailMode = iota
	modeTable
)

// MonitoredItem describe the expectation for any monitorable item: just a set of metadata tha can be queried
// independently.
type MonitoredItem interface {
	// GetPrimaryKey returns the defauly key that will be used by the application if none is explicitly selected.
	// The returned primary key should be an element of the array of all the possible keys returned by GetKeys.
	GetPrimaryKey() string
	// GetKeys Return all the keys of the metadata properties.
	// Subsequent calls should return the same array: same entries, same order.
	// The array should contain the value returned by GetPrimaryKey.
	GetKeys() []string
	// GetValue returns the value of a single given key. Any error results in an empty output.
	GetValue(k string) string
	// GetDetail generates a description of the object, that will be displayed in the "detail" panel when not in
	// "table" mode. The MonitoredItem implementation MAY return a dump of the metadata (i.e the keys and their
	// respective values), the application doesn't care.
	GetDetail() string
}

// Monitorable implements a source of items.
// The application is responsible to call for a new list of objects. The list is cached in the application between
// two fetches.
type Monitorable interface {
	// FetchAll returns the whole list of items. They will be saved in the CLI app. until the next FetchAll call.
	FetchAll(query string) ([]MonitoredItem, error)
}

type detailMode int

type monitorApp struct {
	gui *gocui.Gui

	// Displays the main query string that configures the source of objects
	panelQuery *gocui.View
	// A coma-separated list of fnmatch patterns to restrict the fields displayed in the list panel
	panelFilter *gocui.View
	// A minor panel to display the last error encountered.
	panelError *gocui.View
	// The main panel displaying the selected key of all the objects fetched using the query string.
	// Moving the cursor along the list selects a new object on each movement, and update the details
	// Panel.
	panelList *gocui.View
	// A panel displaying either the full detail of the selected object, or a table of the selected field.
	panelDetail *gocui.View

	source Monitorable

	mode  detailMode
	query string
	err   error

	items      []MonitoredItem
	currentKey string

	possibleKeys sort.StringSlice
}

// Monitor displays a terminal application that navigates in the data source
func Monitor(listable Monitorable, firstQuery string) error {
	var err error

	app := monitorApp{
		source: listable,
		mode:   modeDetail,
		query:  firstQuery,
	}

	app.gui, err = gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Panicf("GUI open error: %v", err)
	}
	defer app.gui.Close()
	app.gui.Cursor = true
	app.gui.SetManagerFunc(func(_ *gocui.Gui) error { return app.layout() })

	app.createPanels()
	app.bindKeys()
	app.doQuery()
	app.choosePanel(app.panelQuery)

	if err := app.gui.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}

	return nil
}

func (app *monitorApp) createPanels() {
	var x0, y0, x1, y1 int
	var err error

	x0, y0, x1, y1 = app.dimensionQuery()
	app.panelQuery, err = app.gui.SetView(panelNameQuery, x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			log.Panicf("Panel open error: %v", err)
		} else {
			app.panelQuery.Title = "Query"
			app.panelQuery.BgColor = gocui.ColorDefault
			app.panelQuery.SelBgColor = gocui.ColorYellow
			app.panelQuery.Highlight = true
			app.panelQuery.Editable = true
			app.panelQuery.Editor = gocui.DefaultEditor
			if _, err = fmt.Fprint(app.panelQuery, app.query); err != nil {
				log.Panicln("The grail is in the castle or Aaaaaaaarrr...")
			}
		}
	} else {
		log.Panicln("WTF")
	}

	x0, y0, x1, y1 = app.dimensionFilter()
	app.panelFilter, err = app.gui.SetView(panelNameFilter, x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			log.Panicf("Panel open error: %v", err)
		} else {
			app.panelFilter.Title = "Filter"
			app.panelFilter.BgColor = gocui.ColorDefault
			app.panelFilter.SelBgColor = gocui.ColorYellow
			app.panelFilter.Highlight = true
			app.panelFilter.Editable = true
			app.panelFilter.Editor = gocui.DefaultEditor
			fmt.Fprint(app.panelFilter, ".*")
		}
	} else {
		log.Panicln("WTF")
	}

	x0, y0, x1, y1 = app.dimensionError()
	app.panelError, err = app.gui.SetView(panelNameError, x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			log.Panicf("Panel open error: %v", err)
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
	app.panelList, err = app.gui.SetView(panelNameList, x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			log.Panicf("Panel open error: %v", err)
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
	app.panelDetail, err = app.gui.SetView(panelNameDetail, x0, y0, x1, y1)
	if err != nil {
		if err != gocui.ErrUnknownView {
			log.Panicf("Panel open error: %v", err)
		} else {
			app.panelDetail.Title = "Detail"
			app.panelDetail.BgColor = gocui.ColorDefault
			app.panelDetail.SelBgColor = gocui.ColorYellow
			app.panelDetail.Highlight = false
		}
	} else {
		log.Panicln("WTF")
	}
}

func (app *monitorApp) bindKeys() {
	var err error

	// General commands, whatever the current panel
	err = app.gui.SetKeybinding("", gocui.KeyTab, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error {
			switch app.gui.CurrentView() {
			case app.panelQuery:
				app.doQuery()
				app.redrawList()
				app.redrawTable()
				app.choosePanel(app.panelFilter)
			case app.panelFilter:
				app.redrawTable()
				app.choosePanel(app.panelList)
			case app.panelList:
				app.choosePanel(app.panelQuery)
			}
			return nil
		})
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding("", 'm', gocui.ModAlt,
		func(_ *gocui.Gui, v *gocui.View) error {
			switch app.mode {
			case modeTable:
				app.mode = modeDetail
				app.panelDetail.Highlight = false
			case modeDetail:
				app.mode = modeTable
				app.panelDetail.Highlight = true
				app.redrawTable()
			}
			app.redrawDetail()
			return nil
		})
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error { return app.signalQuit() })
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding("", gocui.KeyEnter, gocui.ModNone,
		func(_ *gocui.Gui, _ *gocui.View) error {
			app.doQuery()
			app.redrawList()
			app.redrawTable()
			return nil
		})
	if err != nil {
		log.Panicln(err)
	}

	// Specific bindings for the list panel
	err = app.gui.SetKeybinding(app.panelList.Name(), gocui.KeyArrowUp, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error {
			app.panelList.MoveCursor(0, -1, false)
			app.redrawDetail()
			return nil
		})
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding(app.panelList.Name(), gocui.KeyArrowDown, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error {
			app.panelList.MoveCursor(0, 1, false)
			app.redrawDetail()
			return nil
		})
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding(app.panelList.Name(), gocui.KeyPgup, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error {
			app.shiftByNbPages(app.panelList, -1)
			app.redrawDetail()
			return nil
		})
	if err != nil {
		log.Panicln(err)
	}
	err = app.gui.SetKeybinding(app.panelList.Name(), gocui.KeyPgdn, gocui.ModNone,
		func(_ *gocui.Gui, v *gocui.View) error {
			app.shiftByNbPages(app.panelList, 1)
			app.redrawDetail()
			return nil
		})
	if err != nil {
		log.Panicln(err)
	}
}

func (app *monitorApp) dimensionQuery() (x0, y0, x1, y1 int) {
	maxX, _ := app.gui.Size()
	return 0, 0, maxX - widthError - 2, heightQuery + 1
}

func (app *monitorApp) dimensionFilter() (x0, y0, x1, y1 int) {
	maxX, _ := app.gui.Size()
	return 0, heightQuery + 2, maxX - widthError - 2, heightQuery + 2 + heightFilter + 1
}

func (app *monitorApp) dimensionError() (x0, y0, x1, y1 int) {
	maxX, _ := app.gui.Size()
	return maxX - widthError - 1, 0, maxX - 1, heightQuery + 2 + heightFilter + 1
}

func (app *monitorApp) dimensionList() (x0, y0, x1, y1 int) {
	_, maxY := app.gui.Size()
	return 0, heightQuery + 2 + heightFilter + 2, widthList, maxY - 1
}

func (app *monitorApp) dimensionDetail() (x0, y0, x1, y1 int) {
	maxX, maxY := app.gui.Size()
	return widthList + 1, heightQuery + 2 + heightFilter + 2, maxX - 1, maxY - 1
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
	_, err := app.gui.SetView(app.panelList.Name(), x0, y0, x1, y1)
	if err != nil {
		log.Panicln(err)
	}
	return nil
}

func (app *monitorApp) layoutDetail() error {
	x0, y0, x1, y1 := app.dimensionDetail()
	_, err := app.gui.SetView(app.panelDetail.Name(), x0, y0, x1, y1)
	if err != nil {
		log.Panicln(err)
	}
	app.alignTableOnList()
	return nil
}

func (app *monitorApp) layout() error {
	if err := app.layoutQuery(); err != nil {
		log.Panicf("query layout: %v", err)
		return err
	}
	if err := app.layoutError(); err != nil {
		log.Panicf("error layout: %v", err)
		return err
	}
	if err := app.layoutList(); err != nil {
		log.Panicf("list layout: %v", err)
		return err
	}
	if err := app.layoutDetail(); err != nil {
		log.Panicf("detail layout: %v", err)
		return err
	}
	return nil
}

func (app *monitorApp) signalQuit() error {
	return gocui.ErrQuit
}

func (app *monitorApp) shiftByNbPages(v *gocui.View, nb int) {
	count := len(v.BufferLines())
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
}

func (app *monitorApp) choosePanel(panel *gocui.View) {
	if v, err := app.gui.SetCurrentView(panel.Name()); err != nil {
		log.Panicf("Unknown panel %s: %v", panel.Name(), err)
	} else if v != panel {
		log.Panicf("Unexpected panel %s", panel.Name())
	}
	app.panelQuery.BgColor = gocui.ColorDefault
	app.panelError.BgColor = gocui.ColorDefault
	app.panelList.BgColor = gocui.ColorDefault
	app.panelDetail.BgColor = gocui.ColorDefault
	panel.BgColor = gocui.ColorCyan
}

func (app *monitorApp) getKeyName(i int) string {
	if app.currentKey != "" {
		return app.currentKey
	} else {
		return app.items[i].GetPrimaryKey()
	}
}

func (app *monitorApp) getKeyValue(i int) string { return app.items[i].GetValue(app.getKeyName(i)) }

func (app *monitorApp) doQuery() {
	app.query = app.panelQuery.Buffer()
	app.query = strings.Trim(app.query, "  \r\n\t")
	items, err := app.source.FetchAll(app.query)
	if err != nil {
		app.err = err
		app.items = []MonitoredItem{}
	} else {
		app.err = nil
		app.items = items
	}

	// Extract the possible keys
	possibleKeys := make(map[string]bool)
	for _, item := range app.items {
		possibleKeys[item.GetPrimaryKey()] = true
		for _, k := range item.GetKeys() {
			possibleKeys[k] = true
		}
	}
	app.possibleKeys = make([]string, 0)
	for k, _ := range possibleKeys {
		app.possibleKeys = append(app.possibleKeys, k)
	}
	app.possibleKeys.Sort()

	// Sort the item according to the selected key
	less := func(idx0, idx1 int) bool { return 0 > strings.Compare(app.getKeyValue(idx0), app.getKeyValue(idx1)) }
	sort.Slice(app.items, less)
}

func (app *monitorApp) redrawDetail() {
	_, cy := app.panelList.Cursor()
	_, oy := app.panelList.Origin()
	index := cy + oy

	var current MonitoredItem
	if index < len(app.items) {
		current = app.items[index]
	}

	if app.mode == modeDetail {
		app.panelDetail.Clear()
		if current != nil {
			fmt.Fprintf(app.panelDetail, "%v", current.GetDetail())
		}
	}
}

func (app *monitorApp) redrawList() {
	app.panelList.Clear()
	app.panelDetail.Clear()
	separator := ""
	for _, item := range app.items {
		k := app.currentKey
		if k == "" {
			k = item.GetPrimaryKey()
		}
		fmt.Fprintf(app.panelList, "%s%v", separator, item.GetValue(k))
		separator = "\n"
	}

	if app.err == nil {
		app.choosePanel(app.panelList)
		app.panelList.SetOrigin(0, 0)
		app.panelList.SetCursor(0, 0)
	}
}

func (app *monitorApp) redrawTable() {
	if app.mode != modeTable {
		return
	}
	app.panelDetail.Clear()
	w := tabwriter.NewWriter(app.panelDetail, 8, 1, 2, ' ', 0)

	// Prepare the key patterns
	csp := app.panelFilter.ViewBuffer()
	patterns := strings.Split(csp, ",")
	for i, p := range patterns {
		patterns[i] = strings.Trim(p, " \t\n\r")
	}
	matchers := make([]*regexp.Regexp, 0)
	for _, pattern := range patterns {
		r, _ := regexp.Compile(pattern)
		matchers = append(matchers, r)
	}
	matches := func(k string) bool {
		for _, m := range matchers {
			if m.MatchString(k) {
				return true
			}
		}
		return false
	}

	crlf := ""
	for i, item := range app.items {
		fmt.Fprint(w, crlf)
		crlf = "\n"

		currentKey := app.getKeyName(i)
		separator := ""
		for _, k := range item.GetKeys() {
			if k == currentKey || !matches(k) {
				continue
			}
			fmt.Fprintf(w, "%s%s", separator, item.GetValue(k))
			separator = "\t"
		}
	}
	w.Flush()
}

func (app *monitorApp) alignTableOnList() {
	if app.mode != modeTable {
		return
	}
	_, oy := app.panelList.Origin()
	_, cy := app.panelList.Cursor()
	if err := app.panelDetail.SetOrigin(0, oy); err != nil {
		log.Panicf("reset table origin: %v", err)
	}
	if err := app.panelDetail.SetCursor(0, cy); err != nil {
		log.Panicf("reset table cursor: %v", err)
	}
}
