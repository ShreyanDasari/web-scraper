package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-shiori/go-readability"
	"github.com/gocolly/colly"
)

func main() {
	targetURL := "https://www.livemint.com/news/world/what-to-know-about-ras-laffan-industrial-city-how-irans-missile-attack-on-key-lng-hub-may-cripple-india-11773901984857.html?_gl=1*1diokof*_up*MQ..*_gs*MQ..&gclid=Cj0KCQjwve7NBhC-ARIsALZy9HUEb-6XnBcpWzZXhw7cO5Igi5b8qS_7uYp-_LDTgCD0dT8Pbu2znt4aAl--EALw_wcB&gbraid=0AAAAADadt_7Omx1qyPavzc5l8eDoweQzu" // Your URL here

	c := colly.NewCollector()

	// Add User-Agent to avoid being blocked
	c.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

	c.OnResponse(func(r *colly.Response) {
		fmt.Printf("Response received with status code: %d\n", r.StatusCode)

		// Convert the colly response body into a reader
		reader := strings.NewReader(string(r.Body))

		parsedURL, err := url.Parse(targetURL)
		if err != nil {
			fmt.Printf("Failed to parse URL: %v\n", err)
			return
		}

		// This is the "Smart" part: it analyzes the HTML automatically
		article, err := readability.FromReader(reader, parsedURL)
		if err != nil {
			fmt.Printf("Failed to parse article: %v\n", err)
			return
		}

		fmt.Println("TITLE:", article.Title)
		fmt.Println("\nTEXT CONTENT:\n", article.TextContent)
	})

	c.OnError(func(_ *colly.Response, err error) {
		fmt.Printf("Error during visit: %v\n", err)
	})

	fmt.Println("Starting to visit URL...")
	c.Visit(targetURL)
	fmt.Println("Visit completed.")
}
