package main

import (
	"context"
	"os"
	"fmt"
	"log"
	"strings"

	"github.com/go-shiori/go-readability"
	"github.com/gocolly/colly"
	"github.com/jackc/pgx/v5"
)
func initDatabase(db *pgx.Conn) error {
    query := `
    CREATE TABLE IF NOT EXISTS scraped_pages (
        id SERIAL PRIMARY KEY,
        url TEXT UNIQUE NOT NULL,
        raw_content TEXT,
        summary TEXT,
        is_summarized BOOLEAN DEFAULT FALSE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );`
    
    _, err := db.Exec(context.Background(), query)
    return err
}
func main() {
	// --- DATABASE SETUP ---
	ctx := context.Background()
	connStr := os.Getenv("DB_URL")
	if connStr == "" {
        connStr = "postgres://postgres:abc@123@localhost:5433/web-scraper"
    }
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(ctx) // Closes connection when main() finishes
	// 1. Define a list of URLs to scrape
	// ... after connecting to the DB ...
	err = initDatabase(conn)
	if err != nil {
	log.Fatalf("Could not create tables: %v", err)
	}
	log.Println("✅ Database tables are ready!")
	urls := []string{
		"https://www.livemint.com/news/world/what-to-know-about-ras-laffan-industrial-city-how-irans-missile-attack-on-key-lng-hub-may-cripple-india-11773901984857.html",
		"https://www.aljazeera.com/news/2026/3/23/un-expert-says-world-has-given-israel-licence-to-torture-palestinians"}

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
		fmt.Println("CONTENT PREVIEW:", article.TextContent)
		query := `
			INSERT INTO scraped_pages (url, raw_content, is_summarized) 
			VALUES ($1, $2, $3)
			ON CONFLICT (url) DO NOTHING;`

		_, err = conn.Exec(ctx, query,
			r.Request.URL.String(), // $1
			article.TextContent,    // $2
			false,                  // $3 (Default state)
		)

		if err != nil {
			fmt.Printf("Database error: %v\n", err)
		} else {
			fmt.Printf("Successfully saved: %s\n", article.Title)
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		fmt.Printf("Error on %s: %v\n", r.Request.URL, err)
	})

	// 4. Start the loop
	for _, u := range urls {
		c.Visit(u)
	}
}
