package main

import (
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	maxPage int
)

type Scraper struct {
	db *gorm.DB
}

func main() {

	dsn := ""
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&Product{})
	s := &Scraper{db: db}
	s.start()
}

func (u *Scraper) start() {
	c := colly.NewCollector(
		colly.AllowedDomains("www.zoll-auktion.de", "zoll-auktion.de"),
	)
	productsCollector := c.Clone()
	productsCollector.Limit(&colly.LimitRule{
		DomainGlob: "*httpbin.*",
		Delay:      5 * time.Second,
	})
	r := regexp.MustCompile(`.pagination=(?P<Pagination>\d+)`)
	// Count total pages
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		match := r.FindStringSubmatch(link)
		if len(match) > 1 {
			pagination, err := strconv.Atoi(match[1])
			if err != nil {
				panic(err)
			}
			if pagination > 2 {
				maxPage = pagination
			}
		}
	})
	c.Visit("https://www.zoll-auktion.de/auktion/auktionsuebersicht.php")
	var products []Product
	productsCollector.OnHTML("article", func(e *colly.HTMLElement) {
		var product Product
		// Hole Link der Auktion
		product.Link = e.ChildAttr("div.kachel_auktion_link>a[href]", "href")
		// Hole ID der Auktion
		aID, err := strconv.Atoi(path.Base(product.Link))
		if err != nil {
			panic(err)
		}
		product.AuctionID = aID
		// Hole Name der Auktion
		product.Name = e.ChildText("div.kachel_auktion_link>a[href]")
		e.ForEach("li", func(_ int, ch *colly.HTMLElement) {
			if strings.Contains(ch.Text, "Gebot") {
				// Anzahl der Gebote
				product.Bids = ch.Text
			} else if strings.Contains(ch.Text, "noch") {
				// Zeit bis Ablauf
				product.EndTime = ch.Text
			} else if strings.Contains(ch.Text, "Abholung") {
				// Nur Abholung!
				product.OnlyPickup = true
			}
		})
		// Hole Ort der Auktion/Abholort
		product.Location = e.ChildText("ul[aria-label='Auktionsdetails']>li:first-child")
		// Hole den aktuellen Preis
		product.Price = e.ChildText("p.text-right>span.font-weight-bold")
		products = append(products, product)
	})
	productsCollector.OnScraped(func(r *colly.Response) {
		u.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "auction_id"}},
			UpdateAll: true,
		}).Create(&products)
		products = nil
	})
	productsCollector.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL.String())
	})

	//productsCollector.Visit("https://www.zoll-auktion.de/auktion/auktionsuebersicht.php?n0=search&t=t1&s=12&pagination=1")
	for i := 1; i < (maxPage + 1); i++ {
		productsCollector.Visit(fmt.Sprintf("https://www.zoll-auktion.de/auktion/auktionsuebersicht.php?n0=search&t=t1&s=12&pagination=%d", i))
	}
}
