package main

import (
	"log"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
)

type MenuItem struct {
	Name string
}

type DailyMenu struct {
	Sections []MenuSection
}

type StationMenu struct {
	Name string
	Menu []MenuItem
}

type MenuSection struct {
	Name     string
	Stations []StationMenu
}

type Menu struct {
	Name  string
	Daily map[Date]*DailyMenu
}

func fetchMenu(url string) (result Menu, err error) {
	result.Daily = map[Date]*DailyMenu{}

	c := colly.NewCollector()

	// Extract the DC name from the page title
	c.OnHTML("title", func(h *colly.HTMLElement) {
		result.Name = strings.SplitN(h.DOM.Text(), "|", 2)[0]
		result.Name = strings.Trim(result.Name, " ")
		for _, suffixWord := range []string{"Menu", "DC"} {
			result.Name, _ = strings.CutSuffix(result.Name, suffixWord)
			result.Name = strings.Trim(result.Name, " ")
		}
	})

	c.OnHTML(".menu_maincontainer", func(h *colly.HTMLElement) {
		dateString := h.DOM.Find("h3").Text()
		log.Println("Found date ", dateString)
		date, err := time.Parse("Monday, January 2, 2006", dateString)

		if err != nil {
			log.Println(err)
			return
		}

		ymd := dateFromTime(date)
		result.Daily[ymd] = &DailyMenu{}

		// parse section (Breakfast, Lunch, Dinner)
		h.DOM.Find("h4").Each(func(i int, s *goquery.Selection) {
			var section MenuSection
			section.Name = s.Text()

			// parse station (Tomato Street Grill, etc)
			s.Parent().Find("h5").Each(func(i int, s *goquery.Selection) {
				var station StationMenu
				station.Name = s.Text()
				log.Println("Station found ", station.Name)

				// match the ul after
				s.Next().Find(".nutrition").Each(func(i int, s *goquery.Selection) {
					itemName := s.Parent().Find("span:not(.collapsible-heading-status)").Text()
					log.Println("Found a ", section.Name, " menu item", itemName)
					station.Menu = append(station.Menu, MenuItem{Name: itemName})
				})

				section.Stations = append(section.Stations, station)
			})

			result.Daily[ymd].Sections = append(result.Daily[ymd].Sections, section)
		})
	})

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
	})

	err = c.Visit(url)
	return
}
