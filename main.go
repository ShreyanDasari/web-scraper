package main

import (
	"fmt"
	"strings"

	"github.com/go-shiori/go-readability"
	"github.com/gocolly/colly"
)

func main() {
	// 1. Define a list of URLs to scrape
	urls := []string{
		"https://www.livemint.com/news/world/what-to-know-about-ras-laffan-industrial-city-how-irans-missile-attack-on-key-lng-hub-may-cripple-india-11773901984857.html",
		"https://www.aljazeera.com/economy/2026/3/23/iran-war-whats-happening-on-day-24-of-us-israel-attacks",
	}

	c := colly.NewCollector()

	// Set User-Agent to avoid being blocked
	c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

	// 2. Safety: Don't hammer the server too fast
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*livemint.com*",
		Parallelism: 2,
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("\n--- Processing: %s ---\n", r.Request.URL)

		// 3. Use the response's internal Reader and URL snapshot
		reader := strings.NewReader(string(r.Body))

		// We pass r.Request.URL here so readability knows exactly which page it's on
		article, err := readability.FromReader(reader, r.Request.URL)
		if err != nil {
			fmt.Printf("Failed to parse %s: %v\n", r.Request.URL, err)
			return
		}

		fmt.Println("TITLE:", article.Title)
		fmt.Println("CONTENT PREVIEW:", article.TextContent[:200], "...")
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error on %s: %v\n", r.Request.URL, err)
	})

	// 4. Start the loop
	for _, u := range urls {
		c.Visit(u)
	}
}
