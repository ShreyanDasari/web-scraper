package main

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly"
)

func main() {
	c := colly.NewCollector(
		// Use a real browser user-agent to reduce the chances of being blocked.
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36"),
	)

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("Received response: status=%d, bytes=%d\n", r.StatusCode, len(r.Body))
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Request error: %v (status %d)\n", err, r.StatusCode)
	})

	c.OnHTML(".storyContent", func(e *colly.HTMLElement) {
		var paragraphs []string

		// Loop through all <p> tags inside the div
		e.ForEach("p", func(_ int, el *colly.HTMLElement) {
			text := el.Text
			paragraphs = append(paragraphs, text)
		})

		// Join all paragraphs into one article
		article := strings.Join(paragraphs, "\n\n")

		fmt.Println("Full Article:\n", article)
	})

	c.Visit("https://www.livemint.com/news/world/what-to-know-about-ras-laffan-industrial-city-how-irans-missile-attack-on-key-lng-hub-may-cripple-india-11773901984857.html?_gl=1*1diokof*_up*MQ..*_gs*MQ..&gclid=Cj0KCQjwve7NBhC-ARIsALZy9HUEb-6XnBcpWzZXhw7cO5Igi5b8qS_7uYp-_LDTgCD0dT8Pbu2znt4aAl--EALw_wcB&gbraid=0AAAAADadt_7Omx1qyPavzc5l8eDoweQzu")
}
