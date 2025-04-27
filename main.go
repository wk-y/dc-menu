package main

import (
	"fmt"
	"image/color"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"gioui.org/app"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

func main() {
	go func() {
		if err := run(); err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}

var dcs = [...]string{"segundo", "cuarto", "tercero", "latitude"}

type DcTab struct {
	sync.RWMutex
	name        string
	dc          string
	section     int
	err         error
	menu        Menu
	scroller    layout.List
	button      widget.Clickable
	sectionTabs []widget.Clickable
}

func (d *DcTab) load(today Date) {
	d.Lock()
	defer d.Unlock()

	log.Printf("Fetching menu [%s]\n", d.dc)
	cached, err := loadFromCache(d.dc)
	if err != nil {
		log.Println("Cache: ", err)
	} else {
		if cached.Daily[today] != nil {
			log.Printf("Using cached menu [%s]\n", d.dc)
			d.menu = cached
			d.err = nil
			return
		}
		log.Printf("Ignoring stale cache [%s]\n", d.dc)
	}

	d.menu, d.err = fetchMenu(dcUrl(d.dc))
	if d.err != nil {
		log.Println(err)
		return
	}

	err = saveToCache(d.dc, d.menu)
	if err != nil {
		log.Printf("Error saving cache [%s]: %v", d.dc, err)
	}
}

func (d *DcTab) layout(th *material.Theme, gtx layout.Context, today Date) layout.Dimensions {
	locked := d.TryRLock()
	if !locked {
		gtx.Execute(op.InvalidateCmd{})
		return material.Body1(th, "Loading...").Layout(gtx)
	}
	defer d.RUnlock()

	if d.err != nil {
		return material.Body1(th, fmt.Sprintf("Error: %v", d.err)).Layout(gtx)
	}

	currentDay := d.menu.Daily[today]
	if currentDay == nil {
		return material.Body1(th, "Today's menu was not fetched.").Layout(gtx)
	}

	// resize tabs
	for range len(currentDay.Sections) - len(d.sectionTabs) {
		d.sectionTabs = append(d.sectionTabs, widget.Clickable{})
	}
	d.sectionTabs = d.sectionTabs[:len(currentDay.Sections)]

	// update section based on tab clicks
	for i := range d.sectionTabs {
		if d.sectionTabs[i].Clicked(gtx) {
			d.section = i
			d.scroller = layout.List{}
		}
	}
	if d.section > len(currentDay.Sections) {
		d.section = 0
	}

	// layout for tabs
	tabButtons := make([]layout.FlexChild, len(currentDay.Sections))
	for i, section := range currentDay.Sections {
		theme := th
		if i != d.section {
			theme = inactiveTh
		}
		tabButtons[i] = layout.Flexed(1, material.Button(theme, &d.sectionTabs[i], section.Name).Layout)
	}

	d.scroller.Axis = layout.Vertical
	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, tabButtons...)
		}),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			return d.scroller.Layout(gtx, len(currentDay.Sections), func(gtx layout.Context, index int) layout.Dimensions {
				elems := []layout.FlexChild{}

				section := currentDay.Sections[d.section]
				elems = append(elems, layout.Rigid(material.H3(th, section.Name).Layout))

				for _, station := range section.Stations {
					elems = append(elems, layout.Rigid(material.H4(th, station.Name).Layout))

					for _, menuItem := range station.Menu {
						elems = append(elems, layout.Rigid(material.Body1(th, menuItem.Name).Layout))
					}
				}

				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, elems...)
			})
		}),
	)
}

var th, inactiveTh *material.Theme

func init() {
	th = material.NewTheme()
	inactiveTh = material.NewTheme()
	inactiveTh.Palette.ContrastBg = color.NRGBA{128, 128, 128, 255}
}

func run() error {
	tz, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Println(err)
		tz = time.UTC // todo: fallback to PST
	}
	today := dateFromTime(time.Now().In(tz))
	log.Printf("Today is %s\n", today)

	var window app.Window
	tabs := make([]DcTab, len(dcs))

	for i, dc := range dcs {
		i := i
		dc := dc

		go func() {
			tabs[i] = DcTab{
				name: strings.ToTitle(dc),
				dc:   dc,
			}

			tabs[i].load(today)
			window.Invalidate()
		}()
	}

	activeTab := &tabs[0]
	tabButtons := make([]layout.FlexChild, len(tabs))
	for i := range tabs {
		i := i
		tabButtons[i] = layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			theme := inactiveTh
			if &tabs[i] == activeTab {
				theme = th
			}

			return material.Button(theme, &tabs[i].button, tabs[i].name).Layout(gtx)
		})
	}

	var ops op.Ops

	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			for i := range tabs {
				if tabs[i].button.Clicked(gtx) {
					activeTab = &tabs[i]
					activeTab.scroller.Position = layout.Position{} // reset scroll
				}
			}

			layout.Flex{Axis: layout.Vertical}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, tabButtons...)
				}),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return activeTab.layout(th, gtx, today)
				}),
			)

			e.Frame(gtx.Ops)
		}
	}
}

func dcUrl(name string) string {
	return "https://housing.ucdavis.edu/dining/menus/dining-commons/" + name
}
